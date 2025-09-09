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
	"time"

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
	version time.Time // blob modification time
	path    string
	runtime common.RuntimeType
}

func NewHandlerPuller(dirMaker *common.DirMaker) (*HandlerPuller, error) {
	ctx := context.Background()
	storeURL := common.Conf.Registry

	// If no recognized scheme is present, assume local path and add file://
	if !strings.HasPrefix(storeURL, "file://") &&
		!strings.HasPrefix(storeURL, "s3://") &&
		!strings.HasPrefix(storeURL, "gs://") {
		storeURL = "file://" + storeURL
	}

	if strings.HasPrefix(storeURL, "file://") {
		dir := strings.TrimPrefix(storeURL, "file://")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create local lambda store directory: %w", err)
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

	key := name + common.LambdaFileExtension

	attrs, err := cp.bucket.Attributes(context.Background(), key)
	if err == nil {
		version := attrs.ModTime
		if cached := cp.getCache(name); cached != nil && cached.version.Equal(version) {
			return cached.runtime, cached.path, nil
		}
	}

	rt, dir, err := cp.pullFromBlob(key, name)
	if err == nil {
		var version time.Time
		if attrs != nil {
			version = attrs.ModTime
		}
		cp.putCache(name, version, dir, rt)
		return rt, dir, nil
	} else if err != errNotFound404 {
		return RT_UNKNOWN, "", err
	}
	return RT_UNKNOWN, "", fmt.Errorf(
		"lambda %q not found in blob store (bucket=%q, key=%q)",
		name, common.Conf.Registry, key,
	)
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
	if err := os.MkdirAll(targetDir, 0755); err != nil {
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

func (cp *HandlerPuller) getCache(name string) *CacheEntry {
	entry, found := cp.dirCache.Load(name)
	if !found {
		return nil
	}
	return entry.(*CacheEntry)
}
func (cp *HandlerPuller) putCache(name string, version time.Time, path string, runtime common.RuntimeType) {
	// Clean up old cache entry if it exists
	if old := cp.getCache(name); old != nil && old.path != path {
		os.RemoveAll(old.path)
	}
	cp.dirCache.Store(name, &CacheEntry{version, path, runtime})
}
