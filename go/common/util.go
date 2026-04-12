package common

import (
	"os"
	"path/filepath"
)

// DirSize returns the total size of all files in a directory tree, in bytes.
// It follows the directory recursively but skips non-regular files.
func DirSize(dirPath string) (int64, error) {
	var total int64
	err := filepath.Walk(dirPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}
