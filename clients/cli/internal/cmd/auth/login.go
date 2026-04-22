package authcmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

type loginResult struct {
	Server string       `json:"server"`
	User   *apigen.User `json:"user"`
}

const envNoBrowser = "HOROLOGIA_NO_BROWSER"

var openBrowser = runtime.OpenBrowser

func newLoginCmd(flags *support.RootFlags) *cobra.Command {
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with the browser-based OAuth flow",
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			if !app.Config.HasServer() {
				return runtime.MissingServerError()
			}

			verifier, err := runtime.GenerateCodeVerifier()
			if err != nil {
				return err
			}
			state, err := runtime.GenerateOAuthState()
			if err != nil {
				return err
			}

			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return fmt.Errorf("listen for oauth callback: %w", err)
			}
			defer func() { _ = ln.Close() }()

			redirectURI := "http://" + ln.Addr().String() + "/oauth/callback"
			resultCh := make(chan struct {
				code string
				err  error
			}, 1)

			srv := &http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if errText := r.URL.Query().Get("error"); errText != "" {
						http.Error(w, errText, http.StatusBadRequest)
						resultCh <- struct {
							code string
							err  error
						}{err: fmt.Errorf("authorization failed: %s", errText)}
						return
					}
					if r.URL.Query().Get("state") != state {
						http.Error(w, "state mismatch", http.StatusBadRequest)
						resultCh <- struct {
							code string
							err  error
						}{err: errors.New("oauth state mismatch")}
						return
					}
					code := r.URL.Query().Get("code")
					if code == "" {
						http.Error(w, "missing code", http.StatusBadRequest)
						resultCh <- struct {
							code string
							err  error
						}{err: errors.New("oauth callback missing code")}
						return
					}
					_, _ = w.Write([]byte("Authentication complete. You can close this window."))
					resultCh <- struct {
						code string
						err  error
					}{code: code}
				}),
				ReadHeaderTimeout: 5 * time.Second,
			}
			go func() { _ = srv.Serve(ln) }()
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(cmd.Context()), 5*time.Second)
				defer cancel()
				_ = srv.Shutdown(shutdownCtx)
			}()

			authURL := resolveAuthorizeURL(app.Config.Server, redirectURI, state, verifier)
			printAuthorizationURL(app, authURL)
			if !noBrowser && !envTruthy(os.Getenv(envNoBrowser)) {
				if err := openBrowser(authURL); err != nil {
					app.Printf("Browser launch failed; continue with the URL above.\n")
				}
			}

			var callback struct {
				code string
				err  error
			}
			select {
			case callback = <-resultCh:
			case <-time.After(2 * time.Minute):
				return errors.New("timed out waiting for browser authorization")
			case <-cmd.Context().Done():
				return cmd.Context().Err()
			}
			if callback.err != nil {
				return callback.err
			}

			creds, err := runtime.ExchangeAuthorizationCode(cmd.Context(), app.Config.Server, redirectURI, callback.code, verifier)
			if err != nil {
				return err
			}
			if err := runtime.SaveOAuthCredentials(app.Config.ServerString(), creds); err != nil {
				return err
			}
			app.SetOAuthCredentials(creds)

			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			user, err := api.UsersMe(cmd.Context())
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(loginResult{Server: app.Config.ServerString(), User: user})
			}
			app.Printf("Server:   %s\n", app.Config.ServerString())
			app.Printf("Identity: %s <%s>\n", user.Name, user.Email)
			return nil
		}),
	}
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the authorization URL without attempting to open a browser")
	return cmd
}

func resolveAuthorizeURL(server *url.URL, redirectURI, state, verifier string) string {
	authURL := resolveServerURL(server, "/oauth/authorize")
	q := authURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", "horologia-cli")
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", runtime.OAuthScopeString())
	q.Set("state", state)
	q.Set("code_challenge", runtime.CodeChallengeS256(verifier))
	q.Set("code_challenge_method", "S256")
	q.Set("resource", resolveServerURL(server, "/api").String())
	authURL.RawQuery = q.Encode()
	return authURL.String()
}

func resolveServerURL(base *url.URL, refPath string) *url.URL {
	u := *base
	cleanBase := strings.TrimRight(u.Path, "/")
	u.Path = path.Join(cleanBase, refPath)
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return &u
}

func printAuthorizationURL(app *runtime.App, authURL string) {
	app.Printf("Open this URL in your browser to continue:\n%s\n", authURL)
}

func envTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
