package lambda

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/sandbox"
	"github.com/open-lambda/open-lambda/ol/stats"
)

/*
 * ModulePuller is the interface for installing pip packages locally.
 * The manager installs to the worker host from an optional pip mirror.
 *
 * TODO: implement eviction, support multiple versions
 */
type ModulePuller struct {
	sbPool sandbox.SandboxPool

	// directory of lambda code that installs pip packages
	pipLambda string

	modules sync.Map
}

type Module struct {
	name         string
	installMutex sync.Mutex
	installed    uint32
}

func NewModulePuller(sbPool sandbox.SandboxPool) (*ModulePuller, error) {
	// create a lambda function for installing pip modules.  We do
	// each install in a Sandbox for two reasons:
	//
	// 1. packages may be malicious
	// 2. we want to install the right version, matching the Python
	//    in the Sandbox
	lines := []string{
		"import os",
		"",
		"def handler(event):",
		"    rc = os.system('pip3 install --no-deps %s -t /host' % event)",
		"    print('pip install returned code %d' % rc)",
		"    return rc",
		"",
	}
	pipLambda := filepath.Join(config.Conf.Worker_dir, "admin-lambdas", "pip-install")
	if err := os.MkdirAll(pipLambda, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(pipLambda, "lambda_func.py")
	code := []byte(strings.Join(lines, "\n"))
	if err := ioutil.WriteFile(path, code, 0600); err != nil {
		return nil, err
	}

	installer := &ModulePuller{
		sbPool:    sbPool,
		pipLambda: pipLambda,
	}

	return installer, nil
}

func (mp *ModulePuller) GetMod(pkg string) *Module {
	mod, _ := mp.modules.LoadOrStore(pkg, &Module{name: pkg})
	return mod.(*Module)
}

// do the pip install within a new Sandbox, to a directory mapped from
// the host.  We want the package on the host to share with all, but
// want to run the install in the Sandbox because we don't trust it.
func (mp *ModulePuller) sandboxInstall(pkg string) (err error) {
	// the pip-install lambda installs to /host, which is the the
	// same as scratchDir, which is the same as a sub-directory
	// named after the package in the packages dir
	scratchDir := filepath.Join(config.Conf.Pkgs_dir, pkg)

	if _, err := os.Stat(scratchDir); err == nil {
		// assume dir exististence means it is installed already
		log.Printf("skip installing %s, appears already installed from previous run of OL", pkg)
		return err
	}

	log.Printf("run pip install %s from a new Sandbox to %s on host", pkg, scratchDir)
	if err := os.Mkdir(scratchDir, 0700); err != nil {
		return err
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
	reqBody := bytes.NewReader([]byte(fmt.Sprintf("\"%s\"", pkg)))
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
		log.Printf("stat=%v, err=%v", stat, err)
		if b, err := strconv.ParseBool(stat); err == nil && b {
			return fmt.Errorf("ran out of memory while installing %s", pkg)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("install lambda returned status %d, body '%s'", resp.StatusCode, string(body))
	}

	sbody := string(body)
	if sbody != "0" {
		return fmt.Errorf("pip install returned status '%s'", sbody)
	}

	return nil
}

// does the pip install in a Sandbox, taking care to never install the
// same Sandbox more than once.
//
// the fast/slow path code is tweaked from the sync.Once code, the
// difference being that may try the installed more than once, but we
// will never try more after the first success
func (mp *ModulePuller) Install(pkg string) (err error) {
	m := mp.GetMod(pkg)

	// fast path
	if atomic.LoadUint32(&m.installed) == 1 {
		return nil
	}

	// slow path
	m.installMutex.Lock()
	defer m.installMutex.Unlock()
	if m.installed == 0 {
		if err := mp.sandboxInstall(pkg); err != nil {
			return err
		}
		atomic.StoreUint32(&m.installed, 1)
	}

	return nil
}
