package admin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("boss/worker returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func adminInstall(ctx *cli.Context) error {
	args := ctx.Args().Slice()
	var installTarget string
	var funcDir string

	workerPath := ctx.String("path")

	if len(args) == 0 {
		return fmt.Errorf("usage: ol admin install [boss | -p <worker_path>] <function_directory>")
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
		return fmt.Errorf("usage: ol admin install [boss | -p <worker_path>] <function_directory>")
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

	funcDir = strings.TrimSuffix(funcDir, "/")

	funcName := filepath.Base(funcDir)

	if _, err := os.Stat(funcDir); os.IsNotExist(err) {
		return fmt.Errorf("directory %s does not exist", funcDir)
	}

	tarData, err := createTarGz(funcDir)
	if err != nil {
		return fmt.Errorf("failed to create tar.gz: %v", err)
	}

	if err := uploadToLambdaStore(funcName, tarData, portToUploadLambda); err != nil {
		return fmt.Errorf("failed to upload to lambda store: %v", err)
	}

	fmt.Printf("Successfully installed lambda function: %s\n", funcName)
	return nil
}

func createTarGz(funcDir string) ([]byte, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	fpyPath := filepath.Join(funcDir, "f.py")
	if _, err := os.Stat(fpyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("required file f.py not found in %s", funcDir)
	}

	err := filepath.Walk(funcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error: %v", err)
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(funcDir, path)
		if err != nil {
			return fmt.Errorf("unable to compute relative path: %v", err)
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func AdminCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:      "install",
			Usage:     "Install a lambda function from directory",
			UsageText: "ol admin install [boss | -p <worker_path>] <function_directory>",
			Action:    adminInstall,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "path",
					Aliases: []string{"p"},
					Usage:   "Worker directory path (e.g., -p myworker)",
				},
			},
		},
	}
}
