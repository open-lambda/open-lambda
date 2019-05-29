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
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
)

var notFound404 = errors.New("file does not exist")

const SEPARATOR = ":"

// TODO: for web registries, support an HTTP-based access key
// (https://en.wikipedia.org/wiki/Basic_access_authentication)

// TODO: garbage collect old directories not used by any handler
// anymore

type CodePuller struct {
	codeCacheDir string   // where to download/copy code
	prefix       string   // combine with name to get file path or URL
	nextId       int64    // used to generate directory names for lambda code dirs
	dirCache     sync.Map // key=lambda name, value=version, directory path
}

type CacheEntry struct {
	version string // could be a timestamp for a file or web resource
	path    string // where code is extracted to a dir
}

func NewCodePuller(codeCacheDir, pullPrefix string) (cp *CodePuller, err error) {
	if err := os.MkdirAll(codeCacheDir, os.ModeDir); err != nil {
		return nil, fmt.Errorf("fail to create directory at %s: ", codeCacheDir, err)
	}

	return &CodePuller{codeCacheDir: codeCacheDir, prefix: pullPrefix, nextId: 0}, nil
}

func (cp *CodePuller) isRemote() bool {
	return strings.HasPrefix(cp.prefix, "http://") || strings.HasPrefix(cp.prefix, "https://")
}

func (cp *CodePuller) Pull(name string) (targetDir string, err error) {
	matched, err := regexp.MatchString(`^[A-Za-z0-9\.\[\_]+$`, name)
	if err != nil {
		return "", err
	} else if !matched {
		msg := "lambda names can only contain letters, numbers, period, dash, and underscore; found '%s'"
		return "", fmt.Errorf(msg, name)
	}

	if cp.isRemote() {
		// registry type = web
		urls := []string{
			cp.prefix + "/" + name + ".tar.gz",
			cp.prefix + "/" + name + ".py",
		}

		for i := 0; i < len(urls); i++ {
			targetDir, err = cp.pullRemoteFile(urls[i], name)
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
			filepath.Join(cp.prefix, name) + ".tar.gz",
			filepath.Join(cp.prefix, name) + ".py",
			filepath.Join(cp.prefix, name),
		}

		for i := 0; i < len(paths); i++ {
			if _, err := os.Stat(paths[i]); !os.IsNotExist(err) {
				targetDir, err = cp.pullLocalFile(paths[i], name)
				return targetDir, err
			}
		}

		return "", fmt.Errorf("lambda not found at any of these locations: %s", strings.Join(paths, ", "))
	}
}

func (cp *CodePuller) newCodeDir(lambdaName string) (targetDir string) {
	targetDir = fmt.Sprintf("%d-%s", atomic.AddInt64(&cp.nextId, 1), lambdaName)
	targetDir = filepath.Join(cp.codeCacheDir, targetDir)
	return targetDir
}

func (cp *CodePuller) pullLocalFile(src, lambdaName string) (targetDir string, err error) {
	stat, err := os.Stat(src)
	if err != nil {
		return "", err
	}

	if stat.Mode().IsDir() {
		// this is really just a debug mode, and is not
		// expected to be efficient
		targetDir = cp.newCodeDir(lambdaName)

		cmd := exec.Command("cp", "-r", src, targetDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("%s :: %s", err, string(output))
		}
		return targetDir, nil
	} else if !stat.Mode().IsRegular() {
		return "", fmt.Errorf("%s not a file or directory", src)
	}

	// for regular files, we cache based on mod time.  We don't
	// cache at the file level if this is a remote store (because
	// caching is handled at the web level)
	version := stat.ModTime().String()
	if !cp.isRemote() {
		cacheEntry := cp.getCache(lambdaName)
		if cacheEntry != nil && cacheEntry.version == version {
			// hit:
			return cacheEntry.path, nil
		}
	}

	// miss:
	targetDir = cp.newCodeDir(lambdaName)
	if err := os.Mkdir(targetDir, os.ModeDir); err != nil {
		return "", err
	}

	if strings.HasSuffix(src, ".py") {
		cmd := exec.Command("cp", src, filepath.Join(targetDir, "lambda_func.py"))
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("%s :: %s", err, string(output))
		}
	} else if strings.HasSuffix(src, ".tar.gz") {
		cmd := exec.Command("tar", "-xzf", src, "--directory", targetDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("%s :: %s", err, string(output))
		}
	} else {
		return "", fmt.Errorf("lambda file %s not a .ta.rgz or .py", src)
	}

	if !cp.isRemote() {
		cp.putCache(lambdaName, version, targetDir)
	}

	return targetDir, nil
}

func (cp *CodePuller) pullRemoteFile(src, lambdaName string) (targetDir string, err error) {
	// grab latest lambda code if it's changed (pass
	// If-Modified-Since so this can be determined on server side
	client := &http.Client{}
	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return "", err
	}

	cacheEntry := cp.getCache(lambdaName)
	if cacheEntry != nil {
		req.Header.Set("If-Modified-Since", cacheEntry.version)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", notFound404
	}

	if resp.StatusCode == http.StatusNotModified {
		return cacheEntry.path, nil
	}

	// download to local file, then use pullLocalFile to finish
	dir, err := ioutil.TempDir("", "ol-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	parts := strings.Split(src, "/")
	localPath := filepath.Join(dir, parts[len(parts)-1])
	out, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return "", err
	}

	targetDir, err = cp.pullLocalFile(localPath, lambdaName)

	// record directory in cache, by mod time
	if err == nil {
		version := resp.Header.Get("Last-Modified")
		if version != "" {
			cp.putCache(lambdaName, version, targetDir)
		}
	}

	return targetDir, err
}

func (cp *CodePuller) getCache(name string) *CacheEntry {
	entry, found := cp.dirCache.Load(name)
	if !found {
		return nil
	}
	return entry.(*CacheEntry)
}

func (cp *CodePuller) putCache(name, version, path string) {
	cp.dirCache.Store(name, &CacheEntry{version, path})
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
