package registry

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	r "github.com/open-lambda/open-lambda/registry/src"
	"github.com/open-lambda/open-lambda/worker/config"
)

// RegistryManager is the common interface for lambda code pulling functions.
type RegistryManager interface {
	Pull(name string) (codeDir string, installs, imports []string, err error)
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
func (lm *LocalManager) Pull(name string) (string, []string, []string, error) {
	handlerDir := filepath.Join(lm.regDir, name)
	if _, err := os.Stat(handlerDir); os.IsNotExist(err) {
		return "", nil, nil, fmt.Errorf("handler does not exists at %s", handlerDir)
	} else if err != nil {
		return "", nil, nil, err
	}

	pkgPath := filepath.Join(handlerDir, "packages.txt")
	_, err := os.Stat(pkgPath)
	if os.IsNotExist(err) {
		return handlerDir, []string{}, []string{}, nil
	} else if err == nil {
		imports, installs, err := parsePkgFile(pkgPath)
		if err != nil {
			return "", nil, nil, err
		}

		return handlerDir, imports, installs, nil
	}

	return "", nil, nil, err
}

func parsePkgFile(path string) (imports, installs []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scnr := bufio.NewScanner(file)
	for scnr.Scan() {
		split := strings.Split(scnr.Text(), ":")
		if len(split) != 2 {
			example := "Each line should be of the form: <import>:<install>"
			errmsg := fmt.Sprintf("Malformed packages.txt file: %s\n%s", path, example)
			return nil, nil, errors.New(errmsg)
		}

		imports = append(imports, split[0])
		installs = append(installs, split[0])
	}

	return imports, installs, nil
}

// NewOLStoreManager creates an olstore manager.
func NewOLStoreManager(opts *config.Config) (*OLStoreManager, error) {
	pullClient := r.InitPullClient(opts.Reg_cluster, r.DATABASE, r.TABLE)

	return &OLStoreManager{opts.Reg_dir, pullClient}, nil
}

// Pull pulls lambda handler tarball from olstore and decompress it to a local directory.
func (om *OLStoreManager) Pull(name string) (string, []string, []string, error) {
	handlerDir := filepath.Join(om.regDir, name)
	if err := os.Mkdir(handlerDir, os.ModeDir); err != nil {
		return "", nil, nil, err
	}

	pfiles := om.pullclient.Pull(name)
	handler := pfiles[r.HANDLER].([]byte)
	r := bytes.NewReader(handler)

	// TODO: try to uncompress without execing - faster?
	cmd := exec.Command("tar", "-xzf", "-", "--directory", handlerDir)
	cmd.Stdin = r
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", nil, nil, fmt.Errorf("%s: %s", err, string(output))
	}

	return handlerDir, []string{}, []string{}, nil //TODO: actually get the packages
}
