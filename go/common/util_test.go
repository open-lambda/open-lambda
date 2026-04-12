package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirSize_Empty(t *testing.T) {
	dir := t.TempDir()
	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 0 {
		t.Fatalf("expected 0, got %d", size)
	}
}

func TestDirSize_KnownFiles(t *testing.T) {
	dir := t.TempDir()

	// Create two files with known sizes
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), make([]byte, 1024), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.txt"), make([]byte, 2048), 0600); err != nil {
		t.Fatal(err)
	}

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := int64(1024 + 2048)
	if size != expected {
		t.Fatalf("expected %d, got %d", expected, size)
	}
}

func TestDirSize_NonExistent(t *testing.T) {
	_, err := DirSize("/tmp/does-not-exist-ol-test")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}
