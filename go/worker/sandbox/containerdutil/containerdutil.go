// package includes utility functions not provided by containerd client
package containerdutil

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/snapshots"
)

const (
	CONTAINERD_LABEL_CLUSTER = "ol.cluster" // cluster name
	CONTAINERD_LABEL_RUNTIME = "ol.runtime" // runtime handler
)

// ImageExists checks if an image of name exists.
func ImageExists(ctx context.Context, client *containerd.Client, name string) (bool, error) {
	_, err := client.GetImage(ctx, name)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("error checking image %s: %w", name, err)
	}
	return true, nil
}

// graceful cleanup during normal Destroy():
// Resumes if paused;
// Sends SIGKILL to ALL processes
// WAITS for processes to exit
// Deletes task with WithProcessKill (safety net)
// Handles exec process separately
// Deletes container with snapshots
func CleanupContainerdResources(ctx context.Context, containerID string,
	c containerd.Container, task containerd.Task, execProcess containerd.Process) bool {

	if containerID == "" {
		if c != nil {
			containerID = fmt.Sprintf("container-%p", c)
		} else {
			containerID = fmt.Sprintf("cleanup-at-%d", time.Now().Unix())
		}
	}

	slog.Info("Cleaning up containerd resources for container", "container_id", containerID)
	hasErrors := false

	// Kill all processes in the container
	if task != nil {
		// Check if task is paused and resume it first (can't kill a paused task)
		status, err := task.Status(ctx)
		if err != nil {
			// Can't check status, but try to resume anyway in case it's paused
			slog.Error("Failed to get task status for container", "container_id", containerID, "error", err)
			if resumeErr := task.Resume(ctx); resumeErr != nil {
				// Log but continue - if it fails, we'll try to kill anyway
				if !errdefs.IsNotFound(resumeErr) {
					slog.Error("Failed to resume task for container", "container_id", containerID, "error", resumeErr)
				}
			}
		} else if status.Status == containerd.Paused {
			slog.Info("Container is paused, resuming before cleanup", "container_id", containerID)
			if err := task.Resume(ctx); err != nil {
				if !errdefs.IsNotFound(err) {
					slog.Error("Failed to resume paused task for container", "container_id", containerID, "error", err)
					// Continue with cleanup attempt anyway
				}
			}
		}

		if err := task.Kill(ctx, syscall.SIGKILL, containerd.WithKillAll); err != nil {
			// Ignore "not found" errors as the task may already be dead
			if !errdefs.IsNotFound(err) {
				slog.Error("Failed to kill task for container", "container_id", containerID, "error", err)
				hasErrors = true
			}
		}

		// Wait for the task to actually stop after killing it
		// Kill is asynchronous, so we need to wait for the process to exit
		if waitCh, err := task.Wait(ctx); err == nil {
			select {
			case <-waitCh:
			case <-ctx.Done():
				slog.Warn("Context cancelled while waiting for task to stop for container", "container_id", containerID)
			}
		}

		if _, err := task.Delete(ctx, containerd.WithProcessKill); err != nil {
			if !errdefs.IsNotFound(err) {
				slog.Error("Failed to delete task for container", "container_id", containerID, "error", err)
				hasErrors = true
			}
		}
	}

	if execProcess != nil {
		if _, err := execProcess.Delete(ctx); err != nil {
			if !errdefs.IsNotFound(err) {
				slog.Error("Failed to delete exec process for container", "container_id", containerID, "error", err)
				hasErrors = true
			}
		}
	}

	// Delete the container and its snapshot
	if c != nil {
		if err := c.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
			if !errdefs.IsNotFound(err) {
				slog.Error("Failed to delete container with snapshot cleanup", "container_id", containerID, "error", err)
				hasErrors = true
			}
		}
	}

	if hasErrors {
		slog.Warn("Container cleanup completed with errors", "container_id", containerID)
	} else {
		slog.Info("Container cleanup completed successfully", "container_id", containerID)
	}

	return !hasErrors
}

// hard kill at shutdown
func SafeKill(ctx context.Context, container containerd.Container) error {
	task, err := container.Task(ctx, nil)
	if err != nil {
		if errdefs.IsNotFound(err) {
			slog.Info("Container has no task, already stopped", "container_id", container.ID())
			return nil
		}
		return fmt.Errorf("failed to get task for container %s: %w", container.ID(), err)
	}

	// Ignoring status and forcibly delete the task
	if _, err := task.Delete(ctx, containerd.WithProcessKill); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to delete task: %w", err)
		}
	}

	return nil
}

// hard removal used at shutdown
func SafeRemove(ctx context.Context, client *containerd.Client, container containerd.Container) error {
	if err := SafeKill(ctx, container); err != nil {
		return err
	}

	slog.Info("Remove container", "container_id", container.ID())
	if err := container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to remove container %s: %w", container.ID(), err)
		}
	}

	return nil
}

