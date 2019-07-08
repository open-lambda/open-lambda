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
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/sandbox"
	"github.com/open-lambda/open-lambda/ol/stats"
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
        rc = os.system('pip3 install --no-deps %s -t /host' % pkg)
        print('pip install returned code %d' % rc)
        assert(rc == 0)
    name = pkg.split("==")[0]
    d = deps("/host")
    t = top("/host")
    return {"Deps":d, "TopLevel":t}
`

/*
 * PackagePuller is the interface for installing pip packages locally.
 * The manager installs to the worker host from an optional pip mirror.
 */
type PackagePuller struct {
	sbPool sandbox.SandboxPool

	// directory of lambda code that installs pip packages
	pipLambda string

	modules sync.Map
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

func NewPackagePuller(sbPool sandbox.SandboxPool) (*PackagePuller, error) {
	// create a lambda function for installing pip modules.  We do
	// each install in a Sandbox for two reasons:
	//
	// 1. packages may be malicious
	// 2. we want to install the right version, matching the Python
	//    in the Sandbox
	pipLambda := filepath.Join(config.Conf.Worker_dir, "admin-lambdas", "pip-install")
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
		pipLambda: pipLambda,
	}

	return installer, nil
}

func (mp *PackagePuller) GetPkg(pkg string) *Package {
	mod, _ := mp.modules.LoadOrStore(pkg, &Package{name: pkg})
	return mod.(*Package)
}

// do the pip install within a new Sandbox, to a directory mapped from
// the host.  We want the package on the host to share with all, but
// want to run the install in the Sandbox because we don't trust it.
func (mp *PackagePuller) sandboxInstall(p *Package) (err error) {
	// the pip-install lambda installs to /host, which is the the
	// same as scratchDir, which is the same as a sub-directory
	// named after the package in the packages dir
	scratchDir := filepath.Join(config.Conf.Pkgs_dir, p.name)

	alreadyInstalled := false
	if _, err := os.Stat(scratchDir); err == nil {
		// assume dir exististence means it is installed already
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

	t := stats.T0("pip-install")
	defer t.T1()

	meta := &sandbox.SandboxMeta{
		MemLimitMB: config.Conf.Limits.Installer_mem_mb,
	}
	sb, err := mp.sbPool.Create(nil, true, mp.pipLambda, scratchDir, meta)
	if err != nil {
		return err
	}
	defer sb.Destroy()

	proxy, err := sb.HttpProxy()
	if err != nil {
		return err
	}

	// the URL doesn't matter, since it is local anyway
	msg := fmt.Sprintf(`{"pkg": "%s", "alreadyInstalled": %v}`, p.name, alreadyInstalled)
	reqBody := bytes.NewReader([]byte(msg))
	req, err := http.NewRequest("POST", "http://container/run/pip-install", reqBody)
	if err != nil {
		return err
	}
	resp, err := proxy.Transport.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// did we run out of memory?  Or have other issues?
	if stat, err := sb.Status(sandbox.StatusMemFailures); err == nil {
		if b, err := strconv.ParseBool(stat); err == nil && b {
			return fmt.Errorf("ran out of memory while installing %s", p.name)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("install lambda returned status %d, body '%s'", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &p.meta); err != nil {
		return err
	}

	return nil
}

// does the pip install in a Sandbox, taking care to never install the
// same Sandbox more than once.
//
// the fast/slow path code is tweaked from the sync.Once code, the
// difference being that may try the installed more than once, but we
// will never try more after the first success
func (mp *PackagePuller) Install(pkg string) (*Package, error) {
	p := mp.GetPkg(pkg)

	// fast path
	if atomic.LoadUint32(&p.installed) == 1 {
		return p, nil
	}

	// slow path
	p.installMutex.Lock()
	defer p.installMutex.Unlock()
	if p.installed == 0 {
		if err := mp.sandboxInstall(p); err != nil {
			return p, err
		}
		atomic.StoreUint32(&p.installed, 1)
	}

	return p, nil
}
