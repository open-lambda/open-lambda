package sandbox

import (
	"errors"
	"fmt"
	"sync"
)

// SandboxSet manages a thread-safe pool of sandboxes.
// It handles dynamic creation and reuse of sandboxes without goroutines.
type SandboxSet struct {
	mutex sync.Mutex

	// Pool of sandbox wrappers
	sandboxes []*sandboxWrapper

	// Configuration
	maxSandboxes int

	// Dependencies for creating new sandboxes
	sbPool SandboxPool

	// Metadata for creating sandboxes
	meta       *SandboxMeta
	isLeaf     bool
	codeDir    string
	scratchDir string

	// Statistics
	created  int
	borrowed int
	released int
}

// sandboxWrapper tracks a sandbox and its usage state
type sandboxWrapper struct {
	sb    Sandbox
	inUse bool
}

// NewSandboxSet creates a new sandbox pool
func NewSandboxSet(
	sbPool SandboxPool,
	meta *SandboxMeta,
	isLeaf bool,
	codeDir string,
	scratchDir string,
	maxSandboxes int) *SandboxSet {

	if maxSandboxes <= 0 {
		maxSandboxes = 10 // reasonable default
	}

	// Fill in default values for meta
	meta = fillMetaDefaults(meta)

	return &SandboxSet{
		sandboxes:    []*sandboxWrapper{},
		maxSandboxes: maxSandboxes,
		sbPool:       sbPool,
		meta:         meta,
		isLeaf:       isLeaf,
		codeDir:      codeDir,
		scratchDir:   scratchDir,
		created:      0,
		borrowed:     0,
		released:     0,
	}
}

// GetSandbox returns an available sandbox or creates a new one.
// Returns error if all sandboxes are busy and max capacity reached.
func (set *SandboxSet) GetSandbox() (Sandbox, error) {
	set.mutex.Lock()
	defer set.mutex.Unlock()

	// First, try to find an unused sandbox
	for _, wrapper := range set.sandboxes {
		if !wrapper.inUse && wrapper.sb != nil {
			wrapper.inUse = true
			set.borrowed++
			return wrapper.sb, nil
		}
	}

	// All sandboxes are in use - check if we can create a new one
	if len(set.sandboxes) >= set.maxSandboxes {
		return nil, fmt.Errorf(
			"SandboxSet at capacity (%d/%d sandboxes in use)",
			len(set.sandboxes), set.maxSandboxes)
	}

	// Create a new sandbox
	sb, err := set.createSandbox()
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %v", err)
	}

	// Add to pool and mark as in-use
	wrapper := &sandboxWrapper{
		sb:    sb,
		inUse: true,
	}
	set.sandboxes = append(set.sandboxes, wrapper)
	set.created++
	set.borrowed++

	return sb, nil
}

// ReleaseSandbox marks a sandbox as available for reuse
func (set *SandboxSet) ReleaseSandbox(sb Sandbox) error {
	if sb == nil {
		return errors.New("cannot release nil sandbox")
	}

	set.mutex.Lock()
	defer set.mutex.Unlock()

	// Find the sandbox in our pool
	for _, wrapper := range set.sandboxes {
		if wrapper.sb == sb {
			if !wrapper.inUse {
				return fmt.Errorf("sandbox %s was not in use", sb.ID())
			}
			wrapper.inUse = false
			set.released++
			return nil
		}
	}

	return fmt.Errorf("sandbox %s not found in this set", sb.ID())
}

// createSandbox is a helper that creates a new sandbox
// Assumes mutex is already held by caller
func (set *SandboxSet) createSandbox() (Sandbox, error) {
	// Create the sandbox using the pool
	sb, err := set.sbPool.Create(
		nil, // no parent (zygote support will come in Step 3)
		set.isLeaf,
		set.codeDir,
		set.scratchDir,
		set.meta,
	)

	if err != nil {
		return nil, err
	}

	return sb, nil
}

// Destroy cleans up all sandboxes in the set
func (set *SandboxSet) Destroy() {
	set.mutex.Lock()
	defer set.mutex.Unlock()

	for _, wrapper := range set.sandboxes {
		if wrapper.sb != nil {
			wrapper.sb.Destroy("SandboxSet cleanup")
		}
	}
	set.sandboxes = nil
}

// Stats returns statistics about the sandbox set
func (set *SandboxSet) Stats() map[string]int {
	set.mutex.Lock()
	defer set.mutex.Unlock()

	inUse := 0
	for _, wrapper := range set.sandboxes {
		if wrapper.inUse {
			inUse++
		}
	}

	return map[string]int{
		"total":    len(set.sandboxes),
		"in_use":   inUse,
		"idle":     len(set.sandboxes) - inUse,
		"max":      set.maxSandboxes,
		"created":  set.created,
		"borrowed": set.borrowed,
		"released": set.released,
	}
}

// Size returns the current number of sandboxes in the set
func (set *SandboxSet) Size() int {
	set.mutex.Lock()
	defer set.mutex.Unlock()
	return len(set.sandboxes)
}

// InUse returns the number of sandboxes currently in use
func (set *SandboxSet) InUse() int {
	set.mutex.Lock()
	defer set.mutex.Unlock()

	inUse := 0
	for _, wrapper := range set.sandboxes {
		if wrapper.inUse {
			inUse++
		}
	}
	return inUse
}
