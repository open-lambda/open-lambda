package lambda

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
	"gocloud.dev/gcerrors"

	"github.com/open-lambda/open-lambda/ol/common"
)

var errNotFound404 = errors.New("lambda not found in blob store")

var RT_UNKNOWN common.RuntimeType

type HandlerPuller struct {
	bucket   *blob.Bucket
	dirCache sync.Map // key=lambda name, value=*CacheEntry
	dirMaker *common.DirMaker
}

type CacheEntry struct {
	version string // optional: not used with blob
	path    string
}

func NewHandlerPuller(dirMaker *common.DirMaker) (*HandlerPuller, error) {
	ctx := context.Background()
	storeURL := common.Conf.Registry

	// If local, create directory if needed
	if strings.HasPrefix(storeURL, "file://") {
		dir := strings.TrimPrefix(storeURL, "file://")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create local lambda store directory %s: %w", dir, err)
		}
	}

	bucket, err := blob.OpenBucket(ctx, storeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open blob bucket: %w", err)
	}

	return &HandlerPuller{
		bucket:   bucket,
		dirMaker: dirMaker,
	}, nil
}

func (cp *HandlerPuller) Pull(name string) (common.RuntimeType, string, error) {
	t := common.T0("pull-lambda")
	defer t.T1()

	if err := common.ValidateFunctionName(name); err != nil {
		return RT_UNKNOWN, "", err
	}

	if cp.isRemote() {
		// registry type = web - only support tar.gz files
		urls := []string{
			cp.prefix + "/" + name + ".tar.gz",
		}

		for i := 0; i < len(urls); i++ {
			rt_type, targetDir, err = cp.pullRemoteFile(urls[i], name)
			if err == nil {
				return rt_type, targetDir, nil
			} else if err != errNotFound404 {
				// 404 is OK, because we just go on to check the next URLs
				return rt_type, "", err
			}
		}

		return rt_type, "", fmt.Errorf("lambda not found at any of these locations: %s", strings.Join(urls, ", "))
	}

	// registry type = file - only support tar.gz files
	paths := []string{
		filepath.Join(cp.prefix, name) + ".tar.gz",
	}

	for i := 0; i < len(paths); i++ {
		if _, err := os.Stat(paths[i]); !os.IsNotExist(err) {
			rt_type, targetDir, err = cp.pullLocalFile(paths[i], name)
			return rt_type, targetDir, err
		}
	}

	rt, dir, err := cp.pullFromBlob(key, name)
	if err == nil {
		if attrs != nil {
			cp.putCache(name, attrs.ModTime.String(), dir)
		} else {
			cp.putCache(name, "", dir)
		}
		return rt, dir, nil
	} else if err != errNotFound404 {
		return RT_UNKNOWN, "", err
	}

	return RT_UNKNOWN, "", fmt.Errorf("lambda not found in blob store for %s", name)
}

func (cp *HandlerPuller) pullFromBlob(key, lambdaName string) (common.RuntimeType, string, error) {
	ctx := context.Background()

	reader, err := cp.bucket.NewReader(ctx, key, nil)
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			return RT_UNKNOWN, "", errNotFound404
		}
		return RT_UNKNOWN, "", err
	}
	defer reader.Close()

	tmpFile, err := os.CreateTemp("", lambdaName+"_blob")
	if err != nil {
		return RT_UNKNOWN, "", err
	}
	tmpPath := tmpFile.Name()
	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return RT_UNKNOWN, "", err
	}
	tmpFile.Close()
	defer os.Remove(tmpPath)

	targetDir := cp.dirMaker.Get(lambdaName)
	if err := os.Mkdir(targetDir, 0755); err != nil {
		return RT_UNKNOWN, "", err
	}

	cmd := exec.Command("tar", "-xzf", tmpPath, "--directory", targetDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return RT_UNKNOWN, "", fmt.Errorf("tar extract failed: %v :: %s", err, output)
	}

	var rt common.RuntimeType
	if _, err := os.Stat(filepath.Join(targetDir, "f.py")); err == nil {
		rt = common.RT_PYTHON
	} else if _, err := os.Stat(filepath.Join(targetDir, "f.bin")); err == nil {
		rt = common.RT_NATIVE
	} else {
		return RT_UNKNOWN, "", fmt.Errorf("runtime type not found in extracted archive")
	}

	return rt, targetDir, nil
}

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

		err := Copy(src, targetDir)
		if err != nil {
			return rt_type, "", fmt.Errorf("%s :: %s", err)
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
	}

	log.Printf("Created new directory for lambda function at `%s`", targetDir)

	// Only support tar.gz files
	if strings.HasSuffix(stat.Name(), ".tar.gz") {
		log.Printf("Installing `%s` from an archive file", src)

		cmd := exec.Command("tar", "-xzf", src, "--directory", targetDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return rt_type, "", fmt.Errorf("%s :: %s", err, string(output))
		}

		// Validate that tar.gz contains f.py or f.bin
		if _, err := os.Stat(targetDir + "/f.py"); !os.IsNotExist(err) {
			rt_type = common.RT_PYTHON
		} else if _, err := os.Stat(targetDir + "/f.bin"); !os.IsNotExist(err) {
			rt_type = common.RT_NATIVE
		} else {
			return rt_type, "", fmt.Errorf("tar.gz file must contain f.py or f.bin")
		}
	} else {
		return rt_type, "", fmt.Errorf("lambda file %s must be a .tar.gz file", src)
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
		return rt_type, "", errNotFound404
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
