package auth

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var supportedScopes = []string{
	"activity:read",
	"admin:read",
	"admin:write",
	"profile:read",
	"spaces:read",
	"spaces:write",
	"tags:read",
	"tags:write",
	"tasks:read",
	"tasks:write",
	"users:read",
	"users:write",
}

// SupportedScopes returns the OAuth scopes Horologia currently recognizes.
func SupportedScopes() []string {
	return append([]string(nil), supportedScopes...)
}

// NormalizeScopes validates, deduplicates, and sorts a scope string.
func NormalizeScopes(raw string) ([]string, error) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return nil, errors.New("scope is required")
	}

	seen := make(map[string]struct{}, len(fields))
	scopes := make([]string, 0, len(fields))
	for _, scope := range fields {
		if !slices.Contains(supportedScopes, scope) {
			return nil, fmt.Errorf("unsupported scope %q", scope)
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}

	slices.Sort(scopes)
	return scopes, nil
}

// ScopeSetKey returns a stable persistence key for an exact set of scopes.
func ScopeSetKey(scopes []string) string {
	normalized := append([]string(nil), scopes...)
	slices.Sort(normalized)
	return strings.Join(normalized, " ")
}
