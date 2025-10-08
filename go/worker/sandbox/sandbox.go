package sandbox

import (
	"fmt"
	"strings"

	"github.com/open-lambda/open-lambda/go/common"
)

func SandboxPoolFromConfig(name string, sizeMb int) (cf SandboxPool, err error) {
	if common.Conf.Sandbox == "docker" {
		return NewDockerPool("", nil)
	} else if common.Conf.Sandbox == "sock" {
		mem := NewMemPool(name, sizeMb)
		pool, err := NewSOCKPool(name, mem)
		if err != nil {
			return nil, err
		}
		NewSOCKEvictor(pool)
		return pool, nil
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", common.Conf.Sandbox)
}

func (meta *SandboxMeta) String() string {
	return fmt.Sprintf("<installs=[%s], imports=[%s], mem-limit-mb=%d>",
		strings.Join(meta.Installs, ","), strings.Join(meta.Imports, ","), meta.Limits.MemMB)
}

func (e SandboxError) Error() string {
	return string(e)
}

func (e SandboxDeadError) Error() string {
	return string(e)
}