// EnsureImageExists checks if a locally-built image exists in containerd.
// OpenLambda builds images locally (e.g., ol-min) and never pulls from registries.
// Images are built with 'make imgs/ol-min' which uses 'docker build'.
func EnsureImageExists(ctx context.Context, client *containerd.Client, imageName string) error {
	exists, err := ImageExists(ctx, client, imageName)
	if err != nil {
		return fmt.Errorf("error checking if image %s exists: %w", imageName, err)
	}

	if exists {
		slog.Info("Image found in containerd", "image_name", imageName)
		return nil
	}

	slog.Info("Image not found in containerd, attempting to import from Docker", "image_name", imageName)

	// First check if Docker has the image
	checkCmd := exec.Command("docker", "images", "-q", imageName)
	output, err := checkCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check Docker for image %s: %w\nMake sure Docker is running and the image exists", imageName, err)
	}

	if len(output) == 0 {
		return fmt.Errorf("image %s not found in Docker either. Please build it first with: make imgs/%s", imageName, imageName)
	}

	slog.Info("Importing image from Docker to containerd namespace", "image_name", imageName, "namespace", "openlambda")

	// Create the pipeline: docker save | ctr import
	saveCmd := exec.Command("docker", "save", imageName)
	importCmd := exec.Command("ctr", "-n", "openlambda", "images", "import", "-") // imported as " docker.io/library/imageName:latest"

	pipe, err := saveCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	importCmd.Stdin = pipe

	// Capture import command output for error reporting
	var importOutput bytes.Buffer
	importCmd.Stderr = &importOutput

	if err := importCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ctr import: %w", err)
	}

	if err := saveCmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker save: %w", err)
	}

	saveErr := saveCmd.Wait()
	importErr := importCmd.Wait()

	if saveErr != nil {
		return fmt.Errorf("docker save failed: %w", saveErr)
	}

	if importErr != nil {
		return fmt.Errorf("ctr import failed: %w\nOutput: %s", importErr, importOutput.String())
	}

	exists, err = ImageExists(ctx, client, imageName)
	if err != nil {
		return fmt.Errorf("error verifying imported image: %w", err)
	}

	if !exists {
		return fmt.Errorf("image import appeared successful but image %s still not found in containerd", imageName)
	}

	slog.Info("Successfully imported image from Docker to containerd", "image_name", imageName)
	return nil
}

// Dump prints the ID and state of all containers. Only for debugging.
func Dump(ctx context.Context, client *containerd.Client, namespace string) {
	containers, err := client.Containers(ctx)
	if err != nil {
		slog.Error("Could not get container list", "error", err)
		return
	}

	slog.Info("=====================================")
	slog.Info("Dumping container information", "namespace", namespace)
	for idx, container := range containers {
		info, err := container.Info(ctx)
		if err != nil {
			slog.Error("Could not get container info", "error", err)
			continue
		}

		var taskStatus string = "no task"
		task, err := container.Task(ctx, nil)
		if err == nil {
			status, err := task.Status(ctx)
			if err == nil {
				taskStatus = string(status.Status)
			}
		}

		slog.Info("Container information", "container_index", idx,
			"image", info.Image,
			"container_id", container.ID()[:12],
			"task_status", taskStatus)
	}
	slog.Info("=====================================")
}

func RemoveOrphanedResources(ctx context.Context, client *containerd.Client, clusterLabel string) {
	// check for any remaining containers with our cluster label
	containers, err := client.Containers(ctx,
		fmt.Sprintf("labels.%q==%q", CONTAINERD_LABEL_CLUSTER, clusterLabel))
	if err != nil {
		slog.Error("Failed to list containers for orphan cleanup", "error", err)
	} else if len(containers) > 0 {
		slog.Warn("Found orphaned containers, trying to clean up", "count", len(containers))
		for _, c := range containers {
			slog.Info("Removing orphaned container", "container_id", c.ID())
			if err := SafeRemove(ctx, client, c); err != nil {
				slog.Error("Failed to remove orphaned container", "container_id", c.ID(), "error", err)
			}
		}
	}

	snapshotService := client.SnapshotService("")
	err = snapshotService.Walk(ctx, func(ctx context.Context, info snapshots.Info) error {
		if strings.HasPrefix(info.Name, "ctrd-") {
			slog.Info("Removing orphaned snapshot", "snapshot_name", info.Name)
			if err := snapshotService.Remove(ctx, info.Name); err != nil {
				slog.Error("Failed to remove orphaned snapshot", "snapshot_name", info.Name, "error", err)
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("Failed to walk snapshots for cleanup", "error", err)
	}
}
