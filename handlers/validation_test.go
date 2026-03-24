package handlers

import (
	"strings"
	"testing"
)

func TestValidateSecretName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		// Valid names
		{
			name:      "valid hyphenated name",
			input:     "my-secret",
			wantError: false,
		},
		{
			name:      "valid underscore name",
			input:     "secret_123",
			wantError: false,
		},
		{
			name:      "valid mixed case",
			input:     "MySecret",
			wantError: false,
		},
		{
			name:      "valid single character",
			input:     "a",
			wantError: false,
		},
		{
			name:      "valid max length (255 chars)",
			input:     "a" + strings.Repeat("b", 254),
			wantError: false,
		},
		{
			name:      "valid hyphen only",
			input:     "-",
			wantError: false,
		},
		{
			name:      "valid underscore only",
			input:     "_",
			wantError: false,
		},
		{
			name:      "valid numeric only",
			input:     "123",
			wantError: false,
		},
		{
			name:      "valid complex alphanumeric",
			input:     "---___123___---",
			wantError: false,
		},

		// Invalid: empty
		{
			name:      "invalid empty string",
			input:     "",
			wantError: true,
			errorMsg:  "secret name cannot be empty",
		},

		// Invalid: length
		{
			name:      "invalid too long (256 chars)",
			input:     strings.Repeat("a", 256),
			wantError: true,
			errorMsg:  "secret name cannot exceed 255 characters",
		},

		// Invalid: path traversal
		{
			name:      "invalid path traversal ../etc/passwd",
			input:     "../etc/passwd",
			wantError: true,
			errorMsg:  "secret name cannot contain '..' (path traversal)",
		},
		{
			name:      "invalid path traversal ../../secret",
			input:     "../../secret",
			wantError: true,
			errorMsg:  "secret name cannot contain '..' (path traversal)",
		},
		{
			name:      "invalid path traversal foo/../bar",
			input:     "foo/../bar",
			wantError: true,
			errorMsg:  "secret name cannot contain '..' (path traversal)",
		},
		{
			name:      "invalid backslash path traversal",
			input:     "foo\\bar",
			wantError: true,
			errorMsg:  "secret name cannot contain '\\' (path separator)",
		},
		{
			name:      "invalid forward slash",
			input:     "my/secret",
			wantError: true,
			errorMsg:  "secret name cannot contain '/' (path separator)",
		},

		// Invalid: special characters
		{
			name:      "invalid space character",
			input:     "my secret",
			wantError: true,
			errorMsg:  "secret name can only contain alphanumeric characters, hyphens, and underscores",
		},
		{
			name:      "invalid @ character",
			input:     "secret@123",
			wantError: true,
			errorMsg:  "secret name can only contain alphanumeric characters, hyphens, and underscores",
		},
		{
			name:      "invalid # character",
			input:     "secret#123",
			wantError: true,
			errorMsg:  "secret name can only contain alphanumeric characters, hyphens, and underscores",
		},
		{
			name:      "invalid dot character",
			input:     "my.secret",
			wantError: true,
			errorMsg:  "secret name can only contain alphanumeric characters, hyphens, and underscores",
		},
		{
			name:      "invalid dollar sign",
			input:     "secret$var",
			wantError: true,
			errorMsg:  "secret name can only contain alphanumeric characters, hyphens, and underscores",
		},
		{
			name:      "invalid percent sign",
			input:     "secret%20",
			wantError: true,
			errorMsg:  "secret name can only contain alphanumeric characters, hyphens, and underscores",
		},
		{
			name:      "invalid asterisk",
			input:     "secret*",
			wantError: true,
			errorMsg:  "secret name can only contain alphanumeric characters, hyphens, and underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecretName(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("validateSecretName(%q) expected error, got nil", tt.input)
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("validateSecretName(%q) error = %q, want %q", tt.input, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateSecretName(%q) unexpected error: %v", tt.input, err)
				}
			}
		})
	}
}
