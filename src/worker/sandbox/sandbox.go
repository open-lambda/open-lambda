package sandbox

import (
	"fmt"
	"strings"

	"github.com/open-lambda/open-lambda/ol/common"
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

// fillMetaDefaults populates zero-valued fields in meta.Limits using worker defaults.
// Always set; zero means "use default" and gets resolved here.
func fillMetaDefaults(meta *SandboxMeta) {
	if meta == nil {
		return
	}
	if meta.Limits.MemMB == 0 {
		meta.Limits.MemMB = common.Conf.Limits.Mem_mb
	}
	if meta.Limits.CPUPercent == 0 {
		meta.Limits.CPUPercent = common.Conf.Limits.CPU_percent
	}
	// If you moved runtime into Limits, resolve it here as well:
	if meta.Limits.RuntimeSec == 0 {
		meta.Limits.RuntimeSec = common.Conf.Limits.Max_runtime_default
	}
}

func (meta *SandboxMeta) String() string {
	return fmt.Sprintf(
		"<installs=[%s], imports=[%s], mem-limit-mb=%d>",
		strings.Join(meta.Installs, ","),
		strings.Join(meta.Imports, ","),
		meta.Limits.MemMB,
	)
}

func (e SandboxError) Error() string {
	return string(e)
}

func (e SandboxDeadError) Error() string {
	return string(e)
}
