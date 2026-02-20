package common

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// TestReuseSandbox verifies that the reuse-sandbox field defaults to true
// when not specified, and correctly reflects the value when explicitly set.
func TestReuseSandbox(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected bool
	}{
		{
			name:     "no ol.yaml — defaults to true",
			yaml:     "",
			expected: true,
		},
		{
			name:     "ol.yaml present but reuse-sandbox is not specified - defaults to true",
			yaml:     "triggers:\n  http:\n    - method: \"GET\"\n",
			expected: true,
		},
		{
			name:     "reuse-sandbox explicitly set to false",
			yaml:     "reuse-sandbox: false\n",
			expected: false,
		},
		{
			name:     "reuse-sandbox explicitly set to true",
			yaml:     "reuse-sandbox: true\n",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.yaml != "" {
				if err := os.WriteFile(filepath.Join(dir, "ol.yaml"), []byte(tt.yaml), 0644); err != nil {
					t.Fatal(err)
				}
			}
			config, err := LoadLambdaConfig(dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if config.ReuseSandbox != tt.expected {
				t.Errorf("expected ReuseSandbox=%v, got %v", tt.expected, config.ReuseSandbox)
			}
		})
	}
}

// createTestTarGz creates a tar.gz file in memory with the given files
func createTestTarGz(t *testing.T, files map[string]string) []byte {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestExtractConfigFromTarGz_RootLevel verifies that ol.yaml at the root level is correctly extracted
// This includes both "ol.yaml" and "./ol.yaml" formats
func TestExtractConfigFromTarGz_RootLevel(t *testing.T) {
	tests := []struct {
		name       string
		fileName   string
		shouldPass bool
	}{
		{
			name:       "ol.yaml at root",
			fileName:   "ol.yaml",
			shouldPass: true,
		},
		{
			name:       "./ol.yaml at root",
			fileName:   "./ol.yaml",
			shouldPass: true, // Currently fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configYAML := `triggers:
  http:
    - method: "GET"`

			files := map[string]string{
				tt.fileName: configYAML,
				"f.py":      "def handler(event): pass",
			}

			tarData := createTestTarGz(t, files)

			// Write to temp file
			tmpFile := filepath.Join(t.TempDir(), "test.tar.gz")
			if err := os.WriteFile(tmpFile, tarData, 0644); err != nil {
				t.Fatal(err)
			}

			config, err := ExtractConfigFromTarGz(tmpFile)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(config.Triggers.HTTP) == 0 {
				t.Fatal("Expected HTTP triggers to be parsed")
			}

			actualMethod := config.Triggers.HTTP[0].Method
			if tt.shouldPass {
				// Should have parsed the config
				if actualMethod != "GET" {
					t.Errorf("Expected method GET (from config), got %s", actualMethod)
				}
			} else {
				// Should have used default config
				if actualMethod != "*" {
					t.Errorf("Expected method * (from default), got %s", actualMethod)
				}
			}
		})
	}
}
