package sandbox

import (
	"context"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/containerd/containerd"

	"github.com/open-lambda/open-lambda/go/worker/sandbox/containerdutil"
)

type ContainerdContainer struct {
	// core containerd resources
	container  containerd.Container // containerd container handle
	task       containerd.Task      // Running process inside the container
	ctx        context.Context
	scratchDir string

	execProcess containerd.Process

	// state tracking
	isPaused bool // cached pause state to avoid API calls
	// Lambda execution resources
	meta       *SandboxMeta
	httpClient *http.Client
}

func (c *ContainerdContainer) ID() string {
	return c.container.ID()
}

func (c *ContainerdContainer) Destroy(reason string) {
	slog.Info("Destroying container", "container_id", c.container.ID(), "reason", reason)

	cleanupSuccessful := containerdutil.CleanupContainerdResources(c.ctx, c.container.ID(), c.container, c.task, c.execProcess)
	if !cleanupSuccessful {
		slog.Error("Errors occurred during cleanup of container", "container_id", c.container.ID())
	}
}

func (c *ContainerdContainer) DestroyIfPaused(reason string) {
	c.Destroy(reason) // safeSandbox.DestroyIfPaused() checks if paused in wrapper
}

func (c *ContainerdContainer) Pause() error {

	if c.isPaused {
		return nil
	}
	// Attempt pausing directly (not checking status first bc it takes significant time)
	if err := c.task.Pause(c.ctx); err != nil {
		// Only check status if pause fails (rare case)
		status, statusErr := c.task.Status(c.ctx)
		if statusErr != nil || status.Status != containerd.Paused {
			return fmt.Errorf("failed to pause container %s: %v", c.container.ID(), err)
		}
	}
	c.isPaused = true
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *ContainerdContainer) Unpause() error {

	if !c.isPaused {
		return nil
	}

	// Attempt resume directly - containerd handles already-running containers gracefully
	if err := c.task.Resume(c.ctx); err != nil {
		// Only check status if resume fails (rare case)
		if status, statusErr := c.task.Status(c.ctx); statusErr == nil {
			if status.Status == containerd.Running {
				// Container was already running, not an error
				c.isPaused = false
				return nil
			}
		}
		return fmt.Errorf("failed to resume container %s: %v", c.container.ID(), err)
	}

	c.isPaused = false
	return nil
}

func (c *ContainerdContainer) Client() *http.Client {
	return c.httpClient
}

func (c *ContainerdContainer) Meta() *SandboxMeta {
	return c.meta
}

func (c *ContainerdContainer) GetRuntimeLog() string {
	data, err := ioutil.ReadFile(filepath.Join(c.scratchDir, "stdout"))

	if err == nil {
		return string(data)
	}

	return ""
}

func (c *ContainerdContainer) GetProxyLog() string {
	return "containerd does not use proxy"
}

func (c *ContainerdContainer) DebugString() string {
	return "ContainerdContainer ID: " + c.container.ID()
}

func (c *ContainerdContainer) fork(dst Sandbox) error {
	return fmt.Errorf("fork not supported for containerd")
}

func (c *ContainerdContainer) childExit(child Sandbox) {}
