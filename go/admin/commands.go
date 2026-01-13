package admin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/open-lambda/open-lambda/go/boss/config"
	"github.com/open-lambda/open-lambda/go/common"

	"github.com/urfave/cli/v2"
)

func checkStatus(port string) error {
	host := "localhost"

	url := fmt.Sprintf("http://%s:%s/status", host, port)
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("could not reach boss/worker at %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("boss/worker returned status %d (failed to read response body: %v)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("boss/worker returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

const installUsage = "ol admin install [-c <config>] [-r <requirements>] [-n <name>] [boss | -p <worker_path>] <directory_or_git_url>"

// isGitURL returns true if the path looks like a git repository URL
func isGitURL(path string) bool {
	if !strings.HasSuffix(path, ".git") {
		return false
	}
	return strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "git@")
}

// cloneGitRepo clones a git repository to a temporary directory
func cloneGitRepo(gitURL string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "ol-install-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	cmd := exec.Command("git", "clone", "--depth", "1", gitURL, tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("git clone failed: %v\n%s", err, string(output))
	}

	return tmpDir, nil
}

func adminInstall(ctx *cli.Context) error {
	args := ctx.Args().Slice()
	var installTarget string
	var funcDir string

	workerPath := ctx.String("path")

	if len(args) == 0 {
		return fmt.Errorf("usage: %s", installUsage)
	}
	if len(args) == 1 {
		funcDir = args[0]
		installTarget = "worker"

	} else if len(args) == 2 && args[0] == "boss" {
		installTarget = "boss"
		funcDir = args[1]
		if workerPath != "" {
			return fmt.Errorf("cannot use both 'boss' and '-p' flags together")
		}
	} else {
		return fmt.Errorf("usage: %s", installUsage)
	}

	var portToUploadLambda string

	switch installTarget {
	case "boss":
		if err := config.LoadConf("boss.json"); err != nil {
			return fmt.Errorf("failed to load boss config: %v", err)
		}
		if err := checkStatus(config.BossConf.Boss_port); err != nil {
			return fmt.Errorf("boss is not running: %v", err)
		}
		portToUploadLambda = config.BossConf.Boss_port

	case "worker":
		if workerPath == "" {
			olPath, err := common.GetOlPath(ctx)
			if err != nil {
				return err
			}

			if err := common.LoadDefaults(olPath); err != nil {
				return fmt.Errorf("failed to load default worker config for %s: %v", workerPath, err)
			}
		} else {
			if err := common.LoadGlobalConfig(filepath.Join(workerPath, "config.json")); err != nil {
				return fmt.Errorf("failed to load worker config for %s: %v", workerPath, err)
			}
		}

		if err := checkStatus(common.Conf.Worker_port); err != nil {
			return fmt.Errorf("worker %s is not running: %v", workerPath, err)
		}
		portToUploadLambda = common.Conf.Worker_port
	}

	var funcName string
	var tmpDir string

	if isGitURL(funcDir) {
		funcName = strings.TrimSuffix(filepath.Base(funcDir), ".git")
		clonedDir, err := cloneGitRepo(funcDir)
		if err != nil {
			return err
		}
		tmpDir = clonedDir
		funcDir = clonedDir
	} else {
		funcDir = strings.TrimSuffix(funcDir, "/")
		funcName = filepath.Base(funcDir)
		if _, err := os.Stat(funcDir); os.IsNotExist(err) {
			return fmt.Errorf("directory %s does not exist", funcDir)
		}
	}

	// Override function name if specified
	if name := ctx.String("name"); name != "" {
		funcName = name
	}

	// Build overrides map
	overrides := make(map[string]string)
	addOverride := func(flagName, targetFile string) error {
		path := ctx.String(flagName)
		if path == "" {
			return nil
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("%s file %s does not exist", flagName, path)
		}
		if _, err := os.Stat(filepath.Join(funcDir, targetFile)); err == nil {
			fmt.Printf("Warning: overriding existing %s in source with %s\n", targetFile, path)
		}
		overrides[targetFile] = path
		return nil
	}

	if err := addOverride("config", "ol.yaml"); err != nil {
		return err
	}
	if err := addOverride("requirements", "requirements.txt"); err != nil {
		return err
	}

	tarData, err := createTarGz(funcDir, overrides)
	if tmpDir != "" {
		os.RemoveAll(tmpDir)
	}
	if err != nil {
		return fmt.Errorf("failed to create tar.gz: %v", err)
	}

	if err := uploadToLambdaStore(funcName, tarData, portToUploadLambda); err != nil {
		return fmt.Errorf("failed to upload to lambda store: %v", err)
	}

	fmt.Printf("Successfully installed lambda function: %s\n", funcName)
	return nil
}

func createTarGz(funcDir string, overrides map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Determine the Python entry file (default to f.py, or use OL_ENTRY_FILE from ol.yaml)
	// Check override config first, then fall back to source config
	pythonEntryFile := "f.py"
	configDir := funcDir
	if configOverride, ok := overrides["ol.yaml"]; ok {
		configDir = filepath.Dir(configOverride)
	}
	lambdaConfig, err := common.LoadLambdaConfig(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config in %s: %v", configDir, err)
	}
	if lambdaConfig.Environment != nil {
		if entryFile, ok := lambdaConfig.Environment["OL_ENTRY_FILE"]; ok {
			pythonEntryFile = entryFile
		}
	}

	entryPath := filepath.Join(funcDir, pythonEntryFile)
	if _, err := os.Stat(entryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("required file %s not found in %s", pythonEntryFile, funcDir)
	}

	err = filepath.Walk(funcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error: %v", err)
		}

		if info.IsDir() {
			return nil
		}

		if !info.Mode().IsRegular() {
			return fmt.Errorf("cannot archive non-regular file %q (mode: %s)", path, info.Mode().String())
		}

		relPath, err := filepath.Rel(funcDir, path)
		if err != nil {
			return fmt.Errorf("unable to compute relative path: %v", err)
		}

		// Skip files that will be overridden
		if _, ok := overrides[relPath]; ok {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("unable to create header: %v", err)
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write header: %v", err)
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("unable to open file: %v", err)
		}

		if _, err := io.Copy(tarWriter, file); err != nil {
			file.Close()
			return fmt.Errorf("error copying file data: %v", err)
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("error closing file: %v", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Add override files
	for relPath, localPath := range overrides {
		info, err := os.Stat(localPath)
		if err != nil {
			return nil, fmt.Errorf("unable to stat override file %s: %v", localPath, err)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return nil, fmt.Errorf("unable to create header for override %s: %v", relPath, err)
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("failed to write header for override %s: %v", relPath, err)
		}

		file, err := os.Open(localPath)
		if err != nil {
			return nil, fmt.Errorf("unable to open override file %s: %v", localPath, err)
		}

		if _, err := io.Copy(tarWriter, file); err != nil {
			file.Close()
			return nil, fmt.Errorf("error copying override file %s: %v", localPath, err)
		}

		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("error closing override file %s: %v", localPath, err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %v", err)
	}

	return buf.Bytes(), nil
}

func uploadToLambdaStore(funcName string, tarData []byte, port string) error {
	host := "localhost"

	url := fmt.Sprintf("http://%s:%s/registry/%s", host, port, funcName)

	req, err := http.NewRequest("POST", url, bytes.NewReader(tarData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/gzip")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("upload failed with status %d (failed to read response body: %v)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func AdminCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:      "install",
			Usage:     "Install a lambda function from directory or git repo",
			UsageText: installUsage,
			Action:    adminInstall,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "path",
					Aliases: []string{"p"},
					Usage:   "Worker directory path (e.g., -p myworker)",
				},
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "Path to ol.yaml config file to include (overrides existing ol.yaml in source)",
				},
				&cli.StringFlag{
					Name:    "requirements",
					Aliases: []string{"r"},
					Usage:   "Path to requirements.txt file to include (overrides existing requirements.txt in source)",
				},
				&cli.StringFlag{
					Name:    "name",
					Aliases: []string{"n"},
					Usage:   "Lambda function name (defaults to directory or repo name)",
				},
			},
		},
	}
}
