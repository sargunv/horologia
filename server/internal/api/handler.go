package api

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ogen-go/ogen/ogenerrors"

	apigen "github.com/sargunv/tend/server/api/gen"
)

// Handler implements the generated API interface.
type Handler struct {
	apigen.UnimplementedHandler
	DB  *sql.DB
	Log *slog.Logger
}

func (h *Handler) NewError(ctx context.Context, err error) *apigen.ApiErrorStatusCode {
	code := http.StatusInternalServerError
	apiCode := "internal_server_error"
	message := "an internal error occurred"

	var secErr *ogenerrors.SecurityError
	if errors.As(err, &secErr) {
		code = http.StatusUnauthorized
		apiCode = "unauthorized"
		message = "authentication required"
	} else if errors.Is(err, sql.ErrNoRows) {
		code = http.StatusNotFound
		apiCode = "not_found"
		message = "resource not found"
	} else if isUniqueViolation(err) {
		code = http.StatusConflict
		apiCode = "conflict"
		message = "resource already exists"
	} else if isForeignKeyViolation(err) {
		code = http.StatusBadRequest
		apiCode = "bad_request"
		message = "referenced resource does not exist"
	} else if isForbidden(err) {
		code = http.StatusForbidden
		apiCode = "forbidden"
		message = err.Error()
	} else if isBadRequest(err) {
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

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

type badRequestError struct {
	message string
}

func (e *badRequestError) Error() string {
	return e.message
}

func isBadRequest(err error) bool {
	var bre *badRequestError
	return errors.As(err, &bre)
}

func badRequest(msg string) error {
	return &badRequestError{message: msg}
}

type forbiddenError struct {
	message string
}

func (e *forbiddenError) Error() string {
	return e.message
}

func isForbidden(err error) bool {
	var fe *forbiddenError
	return errors.As(err, &fe)
}

func forbidden(msg string) error {
	return &forbiddenError{message: msg}
}

func checkDeleted(result sql.Result) error {
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
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
