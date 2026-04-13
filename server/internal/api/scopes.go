package api

import (
	"context"

	"github.com/sargunv/horologia/server/internal/auth"
)

func (h *Handler) requireScope(ctx context.Context, scope string) error {
	user := auth.UserFromContext(ctx)
	if user == nil {
		return forbidden("authentication required")
	}
	if !user.HasScope(scope) {
		return forbidden("insufficient scope")
	}
	return nil
}

func (h *Handler) requireNonDelegatedToken(ctx context.Context) error {
	user := auth.UserFromContext(ctx)
	if user == nil {
		return forbidden("authentication required")
	}
	if user.Token != nil && user.Token.IsDelegated() {
		return forbidden("delegated tokens cannot manage personal API tokens")
	}
	return nil
}
