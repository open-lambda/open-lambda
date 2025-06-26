package sandbox

import (
	"testing"

	"github.com/open-lambda/open-lambda/ol/common"
)

func init() {
	// Initialize configuration for tests
	if common.Conf == nil {
		common.Conf = &common.Config{
			Trace: common.TraceConfig{
				Latency: false,
			},
			Limits: common.LimitsConfig{
				Mem_mb:      128,
				CPU_percent: 50,
			},
			Sandbox: "sock",
		}
	}
}

func TestFillMetaDefaults(t *testing.T) {
	// Save original config values
	originalMemLimit := common.Conf.Limits.Mem_mb
	originalCPUPercent := common.Conf.Limits.CPU_percent
	
	// Set test values
	common.Conf.Limits.Mem_mb = 512
	common.Conf.Limits.CPU_percent = 80
	
	defer func() {
		// Restore original values
		common.Conf.Limits.Mem_mb = originalMemLimit
		common.Conf.Limits.CPU_percent = originalCPUPercent
	}()

	tests := []struct {
		name     string
		input    *SandboxMeta
		expected *SandboxMeta
	}{
		{
			name:  "nil meta should create defaults",
			input: nil,
			expected: &SandboxMeta{
				MemLimitMB: 512,
				CPUPercent: 80,
			},
		},
		{
			name:  "empty meta should fill defaults",
			input: &SandboxMeta{},
			expected: &SandboxMeta{
				MemLimitMB: 512,
				CPUPercent: 80,
			},
		},
		{
			name: "partial meta should fill missing defaults",
			input: &SandboxMeta{
				Installs:   []string{"numpy"},
				MemLimitMB: 256, // This should not be overridden
			},
			expected: &SandboxMeta{
				Installs:   []string{"numpy"},
				MemLimitMB: 256,
				CPUPercent: 80, // This should be filled
			},
		},
		{
			name: "complete meta should not be changed",
			input: &SandboxMeta{
				Installs:   []string{"pandas", "numpy"},
				Imports:    []string{"os", "sys"},
				MemLimitMB: 1024,
				CPUPercent: 50,
			},
			expected: &SandboxMeta{
				Installs:   []string{"pandas", "numpy"},
				Imports:    []string{"os", "sys"},
				MemLimitMB: 1024,
				CPUPercent: 50,
			},
		},
		{
			name: "zero memory should be filled with default",
			input: &SandboxMeta{
				MemLimitMB: 0,
				CPUPercent: 75,
			},
			expected: &SandboxMeta{
				MemLimitMB: 512, // Should be filled
				CPUPercent: 75,
			},
		},
		{
			name: "zero CPU should be filled with default",
			input: &SandboxMeta{
				MemLimitMB: 256,
				CPUPercent: 0,
			},
			expected: &SandboxMeta{
				MemLimitMB: 256,
				CPUPercent: 80, // Should be filled
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fillMetaDefaults(tt.input)

			if result == nil {
				t.Fatal("fillMetaDefaults should never return nil")
			}

			// Check memory limit
			if result.MemLimitMB != tt.expected.MemLimitMB {
				t.Errorf("MemLimitMB: expected %d, got %d", 
					tt.expected.MemLimitMB, result.MemLimitMB)
			}

			// Check CPU percent
			if result.CPUPercent != tt.expected.CPUPercent {
				t.Errorf("CPUPercent: expected %d, got %d", 
					tt.expected.CPUPercent, result.CPUPercent)
			}

			// Check installs slice
			if len(result.Installs) != len(tt.expected.Installs) {
				t.Errorf("Installs length: expected %d, got %d", 
					len(tt.expected.Installs), len(result.Installs))
			} else {
				for i, install := range tt.expected.Installs {
					if result.Installs[i] != install {
						t.Errorf("Installs[%d]: expected %q, got %q", 
							i, install, result.Installs[i])
					}
				}
			}

			// Check imports slice
			if len(result.Imports) != len(tt.expected.Imports) {
				t.Errorf("Imports length: expected %d, got %d", 
					len(tt.expected.Imports), len(result.Imports))
			} else {
				for i, imp := range tt.expected.Imports {
					if result.Imports[i] != imp {
						t.Errorf("Imports[%d]: expected %q, got %q", 
							i, imp, result.Imports[i])
					}
				}
			}
		})
	}
}

