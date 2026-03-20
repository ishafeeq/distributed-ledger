package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	dockerSecretPath = "/run/secrets"
	localSecretPath  = "secrets"
)

// LoadSecret reads a secret from Docker Secrets path or local fallback.
func LoadSecret(name string) (string, error) {
	// 1. Try Docker Secret
	path := filepath.Join(dockerSecretPath, name)
	data, err := os.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// 2. Try Local Fallback
	path = filepath.Join(localSecretPath, name+".txt")
	data, err = os.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	return "", fmt.Errorf("secret %s not found in %s or %s", name, dockerSecretPath, localSecretPath)
}
