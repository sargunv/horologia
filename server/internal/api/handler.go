package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/sargunv/tend/server/internal/api/gen"
)

// Handler implements the generated API interface.
// All methods currently return "not implemented" via the embedded stub.
type Handler struct {
	gen.UnimplementedHandler
	Log *slog.Logger
}

func (h *Handler) NewError(ctx context.Context, err error) *gen.ApiErrorStatusCode {
	h.Log.ErrorContext(ctx, "handler error", "error", err)
	return &gen.ApiErrorStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response: gen.ApiError{
			Code:    "internal_error",
			Message: "an internal error occurred",
		},
	}
}

// NewServer creates an HTTP handler from the API spec.
func NewServer(handler gen.Handler, log *slog.Logger) (http.Handler, error) {
	return gen.NewServer(handler, gen.WithErrorHandler(errorHandler(log)))
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
