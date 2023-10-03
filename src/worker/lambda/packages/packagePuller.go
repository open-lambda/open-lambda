package packages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/embedded"
	"github.com/open-lambda/open-lambda/ol/worker/sandbox"
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
	Deps     []string `json:"Deps"` // deprecated
	TopLevel []string `json:"TopLevel"`
}

type ModuleInfo struct {
	Name  string
	IsPkg bool
}

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

// GetPkg does the pip install in a Sandbox, taking care to never install the
// same Sandbox more than once.
//
// the fast/slow path code is tweaked from the sync.Once code, the
// difference being that may try the installed more than once, but we
// will never try more after the first success
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

// sandboxInstall does the pip install within a new Sandbox, to a directory mapped from
// the host.  We want the package on the host to share with all, but
// want to run the install in the Sandbox because we don't trust it.
func (pp *PackagePuller) sandboxInstall(p *Package) (err error) {
	t := common.T0("pull-package")
	defer t.T1()

	// the pip-install lambda installs to /host, which is the the
	// same as scratchDir, which is the same as a sub-directory
	// named after the package in the packages dir
	scratchDir := filepath.Join(common.Conf.Pkgs_dir, p.Name)
	log.Printf("do pip install, using scratchDir='%v'", scratchDir)

	alreadyInstalled := false
	if _, err := os.Stat(scratchDir); err == nil {
		// assume dir existence means it is installed already
		// TODO: but still tell sandbox to do the pip-install again? we could fetch deps info from metadata directly or
		//       don't even need deps info. add return statement here to skip the following code
		log.Printf("%s appears already installed from previous run of OL", p.Name)
		alreadyInstalled = true
		return nil
	} else {
		log.Printf("run pip install %s from a new Sandbox to %s on host", p.Name, scratchDir)
		if err := os.Mkdir(scratchDir, 0700); err != nil {
			return err
		}
	}

	defer func() {
		if err != nil {
			os.RemoveAll(scratchDir)
		}
	}()

	meta := &sandbox.SandboxMeta{
		MemLimitMB: common.Conf.Limits.Installer_mem_mb,
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

	//for i, pkg := range p.Meta.Deps {
	//	p.Meta.Deps[i] = NormalizePkg(pkg)
	//}

	return nil
}

// IterModules is a simplified implementation of pkgutil.iterModules
// todo: implement every details in pkgutil.iterModules, or find a efficient way to call pkgutil.iterModules in python
func IterModules(path string) ([]ModuleInfo, error) {
	var modules []ModuleInfo

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			// Check if the directory contains an __init__.py file, which would make it a package.
			if _, err := os.Stat(filepath.Join(path, file.Name(), "__init__.py")); !os.IsNotExist(err) {
				modules = append(modules, ModuleInfo{Name: file.Name(), IsPkg: true})
			}
		} else if strings.HasSuffix(file.Name(), ".py") && file.Name() != "__init__.py" {
			modName := strings.TrimSuffix(file.Name(), ".py")
			modules = append(modules, ModuleInfo{Name: modName, IsPkg: false})
		}
	}
	return modules, nil
}
