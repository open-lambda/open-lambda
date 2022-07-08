package lambda

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/open-lambda/open-lambda/ol/common"
)

var notFound404 = errors.New("file does not exist")

// TODO: for web registries, support an HTTP-based access key
// (https://en.wikipedia.org/wiki/Basic_access_authentication)

type HandlerPuller struct {
	prefix   string   // combine with name to get file path or URL
	dirCache sync.Map // key=lambda name, value=version, directory path
	dirMaker *common.DirMaker
}

type CacheEntry struct {
	version string // could be a timestamp for a file or web resource
	path    string // where code is extracted to a dir
}

func NewHandlerPuller(dirMaker *common.DirMaker) (cp *HandlerPuller, err error) {
	return &HandlerPuller{
		prefix:   common.Conf.Registry,
		dirMaker: dirMaker,
	}, nil
}

func (cp *HandlerPuller) isRemote() bool {
	return strings.HasPrefix(cp.prefix, "http://") || strings.HasPrefix(cp.prefix, "https://")
}

func (cp *HandlerPuller) Pull(name string) (rt_type common.RuntimeType, targetDir string, err error) {
	t := common.T0("pull-lambda")
	defer t.T1()

	matched, err := regexp.MatchString(`^[A-Za-z0-9\.\-\_]+$`, name)
	if err != nil {
		return rt_type, "", err
	} else if !matched {
		msg := "bad lambda name '%s', can only contain letters, numbers, period, dash, and underscore"
		return rt_type, "", fmt.Errorf(msg, name)
	}

	if cp.isRemote() {
		// registry type = web
		urls := []string{
			cp.prefix + "/" + name + ".tar.gz",
			cp.prefix + "/" + name + ".py",
			cp.prefix + "/" + name + ".bin",
		}

		for i := 0; i < len(urls); i++ {
			rt_type, targetDir, err = cp.pullRemoteFile(urls[i], name)
			if err == nil {
				return rt_type, targetDir, nil
			} else if err != notFound404 {
				// 404 is OK, because we just go on to check the next URLs
				return rt_type, "", err
			}
		}

		return rt_type, "", fmt.Errorf("lambda not found at any of these locations: %s", strings.Join(urls, ", "))
	} else {
		// registry type = file
		paths := []string{
			filepath.Join(cp.prefix, name) + ".tar.gz",
			filepath.Join(cp.prefix, name) + ".py",
			filepath.Join(cp.prefix, name) + ".bin",
			filepath.Join(cp.prefix, name),
		}

		for i := 0; i < len(paths); i++ {
			if _, err := os.Stat(paths[i]); !os.IsNotExist(err) {
				rt_type, targetDir, err = cp.pullLocalFile(paths[i], name)
				return rt_type, targetDir, err
			}
		}

		return rt_type, "", fmt.Errorf("lambda not found at any of these locations: %s", strings.Join(paths, ", "))
	}
}

// delete any caching associated with this handler
func (cp *HandlerPuller) Reset(name string) {
	cp.dirCache.Delete(name)
}