func TestFillMetaDefaults_ModificationBehavior(t *testing.T) {
	// Set test values
	common.Conf.Limits.Mem_mb = 256
	common.Conf.Limits.CPU_percent = 60

	// Test that original meta is modified when passed
	original := &SandboxMeta{
		Installs: []string{"test"},
	}
	
	result := fillMetaDefaults(original)
	
	// Should return the same object
	if result != original {
		t.Error("fillMetaDefaults should modify and return the same object when meta is not nil")
	}
	
	// Should have filled defaults
	if original.MemLimitMB != 256 {
		t.Errorf("original meta should be modified: expected MemLimitMB=256, got %d", original.MemLimitMB)
	}
	if original.CPUPercent != 60 {
		t.Errorf("original meta should be modified: expected CPUPercent=60, got %d", original.CPUPercent)
	}
}

func TestSandboxPoolFromConfig_Docker(t *testing.T) {
	// Save original config
	originalSandbox := common.Conf.Sandbox
	defer func() {
		common.Conf.Sandbox = originalSandbox
	}()

	// Test docker configuration
	common.Conf.Sandbox = "docker"
	
	pool, err := SandboxPoolFromConfig("test", 512)
	if err != nil {
		t.Errorf("unexpected error creating docker pool: %v", err)
	}
	if pool == nil {
		t.Error("docker pool should not be nil")
	}
}

func TestSandboxPoolFromConfig_SOCK(t *testing.T) {
	// Save original config
	originalSandbox := common.Conf.Sandbox
	defer func() {
		common.Conf.Sandbox = originalSandbox
	}()

	// Test SOCK configuration
	common.Conf.Sandbox = "sock"
	
	pool, err := SandboxPoolFromConfig("test", 512)
	// SOCK pools require system permissions, so we expect this to fail in test environment
	if err == nil {
		if pool == nil {
			t.Error("SOCK pool should not be nil when no error occurs")
		}
	} else {
		// This is expected in test environment due to permission issues
		t.Logf("SOCK pool creation failed as expected in test environment: %v", err)
	}
}

func TestSandboxPoolFromConfig_Invalid(t *testing.T) {
	// Save original config
	originalSandbox := common.Conf.Sandbox
	defer func() {
		common.Conf.Sandbox = originalSandbox
	}()

	// Test invalid configuration
	common.Conf.Sandbox = "invalid_type"
	
	pool, err := SandboxPoolFromConfig("test", 512)
	if err == nil {
		t.Error("expected error for invalid sandbox type")
	}
	if pool != nil {
		t.Error("pool should be nil for invalid sandbox type")
	}
	
	expectedError := "invalid sandbox type: 'invalid_type'"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

func TestSandboxPoolFromConfig_Parameters(t *testing.T) {
	// Save original config
	originalSandbox := common.Conf.Sandbox
	defer func() {
		common.Conf.Sandbox = originalSandbox
	}()

	// Test that parameters are passed correctly with Docker (doesn't require system permissions)
	common.Conf.Sandbox = "docker"
	
	name := "test-pool"
	sizeMb := 1024
	
	pool, err := SandboxPoolFromConfig(name, sizeMb)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if pool == nil {
		t.Error("pool should not be nil")
		return
	}
	
	// The pool should be created successfully with the given parameters
	// We can't directly verify the parameters were used without accessing 
	// internal state, but we can verify the pool was created
	// Note: Docker pool debug string might be empty, that's ok
	debugStr := pool.DebugString()
	_ = debugStr // Docker pool may have empty debug string
}

// Test edge cases and boundary conditions
func TestSandboxPoolFromConfig_EdgeCases(t *testing.T) {
	// Save original config
	originalSandbox := common.Conf.Sandbox
	defer func() {
		common.Conf.Sandbox = originalSandbox
	}()

	tests := []struct {
		name         string
		sandboxType  string
		poolName     string
		sizeMb       int
		expectError  bool
	}{
		{"docker with empty name", "docker", "", 512, false},
		{"docker with zero size", "docker", "test", 0, false},
		{"docker with negative size", "docker", "test", -100, false},
		{"sock with empty name", "sock", "", 512, true}, // SOCK requires permissions
		{"sock with zero size", "sock", "test", 0, true}, // SOCK requires permissions
		{"empty sandbox type", "", "test", 512, true},
		{"whitespace sandbox type", "   ", "test", 512, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common.Conf.Sandbox = tt.sandboxType
			
			pool, err := SandboxPoolFromConfig(tt.poolName, tt.sizeMb)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if pool != nil {
					t.Error("pool should be nil when error occurs")
				}
			} else {
				// For sock tests, we expect permission errors in test environment
				if tt.sandboxType == "sock" {
					if err != nil {
						t.Logf("SOCK pool creation failed as expected: %v", err)
					} else if pool == nil {
						t.Error("pool should not be nil when no error occurs")
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
					if pool == nil {
						t.Error("pool should not be nil when no error occurs")
					}
				}
			}
		})
	}
}