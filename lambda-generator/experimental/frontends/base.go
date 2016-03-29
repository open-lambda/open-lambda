package frontends

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

type BaseFrontEnd struct {
	Name       string
	OlDir      string
	ProjectDir string
}

func (bf *BaseFrontEnd) FrontEndName() string {
	return bf.Name
}

func (bf *BaseFrontEnd) OlDirLocation() string {
	return bf.OlDir
}

func (bf *BaseFrontEnd) ProjectDirLocation() string {
	return bf.ProjectDir
}

// Returns the ID for a given lambda path
// Id's are used to identify an individual lambda throughout the system
// Docker image tags are an example use of Id's
//
// Given	'hello/my/name.go'
// Returns  'hello-my-name'
func (fe *BaseFrontEnd) GetId(p string) (id string, err error) {
	// Given an absolute path, make it relative to project
	if path.IsAbs(p) {
		p, err = filepath.Rel(fe.ProjectDir, p)
		if err != nil {
			return "", err
		}
	}
	p = strings.Replace(p, string(os.PathSeparator), "-", -1)

	// Remove extention
	id = strings.TrimSuffix(p, filepath.Ext(p))

	return id, nil
}
