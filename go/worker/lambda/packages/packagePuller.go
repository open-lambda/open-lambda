package packages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/embedded"
	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// PackagePuller is the interface for installing pip packages locally.
// The manager installs to the worker host from an optional pip
// mirror.
type PackagePuller struct {
	sbPool    sandbox.SandboxPool
	depTracer *DepTracer

	// directory of lambda code that installs pip packages
	pipLambda string

	packages sync.Map
}

type Package struct {
	Name         string
	Meta         PackageMeta
	installMutex sync.Mutex
	installed    uint32
}

// the pip-install admin lambda returns this
type PackageMeta struct {
	Deps     []string `json:"Deps"`
	TopLevel []string `json:"TopLevel"`
}

// NewPackagePuller creates a new PackagePuller instance and initializes it with the given sandbox pool and dependency tracer.
func NewPackagePuller(sbPool sandbox.SandboxPool, depTracer *DepTracer) (*PackagePuller, error) {
	// create a lambda function for installing pip packages.  We do
	// each install in a Sandbox for two reasons:
	//
	// 1. packages may be malicious
	// 2. we want to install the right version, matching the Python
	//    in the Sandbox
	pipLambda := filepath.Join(common.Conf.Worker_dir, "admin-lambdas", "pip-install")
	if err := os.MkdirAll(pipLambda, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(pipLambda, "f.py")
	code := []byte(embedded.PackagePullerInstaller_py)
	if err := ioutil.WriteFile(path, code, 0600); err != nil {
		return nil, err
	}

	installer := &PackagePuller{
		sbPool:    sbPool,
		depTracer: depTracer,
		pipLambda: pipLambda,
	}

	return installer, nil
}

// From PEP-426: "All comparisons of distribution names MUST
// be case insensitive, and MUST consider hyphens and
// underscores to be equivalent."
func NormalizePkg(pkg string) string {
	return strings.ReplaceAll(strings.ToLower(pkg), "_", "-")
}

// InstallRecursive installs the specified packages and their dependencies recursively.
func (pp *PackagePuller) InstallRecursive(installs []string) ([]string, error) {
	// shrink capacity to length so that our appends are not
	// visible to caller
	installs = installs[:len(installs):len(installs)]

	installSet := make(map[string]bool)
	for _, install := range installs {
		name := strings.Split(install, "==")[0]
		installSet[name] = true
	}

	// Installs may grow as we loop, because some installs have
	// deps, leading to other installs
	for i := 0; i < len(installs); i++ {
		pkg := installs[i]
		if common.Conf.Trace.Package {
			slog.Info(fmt.Sprintf("On %v of %v", pkg, installs))
		}
		p, err := pp.GetPkg(pkg)
		if err != nil {
			return nil, err
		}

		if common.Conf.Trace.Package {
			slog.Info(fmt.Sprintf("Package '%s' has deps %v", pkg, p.Meta.Deps))
			slog.Info(fmt.Sprintf("Package '%s' has top-level modules %v", pkg, p.Meta.TopLevel))
		}

		// push any previously unseen deps on the list of ones to install
		for _, dep := range p.Meta.Deps {
			if !installSet[dep] {
				installs = append(installs, dep)
				installSet[dep] = true
			}
		}
	}

	return installs, nil
}

// GetPkg retrieves the specified package, installing it if necessary.
func (pp *PackagePuller) GetPkg(pkg string) (*Package, error) {
	// get (or create) package
	pkg = NormalizePkg(pkg)
	tmp, _ := pp.packages.LoadOrStore(pkg, &Package{Name: pkg})
	p := tmp.(*Package)

	// fast path
	if atomic.LoadUint32(&p.installed) == 1 {
		return p, nil
	}

	// slow path
	p.installMutex.Lock()
	defer p.installMutex.Unlock()
	if p.installed == 0 {
		if err := pp.sandboxInstall(p); err != nil {
			return p, err
		}

		atomic.StoreUint32(&p.installed, 1)
		pp.depTracer.TracePackage(p)
		return p, nil
	}

	return p, nil
}

// sandboxInstall installs the specified package within a new Sandbox.
func (pp *PackagePuller) sandboxInstall(p *Package) (err error) {
	t := common.T0("pull-package")
	defer t.T1()

	// the pip-install lambda installs to /host, which is the the
	// same as scratchDir, which is the same as a sub-directory
	// named after the package in the packages dir
	scratchDir := filepath.Join(common.Conf.Pkgs_dir, p.Name)
	slog.Info(fmt.Sprintf("do pip install, using scratchDir='%v'", scratchDir))

	alreadyInstalled := false
	if _, err := os.Stat(scratchDir); err == nil {
		// assume dir existence means it is installed already
		slog.Info(fmt.Sprintf("%s appears already installed from previous run of OL", p.Name))
		alreadyInstalled = true
	} else {
		slog.Info(fmt.Sprintf("run pip install %s from a new Sandbox to %s on host", p.Name, scratchDir))
		if err := os.Mkdir(scratchDir, 0700); err != nil {
			return err
		}
	}

	defer func() {
		if err != nil {
			os.RemoveAll(scratchDir)
		}
	}()

	// Use installer profile and inherit any zeros from worker defaults
	inst := common.Conf.InstallerLimits.WithDefaults(&common.Conf.Limits)

	meta := &sandbox.SandboxMeta{
		MemLimitMB: inst.Mem_mb,
	}
	sb, err := pp.sbPool.Create(nil, true, pp.pipLambda, scratchDir, meta, common.RT_PYTHON)
	if err != nil {
		return err
	}
	defer sb.Destroy("package installation complete")

	// we still need to run a Sandbox to parse the dependencies, even if it is already installed
	msg := fmt.Sprintf(`{"pkg": "%s", "alreadyInstalled": %v}`, p.Name, alreadyInstalled)
	reqBody := bytes.NewReader([]byte(msg))

	// the URL doesn't matter, since it is local anyway
	req, err := http.NewRequest("POST", "http://container/run/pip-install", reqBody)
	if err != nil {
		return err
	}
	resp, err := sb.Client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("install lambda returned status %d, Body: '%s', Installer Sandbox State: %s",
			resp.StatusCode, string(body), sb.DebugString())
	}

	if err := json.Unmarshal(body, &p.Meta); err != nil {
		return err
	}

	for i, pkg := range p.Meta.Deps {
		p.Meta.Deps[i] = NormalizePkg(pkg)
	}

	return nil
}
