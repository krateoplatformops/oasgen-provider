package filegetter

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

// GetFile gets a file from a source and writes it to a destination.
func GetFile(dst string, src string, auth *AuthConfig) error {
	var reader io.Reader
	var err error

	// Check if the source is a URL or a local file
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		// Create a new HTTP client
		client := &http.Client{}

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
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error downloading file: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		reader = resp.Body
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
