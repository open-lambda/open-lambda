package sbmanager

/*

Manages lambdas using a "local registry" (directory containing handlers).

Creates lambda containers using the generic base image defined in
dockerManagerBase.go (BASE_IMAGE).

Handler code is mapped into the container by attaching a directory
(<handler_dir>/<lambda_name>) when the container is started.

*/

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-lambda/open-lambda/worker/config"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type LocalManager struct {
	DockerManagerBase
	handler_dir string
}

func NewLocalManager(opts *config.Config) (manager *LocalManager, err error) {
	manager = new(LocalManager)
	manager.DockerManagerBase.init(opts)
	manager.handler_dir = opts.Reg_dir
	exists, err := manager.DockerImageExists(BASE_IMAGE)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("Docker image %s does not exist", BASE_IMAGE)
	}
	return manager, nil
}

func (lm *LocalManager) Create(name string, sandbox_dir string) (sb.Sandbox, error) {
	handler := filepath.Join(lm.handler_dir, name)
	volumes := []string{
		fmt.Sprintf("%s:%s", handler, "/handler"),
		fmt.Sprintf("%s:%s", sandbox_dir, "/host")}

	sandbox, err := lm.create(name, sandbox_dir, BASE_IMAGE, volumes)
	if err != nil {
		return nil, err
	}

	return sandbox, nil
}

func (lm *LocalManager) Pull(name string) error {
	path := filepath.Join(lm.handler_dir, name)
	_, err := os.Stat(path)

	return err

}