func (cp *HandlerPuller) pullLocalFile(src, lambdaName string) (rt_type common.RuntimeType, targetDir string, err error) {
	stat, err := os.Stat(src)
	if err != nil {
		return rt_type, "", err
	}

	if stat.Mode().IsDir() {
		log.Printf("Installing `%s` from a directory", stat.Name())

		// this is really just a debug mode, and is not
		// expected to be efficient
		targetDir = cp.dirMaker.Get(lambdaName)

		cmd := exec.Command("cp", "-r", src, targetDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return rt_type, "", fmt.Errorf("%s :: %s", err, string(output))
		}

		// Figure out runtime type
		if _, err := os.Stat(src + "/f.py"); !os.IsNotExist(err) {
			rt_type = common.RT_PYTHON
		} else if _, err := os.Stat(src + "/f.bin"); !os.IsNotExist(err) {
			rt_type = common.RT_NATIVE
		} else {
			return rt_type, "", fmt.Errorf("Unknown runtime type")
		}

		return rt_type, targetDir, nil
	} else if !stat.Mode().IsRegular() {
		return rt_type, "", fmt.Errorf("%s not a file or directory", src)
	}

	// for regular files, we cache based on mod time.  We don't
	// cache at the file level if this is a remote store (because
	// caching is handled at the web level)
	version := stat.ModTime().String()
	if !cp.isRemote() {
		cacheEntry := cp.getCache(lambdaName)
		if cacheEntry != nil && cacheEntry.version == version {
			// hit:
			return rt_type, cacheEntry.path, nil
		}
	}

	// miss:
	targetDir = cp.dirMaker.Get(lambdaName)
	if err := os.Mkdir(targetDir, os.ModeDir); err != nil {
		return rt_type, "", err
	} else {
		log.Printf("Created new directory for lambda function at `%s`", targetDir)
	}

	// Make sure we include the suffix
	if strings.HasSuffix(stat.Name(), ".py") {
		log.Printf("Installing `%s` from a python file", src)

		cmd := exec.Command("cp", src, filepath.Join(targetDir, "f.py"))
		rt_type = common.RT_PYTHON

		if output, err := cmd.CombinedOutput(); err != nil {
			return rt_type, "", fmt.Errorf("%s :: %s", err, string(output))
		}
	} else if strings.HasSuffix(stat.Name(), ".bin") {
		log.Printf("Installing `%s` from binary file", src)

		cmd := exec.Command("cp", src, filepath.Join(targetDir, "f.bin"))
		rt_type = common.RT_NATIVE

		if output, err := cmd.CombinedOutput(); err != nil {
			return rt_type, "", fmt.Errorf("%s :: %s", err, string(output))
		}
	} else if strings.HasSuffix(stat.Name(), ".tar.gz") {
		log.Printf("Installing `%s` from an archive file", src)

		cmd := exec.Command("tar", "-xzf", src, "--directory", targetDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return rt_type, "", fmt.Errorf("%s :: %s", err, string(output))
		}

		// Figure out runtime type
		if _, err := os.Stat(targetDir + "f.py"); !os.IsNotExist(err) {
			rt_type = common.RT_PYTHON
		} else if _, err := os.Stat(targetDir + "/f.bin"); !os.IsNotExist(err) {
			rt_type = common.RT_NATIVE
		} else {
			return rt_type, "", fmt.Errorf("Found unknown runtime type or no code at all")
		}
	} else {
		return rt_type, "", fmt.Errorf("lambda file %s not a .tar.gz or .py", src)
	}

	if !cp.isRemote() {
		cp.putCache(lambdaName, version, targetDir)
	}

	return rt_type, targetDir, nil
}

func (cp *HandlerPuller) pullRemoteFile(src, lambdaName string) (rt_type common.RuntimeType, targetDir string, err error) {
	// grab latest lambda code if it's changed (pass
	// If-Modified-Since so this can be determined on server side
	client := &http.Client{}
	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return rt_type, "", err
	}

	cacheEntry := cp.getCache(lambdaName)
	if cacheEntry != nil {
		req.Header.Set("If-Modified-Since", cacheEntry.version)
	}

	resp, err := client.Do(req)
	if err != nil {
		return rt_type, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return rt_type, "", notFound404
	}

	if resp.StatusCode == http.StatusNotModified {
		return rt_type, cacheEntry.path, nil
	}

	// download to local file, then use pullLocalFile to finish
	dir, err := ioutil.TempDir("", "ol-")
	if err != nil {
		return rt_type, "", err
	}
	defer os.RemoveAll(dir)

	parts := strings.Split(src, "/")
	localPath := filepath.Join(dir, parts[len(parts)-1])
	out, err := os.Create(localPath)
	if err != nil {
		return rt_type, "", err
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return rt_type, "", err
	}

	rt_type, targetDir, err = cp.pullLocalFile(localPath, lambdaName)

	// record directory in cache, by mod time
	if err == nil {
		version := resp.Header.Get("Last-Modified")
		if version != "" {
			cp.putCache(lambdaName, version, targetDir)
		}
	}

	return rt_type, targetDir, err
}

func (cp *HandlerPuller) getCache(name string) *CacheEntry {
	entry, found := cp.dirCache.Load(name)
	if !found {
		return nil
	}
	return entry.(*CacheEntry)
}

func (cp *HandlerPuller) putCache(name, version, path string) {
	cp.dirCache.Store(name, &CacheEntry{version, path})
}
