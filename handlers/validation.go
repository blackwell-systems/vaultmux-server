package handlers

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// secretNamePattern matches alphanumeric characters, hyphens, and underscores only
	secretNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// validateSecretName checks if a secret name is valid and safe to use.
// Returns error if name is empty, too long, contains path traversal patterns,
// or includes invalid characters.
func validateSecretName(name string) error {
	// Check for empty string
	if name == "" {
		return errors.New("secret name cannot be empty")
	}

	// Check length constraints (1-255 characters)
	if len(name) > 255 {
		return errors.New("secret name cannot exceed 255 characters")
	}

	// Check for path traversal patterns
	if strings.Contains(name, "..") {
		return errors.New("secret name cannot contain '..' (path traversal)")
	}
	if strings.Contains(name, "/") {
		return errors.New("secret name cannot contain '/' (path separator)")
	}
	if strings.Contains(name, "\\") {
		return errors.New("secret name cannot contain '\\' (path separator)")
	}

	// Check character whitelist
	if !secretNamePattern.MatchString(name) {
		return errors.New("secret name can only contain alphanumeric characters, hyphens, and underscores")
	}

	return nil
}
