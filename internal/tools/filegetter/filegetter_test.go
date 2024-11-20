package filegetter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "getfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	var kubeClient client.Client

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	kubeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
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
		{
			name:        "ConfigMap source",
			src:         "configmap://default/test-configmap/test-key",
			auth:        nil,
			expectError: false,
			setup: func() string {
				scheme := runtime.NewScheme()
				corev1.AddToScheme(scheme)
				kubeClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-configmap",
						Namespace: "default",
					},
					Data: map[string]string{
						"test-key": "configmap content",
					},
				}).Build()

				return "configmap://default/test-configmap/test-key"
			},
			validate: func(dst string) bool {
				content, err := os.ReadFile(dst)
				return err == nil && string(content) == "configmap content"
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src := tc.setup()
			if src != "" {
				tc.src = src
			}
			dst := filepath.Join(tempDir, "destination.txt")

			filegetter := &Filegetter{
				Client:     &http.Client{},
				KubeClient: kubeClient,
			}

			err := filegetter.GetFile(context.Background(), dst, tc.src, tc.auth)

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
