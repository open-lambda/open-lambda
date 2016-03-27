package effe

import (
	"github.com/tylerharter/open-lambda/lambda-generator/experimental/frontends"
)

type FrontEnd struct {
	*frontends.BaseFrontEnd
}

func NewFrontEnd() *FrontEnd {
	return &FrontEnd{
		&frontends.BaseFrontEnd{
			Name: "effe",
		},
	}
}
