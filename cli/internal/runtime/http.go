package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	apigen "github.com/sargunv/tend/api/gen"
)

type apiErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HealthResponse is the /healthz response shape.
type HealthResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// User is the API's current-user shape.
type User = apigen.User

// App holds shared CLI runtime state.
type App struct {
	Config Config
	Stdout io.Writer
	Stderr io.Writer

	client *http.Client
	API    *apigen.Client
}

// NewApp builds the shared CLI runtime.
func NewApp(cfg Config, stdout io.Writer, stderr io.Writer) *App {
	app := &App{
		Config: cfg,
		Stdout: stdout,
		Stderr: stderr,
		client: &http.Client{},
	}

	if cfg.HasServer() {
		client, err := apigen.NewClient(
			cfg.APIBaseString(),
			securitySource{token: cfg.Token},
			apigen.WithClient(app.client),
		)
		if err != nil {
			panic(fmt.Sprintf("create generated API client: %v", err))
		}
		app.API = client
	}

	return app
}

// Health checks server liveness through /healthz.
func (a *App) Health(ctx context.Context) (HealthResponse, error) {
	if !a.Config.HasServer() {
		return HealthResponse{}, MissingServerError()
	}

	var resp HealthResponse
	if err := a.doJSON(ctx, requestSpec{
		Method: "GET",
		URL:    resolveURL(a.Config.Server, "/healthz"),
	}, &resp); err != nil {
		return HealthResponse{}, err
	}
	return resp, nil
}

type requestSpec struct {
	Method string
	URL    *url.URL
	Body   any
	Auth   bool
}

func (a *App) doJSON(parent context.Context, spec requestSpec, out any) error {
	var body io.Reader
	if spec.Body != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(spec.Body); err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		body = buf
	}

	req, err := http.NewRequestWithContext(parent, spec.Method, spec.URL.String(), body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if spec.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if spec.Auth {
		req.Header.Set("Authorization", "Bearer "+a.Config.Token)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", spec.Method, spec.URL.String(), err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp)
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}

func decodeAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read error response: %w", err)
	}

	var apiErr apiErrorResponse
	if len(body) > 0 && json.Unmarshal(body, &apiErr) == nil && (apiErr.Message != "" || apiErr.Code != "") {
		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       apiErr.Code,
			Message:    apiErr.Message,
		}
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = resp.Status
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    message,
	}
}

// NormalizeError converts generated client API errors into CLI-facing errors.
func NormalizeError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *apigen.ApiErrorStatusCode
	if stderrors.As(err, &apiErr) {
		return &APIError{
			StatusCode: apiErr.StatusCode,
			Code:       apiErr.Response.Code,
			Message:    apiErr.Response.Message,
		}
	}

	return err
}

type securitySource struct {
	token string
}

func (s securitySource) BearerAuth(ctx context.Context, operationName apigen.OperationName) (apigen.BearerAuth, error) {
	if strings.TrimSpace(s.token) == "" {
		return apigen.BearerAuth{}, MissingTokenError()
	}

	return apigen.BearerAuth{
		Token: s.token,
	}, nil
}

func resolveURL(base *url.URL, refPath string) *url.URL {
	u := *base
	cleanBase := strings.TrimRight(u.Path, "/")
	u.Path = path.Join(cleanBase, refPath)
	if strings.HasSuffix(refPath, "/") && !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return &u
}
