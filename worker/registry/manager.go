package registry

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	r "github.com/open-lambda/open-lambda/registry/src"
	"github.com/open-lambda/open-lambda/worker/config"
)

// RegistryManager is the common interface for lambda code pulling functions.
type RegistryManager interface {
	Pull(name string) (codeDir string, pkgs []string, err error)
}

func InitRegistryManager(config *config.Config) (rm RegistryManager, err error) {
	if config.Registry == "olregistry" {
		rm, err = NewOLStoreManager(config)
	} else if config.Registry == "local" {
		rm, err = NewLocalManager(config)
	} else {
		return nil, errors.New("invalid 'registry' field in config")
	}

	return rm, nil
}

// LocalManager stores lambda code in a local directory.
type LocalManager struct {
	regDir string
}

// OLStoreManager pulls code from olstore and stores it in a local directory.
type OLStoreManager struct {
	regDir     string
	pullclient *r.PullClient
}

// NewLocalManager creates a local manager.
func NewLocalManager(opts *config.Config) (*LocalManager, error) {
	if err := os.MkdirAll(opts.Reg_dir, os.ModeDir); err != nil {
		return nil, fmt.Errorf("fail to create directory at %s: ", opts.Reg_dir, err)
	}

	return &LocalManager{opts.Reg_dir}, nil
}

// Pull checks the lambda handler actually exists in the registry directory.
func (lm *LocalManager) Pull(name string) (string, []string, error) {
	handlerDir := filepath.Join(lm.regDir, name)
	if _, err := os.Stat(handlerDir); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("handler does not exists at %s", handlerDir)
	} else if err != nil {
		return "", nil, err
	}

	pkgPath := filepath.Join(handlerDir, "packages.txt")
	_, err := os.Stat(pkgPath)
	if os.IsNotExist(err) {
		return handlerDir, []string{}, nil
	} else if err == nil {
		pkgs, err := parsePkgFile(pkgPath)
		if err != nil {
			return "", nil, err
		}

		return handlerDir, pkgs, nil
	}

	return "", nil, err
}

// NewOLStoreManager creates an olstore manager.
func NewOLStoreManager(opts *config.Config) (*OLStoreManager, error) {
	pullClient := r.InitPullClient(opts.Reg_cluster, r.DATABASE, r.TABLE)

	return &OLStoreManager{opts.Reg_dir, pullClient}, nil
}

// Pull pulls lambda handler tarball from olstore and decompress it to a local directory.
func (om *OLStoreManager) Pull(name string) (string, []string, error) {
	handlerDir := filepath.Join(om.regDir, name)
	if err := os.Mkdir(handlerDir, os.ModeDir); err != nil {
		return "", nil, err
	}

	pfiles := om.pullclient.Pull(name)
	handler := pfiles[r.HANDLER].([]byte)
	r := bytes.NewReader(handler)

	// TODO: try to uncompress without execing - faster?
	cmd := exec.Command("tar", "-xzf", "-", "--directory", handlerDir)
	cmd.Stdin = r
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("%s: %s", err, string(output))
	}

	pkgPath := filepath.Join(handlerDir, "packages.txt")
	_, err := os.Stat(pkgPath)
	if os.IsNotExist(err) {
		return handlerDir, []string{}, nil
	} else if err == nil {
		pkgs, err := parsePkgFile(pkgPath)
		if err != nil {
			return "", nil, err
		}

		return handlerDir, pkgs, nil
	}

	return "", nil, err
}

func parsePkgFile(path string) (pkgs []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scnr := bufio.NewScanner(file)
	for scnr.Scan() {
		pkgs = append(pkgs, scnr.Text())
	}

	return pkgs, nil
}
