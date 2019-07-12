package sandbox

import (
	"fmt"
	"strings"

	"github.com/open-lambda/open-lambda/ol/config"
)

func SandboxPoolFromConfig(name string, sizeMb int) (cf SandboxPool, err error) {
	if config.Conf.Sandbox == "docker" {
		return NewDockerPool("", nil)
	} else if config.Conf.Sandbox == "sock" {
		mem := NewMemPool(name, sizeMb)
		pool, err := NewSOCKPool(name, mem)
		if err != nil {
			return nil, err
		}
		NewSOCKEvictor(pool)
		return pool, nil
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", config.Conf.Sandbox)
}

func fillMetaDefaults(meta *SandboxMeta) *SandboxMeta {
	if meta == nil {
		meta = &SandboxMeta{}
	}
	if meta.MemLimitMB == 0 {
		meta.MemLimitMB = config.Conf.Limits.Mem_mb
	}
	return meta
}

func (meta *SandboxMeta) String() string {
	return fmt.Sprintf("<installs=[%s], imports=[%s], mem-limit-mb=%v>",
		strings.Join(meta.Installs, ","), strings.Join(meta.Imports, ","), meta.MemLimitMB)
}

func (e SockError) Error() string {
	return string(e)
}
