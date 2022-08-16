package lambda

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
	"github.com/open-lambda/open-lambda/ol/sandbox"
)

// we invoke this lambda to do the pip install in a Sandbox.
//
// the install is not recursive (it does not install deps), but it
// does parse and return a list of deps, based on a rough
// approximation of the PEP 508 format.  We ignore the "extra" marker
// and version numbers (assuming the latest).
const installLambda = `
#!/usr/bin/env python
import os, sys, platform, re

def format_full_version(info):
    version = '{0.major}.{0.minor}.{0.micro}'.format(info)
    kind = info.releaselevel
    if kind != 'final':
        version += kind[0] + str(info.serial)
    return version

# as specified here: https://www.python.org/dev/peps/pep-0508/#environment-markers
os_name = os.name
sys_platform = sys.platform
platform_machine = platform.machine()
platform_python_implementation = platform.python_implementation()
platform_release = platform.release()
platform_system = platform.system()
platform_version = platform.version()
python_version = platform.python_version()[:3]
python_full_version = platform.python_version()
implementation_name = sys.implementation.name
if hasattr(sys, 'implementation'):
    implementation_version = format_full_version(sys.implementation.version)
else:
    implementation_version = "0"
extra = '' # TODO: support extras

def matches(markers):
    return eval(markers)

def top(dirname):
    path = None
    for name in os.listdir(dirname):
        if name.endswith('-info'):
            path = os.path.join(dirname, name, "top_level.txt")
    if path == None or not os.path.exists(path):
        return []
    with open(path) as f:
        return f.read().strip().split("\n")

def deps(dirname):
    path = None
    for name in os.listdir(dirname):
        if name.endswith('-info'):
            path = os.path.join(dirname, name, "METADATA")
    if path == None or not os.path.exists(path):
        return []

    rv = set()
    with open(path, encoding='utf-8') as f:
        for line in f:
            prefix = 'Requires-Dist: '
            if line.startswith(prefix):
                line = line[len(prefix):].strip()
                parts = line.split(';')
                if len(parts) > 1:
                    match = matches(parts[1])
                else:
                    match = True
                if match:
                    name = re.split(' \(', parts[0])[0]
                    rv.add(name)
    return list(rv)

def f(event):
    pkg = event["pkg"]
    alreadyInstalled = event["alreadyInstalled"]
    if not alreadyInstalled:
        rc = os.system('pip3 install --no-deps %s --cache-dir /tmp/.cache -t /host/files' % pkg)
        print('pip install returned code %d' % rc)
        assert(rc == 0)
    name = pkg.split("==")[0]
    d = deps("/host/files")
    t = top("/host/files")
    return {"Deps":d, "TopLevel":t}
`

/*
 * PackagePuller is the interface for installing pip packages locally.
 * The manager installs to the worker host from an optional pip mirror.
 */
type PackagePuller struct {
	sbPool    sandbox.SandboxPool
	depTracer *DepTracer

	// directory of lambda code that installs pip packages
	pipLambda string

	packages sync.Map
}

type Package struct {
	name         string
	meta         PackageMeta
	installMutex sync.Mutex
	installed    uint32
}

// the pip-install admin lambda returns this
type PackageMeta struct {
	Deps     []string `json:"Deps"`
	TopLevel []string `json:"TopLevel"`
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
	code := []byte(installLambda)
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
func normalizePkg(pkg string) string {
	return strings.ReplaceAll(strings.ToLower(pkg), "_", "-")
}

// "pip install" missing packages to Conf.Pkgs_dir
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
			log.Printf("On %v of %v", pkg, installs)
		}
		p, err := pp.GetPkg(pkg)
		if err != nil {
			return nil, err
		}

		if common.Conf.Trace.Package {
			log.Printf("Package '%s' has deps %v", pkg, p.meta.Deps)
			log.Printf("Package '%s' has top-level modules %v", pkg, p.meta.TopLevel)
		}

		// push any previously unseen deps on the list of ones to install
		for _, dep := range p.meta.Deps {
			if !installSet[dep] {
				installs = append(installs, dep)
				installSet[dep] = true
			}
		}
	}

	return installs, nil
}

// GetPkg does the pip install in a Sandbox, taking care to never install the
// same Sandbox more than once.
//
// the fast/slow path code is tweaked from the sync.Once code, the
// difference being that may try the installed more than once, but we
// will never try more after the first success
func (pp *PackagePuller) GetPkg(pkg string) (*Package, error) {
	// get (or create) package
	pkg = normalizePkg(pkg)
	tmp, _ := pp.packages.LoadOrStore(pkg, &Package{name: pkg})
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
	scratchDir := filepath.Join(common.Conf.Pkgs_dir, p.name)
	log.Printf("do pip install, using scratchDir='%v'", scratchDir)

	alreadyInstalled := false
	if _, err := os.Stat(scratchDir); err == nil {
		// assume dir existence means it is installed already
		log.Printf("%s appears already installed from previous run of OL", p.name)
		alreadyInstalled = true
	} else {
		log.Printf("run pip install %s from a new Sandbox to %s on host", p.name, scratchDir)
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
	msg := fmt.Sprintf(`{"pkg": "%s", "alreadyInstalled": %v}`, p.name, alreadyInstalled)
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

	if err := json.Unmarshal(body, &p.meta); err != nil {
		return err
	}

	for i, pkg := range p.meta.Deps {
		p.meta.Deps[i] = normalizePkg(pkg)
	}

	return nil
}
