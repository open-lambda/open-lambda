package frontends

import ()

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
