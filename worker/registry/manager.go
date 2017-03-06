package registry

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	r "github.com/open-lambda/open-lambda/registry/src"
	"github.com/open-lambda/open-lambda/worker/config"
)

// RegistryManager is the common interface for lambda code pulling functions.
type RegistryManager interface {
	Pull(name string) (savedAt string, err error)
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
func (lm *LocalManager) Pull(name string) (string, error) {
	handlerDir := filepath.Join(lm.regDir, name)
	if _, err := os.Stat(handlerDir); os.IsNotExist(err) {
		return "", fmt.Errorf("handler does not exists at %s", handlerDir)
	} else if err != nil {
		return "", err
	}
	return handlerDir, nil
}

// NewOLStoreManager creates an olstore manager.
func NewOLStoreManager(opts *config.Config) (*OLStoreManager, error) {
	pullClient := r.InitPullClient(opts.Reg_cluster, r.DATABASE, r.TABLE)
	return &OLStoreManager{opts.Reg_dir, pullClient}, nil
}

// Pull pulls lambda handler tarball from olstore and decompress it to a local directory.
func (om *OLStoreManager) Pull(name string) (string, error) {
	handlerDir := filepath.Join(om.regDir, name)
	if err := os.Mkdir(handlerDir, os.ModeDir); err != nil {
		return "", err
	}

	pfiles := om.pullclient.Pull(name)
	handler := pfiles[r.HANDLER].([]byte)
	r := bytes.NewReader(handler)

	// TODO: try to uncompress without execing - faster?
	cmd := exec.Command("tar", "-xzf", "-", "--directory", handlerDir)
	cmd.Stdin = r
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s: %s", err, string(output))
	}
	return handlerDir, nil
}
