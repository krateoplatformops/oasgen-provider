package filegetter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AuthType represents the type of authentication
type AuthType int

const (
	NoAuth AuthType = iota
	BasicAuth
	BearerToken
)

// AuthConfig holds authentication information
type AuthConfig struct {
	Type     AuthType
	Username string
	Password string
	Token    string
}

type Filegetter struct {
	Client     *http.Client
	KubeClient client.Client
}

// GetFile gets a file from a source and writes it to a destination.
func (cli *Filegetter) GetFile(ctx context.Context, dst string, src string, auth *AuthConfig) error {
	var reader io.Reader
	var err error

	if cli.Client == nil || cli.KubeClient == nil {
		return fmt.Errorf("http client or kube client not set")
	}

	// Check if the source is a URL or a local file
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		// Create a new request
		req, err := http.NewRequest("GET", src, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}

		// Add authentication if provided
		if auth != nil {
			switch auth.Type {
			case BasicAuth:
				req.SetBasicAuth(auth.Username, auth.Password)
			case BearerToken:
				req.Header.Add("Authorization", "Bearer "+auth.Token)
			}
		}

		// Send the request
		resp, err := cli.Client.Do(req)
		if err != nil {
			return fmt.Errorf("error downloading file: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		reader = resp.Body
	} else if strings.HasPrefix(src, "configmap://") {
		configmapString := strings.TrimPrefix(src, "configmap://")
		configmapParts := strings.Split(configmapString, "/")
		if len(configmapParts) != 3 {
			return fmt.Errorf("invalid configmap source: %s - must be formatted as configmap://<namespace>/<name>/<key>", src)
		}
		namespace := configmapParts[0]
		name := configmapParts[1]
		key := configmapParts[2]

		// Get the configmap name and key
		cm := corev1.ConfigMap{}
		err = cli.KubeClient.Get(ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, &cm)
		if err != nil {
			return fmt.Errorf("error getting configmap: %v", err)
		}

		data, ok := cm.Data[key]
		if !ok {
			return fmt.Errorf("key not found in configmap: %s", key)
		}

		reader = strings.NewReader(data)
	} else {
		// Open local file
		file, err := os.Open(src)
		if err != nil {
			return fmt.Errorf("error opening local file: %v - %s", err, src)
		}
		defer file.Close()
		reader = file
	}

	// Create the destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating destination file: %v", err)
	}
	defer dstFile.Close()

	// Copy the contents
	_, err = io.Copy(dstFile, reader)
	if err != nil {
		return fmt.Errorf("error writing to destination file: %v", err)
	}

	return nil
}
