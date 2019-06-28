package lambda

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/open-lambda/open-lambda/ol/config"
)

/*
 * ModulePuller is the interface for installing pip packages locally.
 * The manager installs to the worker host from an optional pip mirror.
 *
 * TODO: implement eviction, support multiple versions
 */

type PackageState struct {
	mutex *sync.RWMutex
}

type ModulePuller struct {
	cmd       string
	args      []string
	mutex     *sync.Mutex
	pkgStates map[string]*PackageState
}

func NewModulePuller() (*ModulePuller, error) {
	cmd := "pip"
	args := []string{"install", "--no-deps"}
	if config.Conf.Pip_index != "" {
		args = append(args, "-i", config.Conf.Pip_index)
	}

	args = append(args, "-t", config.Conf.Pkgs_dir)

	installer := &ModulePuller{
		cmd:       cmd,
		args:      args,
		mutex:     &sync.Mutex{},
		pkgStates: make(map[string]*PackageState),
	}

	if err := installer.Install(config.Conf.Startup_pkgs); err != nil {
		return nil, err
	}

	return installer, nil
}

func (i *ModulePuller) Install(pkgs []string) error {
	for _, pkg := range pkgs {
		i.mutex.Lock()
		pkgState, ok := i.pkgStates[pkg]
		if !ok {
			rwMutex := &sync.RWMutex{}
			rwMutex.Lock()
			defer rwMutex.Unlock()
			i.pkgStates[pkg] = &PackageState{rwMutex}
			i.mutex.Unlock()
			cmd := exec.Command(i.cmd, append(i.args, pkg)...)
			if err := cmd.Run(); err != nil {
				i.mutex.Lock()
				delete(i.pkgStates, pkg)
				i.mutex.Unlock()
				return fmt.Errorf("failed to install package '%s' :: %v :: %v", pkg, err, cmd)
			}
		} else {
			// The ordering here will have to change when we implement package
			// eviction - the package could be evicted after dropping the global
			// lock. We will also have to release reader locks on eviction of
			// handlers/cache entries.
			i.mutex.Unlock()
			pkgState.mutex.RLock()
		}
	}

	return nil
}
