package frontends

import ()

type BaseFrontEnd struct {
	Name  string
	OlDir string
}

func (bf *BaseFrontEnd) FrontEndName() string {
	return bf.Name
}

func (bf *BaseFrontEnd) OlDirLocation() string {
	return bf.OlDir
}
