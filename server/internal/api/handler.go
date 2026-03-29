package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ogen-go/ogen/ogenerrors"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// Handler implements the generated API interface.
type Handler struct {
	apigen.UnimplementedHandler
	Pool                *pgxpool.Pool
	Log                 *slog.Logger
	SecureCookies       bool
	OIDCEnabled         bool
	OIDCLabel           string
	OIDCAutoRedirect    bool
	PasswordAuthEnabled bool
}

func (h *Handler) NewError(ctx context.Context, err error) *apigen.ApiErrorStatusCode {
	code := http.StatusInternalServerError
	apiCode := "internal_server_error"
	message := "an internal error occurred"

	var secErr *ogenerrors.SecurityError
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &secErr):
		code = http.StatusUnauthorized
		apiCode = "unauthorized"
		message = "authentication required"
	case errors.Is(err, pgx.ErrNoRows):
		code = http.StatusNotFound
		apiCode = "not_found"
		message = "resource not found"
	case errors.As(err, &pgErr) && pgErr.Code == "23505": // unique_violation
		code = http.StatusConflict
		apiCode = "conflict"
		message = "resource already exists"
	case errors.As(err, &pgErr) && pgErr.Code == "23503": // foreign_key_violation
		code = http.StatusBadRequest
		apiCode = "bad_request"
		message = "referenced resource does not exist"
	case types.IsForbiddenError(err):
		code = http.StatusForbidden
		apiCode = "forbidden"
		message = err.Error()
	case types.IsValidationError(err):
		code = http.StatusBadRequest
		apiCode = "bad_request"
		message = err.Error()
	}

	if code >= 500 {
		h.Log.ErrorContext(ctx, "handler error", "error", err)
	}

	return &apigen.ApiErrorStatusCode{
		StatusCode: code,
		Response: apigen.ApiError{
			Code:    apiCode,
			Message: message,
		},
	}
}

var (
	badRequest = types.ValidationError
	forbidden  = types.ForbiddenError
)

func checkDeleted(result pgconn.CommandTag) error {
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// NewServer creates an HTTP handler from the API spec.
func NewServer(handler *Handler, log *slog.Logger) (http.Handler, error) {
	return apigen.NewServer(handler, handler, apigen.WithErrorHandler(errorHandler(log)))
}

func errorHandler(log *slog.Logger) ogenerrors.ErrorHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		attrs := []any{"error", err, "method", r.Method, "path", r.URL.Path}

		var decErr *ogenerrors.DecodeRequestError
		var secErr *ogenerrors.SecurityError
		if errors.As(err, &decErr) || errors.As(err, &secErr) {
			log.DebugContext(ctx, "client error", attrs...)
		} else {
			log.ErrorContext(ctx, "server error", attrs...)
		}

		ogenerrors.DefaultErrorHandler(ctx, w, r, err)
	}
}
