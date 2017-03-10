package pip

import (
	"os/exec"

	"github.com/open-lambda/open-lambda/worker/config"
)

/*
 * InstallManager is the interface for installing pip packages locally.
 * The manager installs to the worker host from an optional pip mirror.
 */

type InstallManager interface {
	Install(pkgs []string) error
}

type Installer struct {
    cmd       string
    args      []string
    installed map[string]bool
}

func NewInstaller(opts *config.Config) *Installer {
    cmd := "pip"
    args := []string{"install"} // TODO: cache option?
    if opts.Pip_mirror != "" {
        args = append(args, "-i", opts.Pip_mirror)
    }

    manager := &Installer{
        cmd:       cmd,
        args:      args,
        installed: make(map[string]bool),
    }

	return manager
}

// TODO: eviction
func (i *Installer) Install(pkgs []string) error {
    for _, pkg := range pkgs {
        if _, ok := i.installed[pkg]; !ok {
            cmd := exec.Command(i.cmd, append(i.args, pkg)...)
            if err := cmd.Run(); err != nil {
                return err
            }
        }
    }

    return nil
}
