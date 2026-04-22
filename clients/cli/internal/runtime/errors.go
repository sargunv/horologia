package runtime

import "fmt"

// ConfigError is a user-facing configuration problem.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

func MissingServerError() error {
	return &ConfigError{Message: "server is not configured; set HOROLOGIA_SERVER or run `horo config set server <url>`"}
}

func MissingTokenError() error {
	return &ConfigError{Message: "token is not configured; set HOROLOGIA_TOKEN or run `horo auth login`"}
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
