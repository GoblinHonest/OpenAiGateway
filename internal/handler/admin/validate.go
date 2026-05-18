package admin

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	maxNameLen        = 255
	maxDescriptionLen = 4096
	maxURLLen         = 2048
	maxIDLen          = 64
	maxTokenValueLen  = 8192
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// ValidationError represents a validation error with field context.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
}

// validateName checks that a name field is non-empty and within length limits.
func validateName(name string) *ValidationError {
	name = strings.TrimSpace(name)
	if name == "" {
		return &ValidationError{Field: "name", Message: "must not be empty"}
	}
	if len(name) > maxNameLen {
		return &ValidationError{Field: "name", Message: fmt.Sprintf("must be at most %d characters", maxNameLen)}
	}
	return nil
}

// validateOptionalName checks an optional name field (may be empty, but if present must be within limits).
func validateOptionalName(name string) *ValidationError {
	if name == "" {
		return nil
	}
	if len(name) > maxNameLen {
		return &ValidationError{Field: "name", Message: fmt.Sprintf("must be at most %d characters", maxNameLen)}
	}
	return nil
}

// validateDescription checks an optional description field length.
func validateDescription(desc string) *ValidationError {
	if len(desc) > maxDescriptionLen {
		return &ValidationError{Field: "description", Message: fmt.Sprintf("must be at most %d characters", maxDescriptionLen)}
	}
	return nil
}

// validateURL checks that a URL field is a valid URL (if non-empty).
func validateURL(rawURL string, fieldName string) *ValidationError {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	if len(rawURL) > maxURLLen {
		return &ValidationError{Field: fieldName, Message: fmt.Sprintf("must be at most %d characters", maxURLLen)}
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return &ValidationError{Field: fieldName, Message: "must be a valid URL"}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return &ValidationError{Field: fieldName, Message: "URL scheme must be http or https"}
	}
	return nil
}

// validateRequiredURL checks that a URL field is non-empty and valid.
func validateRequiredURL(rawURL string, fieldName string) *ValidationError {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return &ValidationError{Field: fieldName, Message: "must not be empty"}
	}
	return validateURL(rawURL, fieldName)
}

// validateID checks that an ID field is non-empty, within length limits, and matches the expected pattern.
func validateID(id string, fieldName string) *ValidationError {
	id = strings.TrimSpace(id)
	if id == "" {
		return &ValidationError{Field: fieldName, Message: "must not be empty"}
	}
	if len(id) > maxIDLen {
		return &ValidationError{Field: fieldName, Message: fmt.Sprintf("must be at most %d characters", maxIDLen)}
	}
	if !idPattern.MatchString(id) {
		return &ValidationError{Field: fieldName, Message: "must contain only alphanumeric characters, hyphens, and underscores"}
	}
	return nil
}

// validateOptionalID checks an optional ID field.
func validateOptionalID(id string, fieldName string) *ValidationError {
	if id == "" {
		return nil
	}
	return validateID(id, fieldName)
}

// validateJSON checks that a map is valid JSON (it already is since it comes from JSON binding,
// but we validate it's not excessively large).
func validateJSON(m map[string]any, fieldName string) *ValidationError {
	if m == nil {
		return nil
	}
	// Marshal to check size
	data, err := json.Marshal(m)
	if err != nil {
		return &ValidationError{Field: fieldName, Message: "must be valid JSON"}
	}
	if len(data) > 65536 { // 64KB max for config JSON
		return &ValidationError{Field: fieldName, Message: "JSON configuration is too large (max 64KB)"}
	}
	return nil
}

// validateIDs checks a slice of ID strings.
func validateIDs(ids []string, fieldName string) *ValidationError {
	for _, id := range ids {
		if err := validateID(id, fieldName); err != nil {
			return err
		}
	}
	return nil
}
