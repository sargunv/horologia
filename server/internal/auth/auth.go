package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"

	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
)

// HashToken returns the hex-encoded SHA-256 of the given token string.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

type contextKey int

const contextKeyUser contextKey = iota

// TokenInfo identifies the API token used for authentication, if any.
type TokenInfo struct {
	ID       int64
	Name     string
	Kind     dbgen.AuthTokenKind
	ClientID string
	Scopes   []string
	Resource string
}

// User is the authenticated user attached to the request context.
type User struct {
	ID               int64
	Email            string
	Name             string
	IsOwner          bool
	Token            *TokenInfo // nil for session tokens
	SessionTokenHash string     // set for session tokens, empty for API tokens
}

// IsDelegated returns true when the request is authenticated with an OAuth
// access token whose permissions must be scope-limited.
func (t *TokenInfo) IsDelegated() bool {
	return t != nil && t.Kind == dbgen.AuthTokenKindOauthAccess
}

// HasScope reports whether the authenticated principal can use the given
// delegated scope. Non-delegated credentials remain full-trust.
func (u *User) HasScope(scope string) bool {
	if u == nil || scope == "" || u.Token == nil || !u.Token.IsDelegated() {
		return true
	}
	return slices.Contains(u.Token.Scopes, scope)
}

// UserFromContext retrieves the authenticated user from the context.
// Returns nil if unauthenticated.
func UserFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(contextKeyUser).(*User)
	return u
}

// ContextWithUser returns a new context with the given user attached.
func ContextWithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, contextKeyUser, u)
}
