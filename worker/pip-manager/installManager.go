package pip

import (
	"os/exec"

	"github.com/open-lambda/open-lambda/worker/config"
)

/*
 * InstallManager is the interface for installing pip packages locally.
 * The manager installs to the worker host from an optional pip mirror.
 *
 * TODO: install on startup, implement eviction, support multiple versions
 */

type InstallManager interface {
	Install(pkgs []string) error
}

type Installer struct {
	cmd       string
	args      []string
	installed map[string]bool
}

func InitInstallManager(opts *config.Config) (*Installer, error) {
	cmd := "pip"
	args := []string{"install"}
	if opts.Pip_index != "" {
		args = append(args, "-i", opts.Pip_index)
	}

	args = append(args, "-t", opts.Pkgs_dir)

	installer := &Installer{
		cmd:       cmd,
		args:      args,
		installed: make(map[string]bool),
	}

	if err := installer.Install(opts.Startup_pkgs); err != nil {
		return nil, err
	}

	return installer, nil
}

func (i *Installer) Install(pkgs []string) error {
	for _, pkg := range pkgs {
		if _, ok := i.installed[pkg]; !ok {
			cmd := exec.Command(i.cmd, append(i.args, pkg)...)
			if err := cmd.Run(); err != nil {
				return err
			}
			i.installed[pkg] = true
		}
	}

	return nil
}
