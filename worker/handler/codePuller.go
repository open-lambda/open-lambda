package handler

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"regexp"
)

var notFound404 = errors.New("file does not exist")

const SEPARATOR = ":"

// TODO: for web registries, support an HTTP-based access key
// (https://en.wikipedia.org/wiki/Basic_access_authentication)

// TODO: implement check on version before pulling something we
// already have (this can be timestamp from HTTP or on local file)

type CodePuller struct {
	codeCacheDir string // where to download/copy code
	prefix string // combine with name to get file path or URL
	nextId int64 // used to generate directory names for lambda code dirs
}

func NewCodePuller(codeCacheDir, pullPrefix string) (cp *CodePuller, err error) {
	if err := os.MkdirAll(codeCacheDir, os.ModeDir); err != nil {
		return nil, fmt.Errorf("fail to create directory at %s: ", codeCacheDir, err)
	}

	return &CodePuller{codeCacheDir: codeCacheDir, prefix: pullPrefix, nextId: 0}, nil
}

func (cp *CodePuller) Pull(name string) (targetDir string, err error) {
	matched, err := regexp.MatchString(`^[A-Za-z0-9\.\[\_]+$`, name)
	if err != nil {
		return "", err
	} else if !matched {
		msg := "lambda names can only contain letters, numbers, period, dash, and underscore; found '%s'"
		return "", fmt.Errorf(msg, name)
	}

	dirName := fmt.Sprintf("%d-name", atomic.AddInt64(&cp.nextId, 1))
	targetDir = filepath.Join(cp.codeCacheDir, dirName)

	if strings.HasPrefix(cp.prefix, "http://") || strings.HasPrefix(cp.prefix, "https://") {
		// registry type = web
		urls := []string{
			cp.prefix + "/" + name + ".tar.gz",
			cp.prefix + "/" + name + ".py",
		}

		for i := 0; i < len(urls); i++ {
			err = cp.pullRemoteFile(urls[i], targetDir)
			if err == nil {
				return targetDir, nil
			} else if err != notFound404 {
				// 404 is OK, because we just go on to check the next URLs
				return "", err
			}
		}

		return "", fmt.Errorf("lambda not found at any of these locations: %s", strings.Join(urls, ", "))
	} else {
		// registry type = file
		paths := []string{
			filepath.Join(cp.prefix, name),
			filepath.Join(cp.prefix, name) + ".tar.gz",
			filepath.Join(cp.prefix, name) + ".py",
		}

		for i := 0; i < len(paths); i++ {
			if _, err := os.Stat(paths[i]); !os.IsNotExist(err) {
				err = cp.pullLocalFile(paths[i], targetDir)
				return targetDir, err
			}
		}

		return "", fmt.Errorf("lambda not found at any of these locations: %s", strings.Join(paths, ", "))
	}
}

func (cp *CodePuller) pullLocalFile(src, dst string) (err error) {
        stat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if stat.Mode().IsDir() {
		cmd := exec.Command("cp", "-r", src, dst)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s :: %s", err, string(output))
		}
	} else if stat.Mode().IsRegular() {
		if err := os.Mkdir(dst, os.ModeDir); err != nil {
			return err
		}

		if strings.HasSuffix(src, ".py") {
			cmd := exec.Command("cp", src, filepath.Join(dst, "lambda_func.py"))
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("%s :: %s", err, string(output))
			}	
		} else if strings.HasSuffix(src, ".tar.gz") {
			cmd := exec.Command("tar", "-xzf", src, "--directory", dst)
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("%s :: %s", err, string(output))
			}
		} else {
			return fmt.Errorf("%s not a directory, .ta.rgz, .py", src)
		}
	} else {
		return fmt.Errorf("%s not a file or directory", src)
	}

	return nil
}

func (cp *CodePuller) pullRemoteFile(src, dst string) (err error) {
	resp, err := http.Get(src)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return notFound404
	}

	dir, err := ioutil.TempDir("", "ol-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	parts := strings.Split(src, "/")
	localPath := filepath.Join(dir, parts[len(parts)-1])
	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
	    return err
	}

	return cp.pullLocalFile(localPath, dst)
}

func parsePkgFile(path string) (imports, installs []string, err error) {
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return []string{}, []string{}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scnr := bufio.NewScanner(file)
	for scnr.Scan() {
		pkgs := strings.Split(scnr.Text(), SEPARATOR)
		if len(pkgs) != 2 {
			return nil, nil, fmt.Errorf("malformed packages.txt, missing separator")
		}
		imports = append(imports, pkgs[0])
		installs = append(installs, pkgs[0])
	}

	return imports, installs, nil
}
