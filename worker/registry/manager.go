package registry

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	minio "github.com/minio/minio-go"

	"github.com/open-lambda/open-lambda/worker/config"
)

const SEPARATOR = ":"

// RegistryManager is the common interface for lambda code pulling functions.
type RegistryManager interface {
	Pull(name string) (handlerDir string, imports, installs []string, err error)
}

func InitRegistryManager(config *config.Config) (rm RegistryManager, err error) {
	if config.Registry == "local" {
		rm, err = NewLocalManager(config)
	} else if config.Registry == "remote" {
		rm, err = NewRemoteManager(config)
	} else {
		return nil, errors.New("invalid 'registry' field in config")
	}

	return rm, nil
}

// LocalManager stores lambda code in a local directory.
type LocalManager struct {
	regDir string
}

// NewLocalManager creates a local manager.
func NewLocalManager(opts *config.Config) (*LocalManager, error) {
	if err := os.MkdirAll(opts.Registry_dir, os.ModeDir); err != nil {
		return nil, fmt.Errorf("fail to create directory at %s: ", opts.Registry_dir, err)
	}

	return &LocalManager{opts.Registry_dir}, nil
}

// Pull checks the lambda handler actually exists in the registry directory.
func (lm *LocalManager) Pull(name string) (handlerDir string,
	imports, installs []string, err error) {

	handlerDir = filepath.Join(lm.regDir, name)
	if _, err := os.Stat(handlerDir); os.IsNotExist(err) {
		return "", nil, nil, fmt.Errorf("handler does not exist: %s", handlerDir)
	} else if err != nil {
		return "", nil, nil, err
	}

	pkgPath := filepath.Join(handlerDir, "packages.txt")
	imports, installs, err = parsePkgFile(pkgPath)
	if err != nil {
		return "", nil, nil, err
	}

	return handlerDir, imports, installs, nil
}

// RemoteManager pulls code from olstore and stores it in a local directory.
type RemoteManager struct {
	regDir string
	client *minio.Client
}

// NewRemoteManager creates an olstore manager.
func NewRemoteManager(opts *config.Config) (*RemoteManager, error) {
	client, err := minio.New(opts.Registry_server, opts.Registry_access_key, opts.Registry_secret_key, false)
	if err != nil {
		return nil, err
	}

	return &RemoteManager{opts.Registry_dir, client}, nil
}

// Pull pulls lambda handler tarball from olstore and decompress it to a local directory.
func (rm *RemoteManager) Pull(name string) (handlerDir string, imports, installs []string, err error) {
	handlerDir = filepath.Join(rm.regDir, name)
	if _, err = os.Stat(handlerDir); err == nil {
		if err := os.RemoveAll(handlerDir); err != nil {
			return "", nil, nil, err
		}
	} else if !os.IsNotExist(err) {
		return "", nil, nil, err
	}

	if err := os.Mkdir(handlerDir, os.ModeDir); err != nil {
		return "", nil, nil, err
	}

	tmpPath := filepath.Join("/tmp", fmt.Sprintf("%s.tar.gz", name))
	defer os.Remove(tmpPath)
	if err := rm.client.FGetObject(config.REGISTRY_BUCKET, name, tmpPath, minio.GetObjectOptions{}); err != nil {
		return "", nil, nil, err
	}

	cmd := exec.Command("tar", "-xvzf", tmpPath, "--directory", handlerDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", nil, nil, fmt.Errorf("%s :: %s", err, string(output))
	}

	pkgPath := filepath.Join(handlerDir, "packages.txt")
	imports, installs, err = parsePkgFile(pkgPath)
	if err != nil {
		return "", nil, nil, err
	}

	return handlerDir, imports, installs, nil
}

func parsePkgFile(path string) (imports, installs []string, err error) {
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return []string{}, []string{}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scnr := bufio.NewScanner(file)
	for scnr.Scan() {
		pkgs := strings.Split(scnr.Text(), SEPARATOR)
		if len(pkgs) != 2 {
			return nil, nil, fmt.Errorf("malformed packages.txt, missing separator")
		}
		imports = append(imports, pkgs[0])
		installs = append(installs, pkgs[0])
	}

	return imports, installs, nil
}
