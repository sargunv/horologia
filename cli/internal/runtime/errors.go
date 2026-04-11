package runtime

import "fmt"

// ConfigError is a user-facing configuration problem.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

func newMissingServerError() error {
	return &ConfigError{Message: "server is not configured; set TEND_SERVER or pass --server"}
}

func newMissingTokenError() error {
	return &ConfigError{Message: "token is not configured; set TEND_TOKEN or pass --token"}
}

// APIError is a structured API error response.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Code != "" {
		return e.Code
	}
	return fmt.Sprintf("request failed with status %d", e.StatusCode)
}
