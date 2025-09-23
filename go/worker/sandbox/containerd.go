package sandbox

import (
	"context"
	"fmt"
	"net/http"
	"log/slog"
        "io/ioutil"
        "path/filepath"

	"github.com/containerd/containerd"
        
	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/sandbox/containerdutil"
)


type ContainerdContainer struct {
	// core containerd resources
	id        	string
	container 	containerd.Container  // containerd container handle
	task      	containerd.Task       // Running process inside the container
	client		*containerd.Client
	ctx			context.Context
	scratchDir	string

	execProcess	containerd.Process  

	// state tracking
	destroyed	bool        // tracks if WE destroyed it, not just if it's gone
	isPaused	bool        // cached pause state to avoid API calls
	// Lambda execution resources
	meta		*SandboxMeta
	rtType		common.RuntimeType
	httpClient	*http.Client
}

func (c *ContainerdContainer) ID() string {
	return c.id
}

func (c *ContainerdContainer) Destroy(reason string) {
	if c.destroyed {
		return // destruction was already attempted
	}

	slog.Info("Destroying container", "container_id", c.id, "reason", reason)

	// Use the shared cleanup function from containerdutil for consistent error handling
	cleanupSuccessful := containerdutil.CleanupContainerdResources(c.ctx, c.id, c.container, c.task, c.execProcess)
	if !cleanupSuccessful {
		slog.Error("Errors occurred during cleanup of container", "container_id", c.id)
	}
	c.destroyed = true
}

func (c *ContainerdContainer) DestroyIfPaused(reason string) {
		c.Destroy(reason) // safeSandbox.DestroyIfPaused() checks if paused in wrapper
}

func (c *ContainerdContainer) Pause() error {
        // Optimized: Use cached state and skip if already paused
        if c.destroyed {
                return fmt.Errorf("cannot pause destroyed container %s", c.id)
        }

        // Skip if already paused (cached state)
        if c.isPaused {
                return nil
        }

        // Attempt pause directly - containerd handles already-paused containers gracefully
        if err := c.task.Pause(c.ctx); err != nil {
                // Only check status if pause fails (rare case)
                if status, statusErr := c.task.Status(c.ctx); statusErr == nil {
                        if status.Status == containerd.Paused {
                                // Container was already paused, not an error
                                c.isPaused = true
                                c.httpClient.CloseIdleConnections()
                                return nil
                        }
                }
                return fmt.Errorf("failed to pause container %s: %v", c.id, err)
        }

        c.isPaused = true

        // idle connections use a LOT of memory in the OL process
        c.httpClient.CloseIdleConnections()

        return nil
}

func (c *ContainerdContainer) Unpause() error {
        // Optimized: Use cached state and skip unnecessary operations
        if c.destroyed {
                return fmt.Errorf("cannot unpause destroyed container %s", c.id)
        }

        // Skip if already running (cached state)
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
                return fmt.Errorf("failed to resume container %s: %v", c.id, err)
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

func (c *ContainerdContainer) GetRuntimeLog() (string) {
	data, err := ioutil.ReadFile(filepath.Join(c.scratchDir, "stdout"))

      if err == nil {
          return string(data)
      }

      return ""
}

func (c *ContainerdContainer) GetProxyLog() (string) {
	return "containerd does not use proxy"
}

func (c *ContainerdContainer) DebugString() string {
	return "ContainerdContainer ID: " + c.id
}

func (c *ContainerdContainer) fork(dst Sandbox) error {
	return fmt.Errorf("fork not supported for containerd")
}

func (c *ContainerdContainer) childExit(child Sandbox) {}

func (c *ContainerdContainer) GetRuntimeType() common.RuntimeType {
	return c.rtType
}


// this function is not implemented for containerd, because we are reusing WaitForServerPipeReady() from docker.go
// func waitForServerPipeReadyContainerd(hostDir string) error {
// 	return nil
// }