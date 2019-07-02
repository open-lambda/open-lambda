package lambda

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/open-lambda/open-lambda/ol/config"
)

/*
 * ModulePuller is the interface for installing pip packages locally.
 * The manager installs to the worker host from an optional pip mirror.
 *
 * TODO: implement eviction, support multiple versions
 */

type ModulePuller struct {
	mutex   sync.Mutex
	pkgInit map[string]*sync.Once
	pkgErr  map[string]error
}

func NewModulePuller() (*ModulePuller, error) {
	installer := &ModulePuller{
		pkgInit: make(map[string]*sync.Once),
		pkgErr:  make(map[string]error),
	}

	if err := installer.InstallAll(config.Conf.Startup_pkgs); err != nil {
		return nil, err
	}

	return installer, nil
}

func (mp *ModulePuller) Install(pkg string) (err error) {
	mp.mutex.Lock()
	once := mp.pkgInit[pkg]
	if once == nil {
		once = &sync.Once{}
		mp.pkgInit[pkg] = once
	}
	mp.mutex.Unlock()

	once.Do(func() {
		targetDir := filepath.Join(config.Conf.Pkgs_dir, pkg)
		if _, err := os.Stat(targetDir); err == nil {
			// assume dir exististence means it is installed already
			return
		}

		cmd := []string{"pip3", "install", "--no-deps", pkg, "-t", config.Conf.Pkgs_dir}
		if config.Conf.Pip_index != "" {
			cmd = append(cmd, "-i", config.Conf.Pip_index)
		}

		err := exec.Command(cmd[0], cmd[1:]...).Run()
		log.Printf("ModulePuller: %s [err=%v]", strings.Join(cmd, " "), err)
		mp.mutex.Lock()
		mp.pkgErr[pkg] = err
		mp.mutex.Unlock()
	})

	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	return mp.pkgErr[pkg]
}

func (mp *ModulePuller) InstallAll(pkgs []string) error {
	for _, pkg := range pkgs {
		if err := mp.Install(pkg); err != nil {
			return err
		}
	}

	return nil
}
