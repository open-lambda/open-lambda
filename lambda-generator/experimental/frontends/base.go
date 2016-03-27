package frontends

import ()

type BaseFrontEnd struct {
	Name string
}

func (bf *BaseFrontEnd) FrontEndName() string {
	return bf.Name
}
