package api

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	apigen "github.com/sargunv/tend/server/api/gen"
	"github.com/sargunv/tend/server/internal/apierrors"
	"github.com/sargunv/tend/server/internal/taskengine"
)

// Handler implements the generated API interface.
type Handler struct {
	apigen.UnimplementedHandler
	DB     *sql.DB
	Log    *slog.Logger
	Engine *taskengine.Engine
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

func isSQLiteCode(err error, code int) bool {
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == code
}

func isUniqueViolation(err error) bool {
	return isSQLiteCode(err, sqlite3.SQLITE_CONSTRAINT_UNIQUE) ||
		isSQLiteCode(err, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY)
}

func isForeignKeyViolation(err error) bool {
	return isSQLiteCode(err, sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY)
}

var (
	badRequest   = apierrors.BadRequest
	isBadRequest = apierrors.IsBadRequest
	forbidden    = apierrors.Forbidden
	isForbidden  = apierrors.IsForbidden
)

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
