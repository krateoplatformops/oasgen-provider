package filegetter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGetFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "getfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases
	testCases := []struct {
		name        string
		src         string
		auth        *AuthConfig
		expectError bool
		setup       func() string
		validate    func(string) bool
	}{
		{
			name:        "Local file copy",
			src:         filepath.Join(tempDir, "local_source.txt"),
			auth:        nil,
			expectError: false,
			setup: func() string {
				content := "local file content"
				err := os.WriteFile(filepath.Join(tempDir, "local_source.txt"), []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to create local source file: %v", err)
				}
				return filepath.Join(tempDir, "local_source.txt")
			},
			validate: func(dst string) bool {
				content, err := os.ReadFile(dst)
				return err == nil && string(content) == "local file content"
			},
		},
		{
			name:        "Download without auth",
			auth:        nil,
			expectError: false,
			setup: func() string {
				content := "downloaded content"
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(content))
				}))
				return server.URL
			},
			validate: func(dst string) bool {
				content, err := os.ReadFile(dst)
				return err == nil && string(content) == "downloaded content"
			},
		},
		{
			name: "Download with basic auth",
			auth: &AuthConfig{
				Type:     BasicAuth,
				Username: "user",
				Password: "pass",
			},
			expectError: false,
			setup: func() string {
				content := "authenticated content"
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					username, password, ok := r.BasicAuth()
					if !ok || username != "user" || password != "pass" {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					w.Write([]byte(content))
				}))
				return server.URL
			},
			validate: func(dst string) bool {
				content, err := os.ReadFile(dst)
				return err == nil && string(content) == "authenticated content"
			},
		},
		{
			name: "Download with bearer token",
			auth: &AuthConfig{
				Type:  BearerToken,
				Token: "secret-token",
			},
			expectError: false,
			setup: func() string {
				content := "token authenticated content"
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer secret-token" {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					w.Write([]byte(content))
				}))
				return server.URL
			},
			validate: func(dst string) bool {
				content, err := os.ReadFile(dst)
				return err == nil && string(content) == "token authenticated content"
			},
		},
		{
			name:        "Non-existent local file",
			src:         filepath.Join(tempDir, "non_existent.txt"),
			auth:        nil,
			expectError: true,
			setup:       func() string { return "" },
			validate:    func(string) bool { return true },
		},
		{
			name:        "Invalid URL",
			src:         "http://localhost:1",
			auth:        nil,
			expectError: true,
			setup:       func() string { return "" },
			validate:    func(string) bool { return true },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src := tc.setup()
			if src != "" {
				tc.src = src
			}
			dst := filepath.Join(tempDir, "destination.txt")

			err := GetFile(dst, tc.src, tc.auth)

			fmt.Println("source:", tc.src)

			if tc.expectError && err == nil {
				t.Errorf("Expected an error, but got none")
			} else if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if err == nil && !tc.validate(dst) {
				t.Errorf("File content validation failed")
			}
		})
	}
}
