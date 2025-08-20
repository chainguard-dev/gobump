package run

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindGoWork(t *testing.T) {
	testCases := []struct {
		name         string
		setupFunc    func(string) error
		goWorkEnv    string
		expectedPath string
	}{
		{
			name: "find go.work in current directory",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.21\n"), 0600)
			},
			goWorkEnv:    "",
			expectedPath: "go.work",
		},
		{
			name: "find go.work in parent directory",
			setupFunc: func(dir string) error {
				subdir := filepath.Join(dir, "subdir")
				if err := os.Mkdir(subdir, 0750); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.22\n"), 0600)
			},
			goWorkEnv:    "",
			expectedPath: "../go.work",
		},
		{
			name:         "no go.work file found",
			setupFunc:    func(_ string) error { return nil },
			goWorkEnv:    "",
			expectedPath: "",
		},
		{
			name: "GOWORK=off disables workspace",
			setupFunc: func(dir string) error {
				// Create go.work file but GOWORK=off should ignore it
				return os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.23\n"), 0600)
			},
			goWorkEnv:    "off",
			expectedPath: "",
		},
		{
			name:         "GOWORK points to specific file",
			setupFunc:    func(_ string) error { return nil },
			goWorkEnv:    "/custom/path/go.work",
			expectedPath: "/custom/path/go.work",
		},
		{
			name: "GOWORK=auto searches for go.work file",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.25\n"), 0600)
			},
			goWorkEnv:    "auto",
			expectedPath: "go.work",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Setup test environment
			if tc.setupFunc != nil {
				if err := tc.setupFunc(tmpDir); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Set GOWORK environment variable if needed
			if tc.goWorkEnv != "" {
				oldGoWork := os.Getenv("GOWORK")
				if err := os.Setenv("GOWORK", tc.goWorkEnv); err != nil {
					t.Fatalf("Failed to set GOWORK: %v", err)
				}
				defer func() {
					if err := os.Setenv("GOWORK", oldGoWork); err != nil {
						t.Logf("Failed to restore GOWORK: %v", err)
					}
				}()
			}

			// Change to test directory or subdirectory
			workDir := tmpDir
			if strings.Contains(tc.name, "parent") {
				workDir = filepath.Join(tmpDir, "subdir")
			}

			// Test findGoWork
			result := findGoWork(workDir)

			// Verify result
			switch {
			case tc.expectedPath == "":
				if result != "" {
					t.Errorf("Expected no go.work file, got %q", result)
				}
			case tc.goWorkEnv == "/custom/path/go.work":
				if result != tc.expectedPath {
					t.Errorf("Expected %q, got %q", tc.expectedPath, result)
				}
			default:
				// For relative paths, check if the result is non-empty for found files
				if tc.expectedPath == "go.work" || tc.expectedPath == "../go.work" {
					if result == "" {
						t.Errorf("Expected to find go.work file, but got empty result")
					}
				}
			}
		})
	}
}

func TestGoVendorDecisionLogic(t *testing.T) {
	testCases := []struct {
		name             string
		forceWork        bool
		goWorkExists     bool
		expectedWorkMode bool // true = go work vendor, false = go mod vendor
	}{
		{
			name:             "use go mod vendor when no work file and forceWork false",
			forceWork:        false,
			goWorkExists:     false,
			expectedWorkMode: false,
		},
		{
			name:             "use go work vendor when work file exists",
			forceWork:        false,
			goWorkExists:     true,
			expectedWorkMode: true,
		},
		{
			name:             "force go work vendor when forceWork is true",
			forceWork:        true,
			goWorkExists:     false,
			expectedWorkMode: true,
		},
		{
			name:             "use go work vendor when both forceWork and work file exist",
			forceWork:        true,
			goWorkExists:     true,
			expectedWorkMode: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tmpDir := t.TempDir()

			// Create go.work file if needed
			if tc.goWorkExists {
				workFile := filepath.Join(tmpDir, "go.work")
				if err := os.WriteFile(workFile, []byte("go 1.21\n\nuse .\n"), 0600); err != nil {
					t.Fatalf("Failed to create go.work file: %v", err)
				}
			}

			// Test the decision logic
			// This mirrors the logic in GoVendor function
			useWorkMode := tc.forceWork || findGoWork(tmpDir) != ""

			if useWorkMode != tc.expectedWorkMode {
				t.Errorf("Expected work mode %v, got %v", tc.expectedWorkMode, useWorkMode)
			}
		})
	}
}
