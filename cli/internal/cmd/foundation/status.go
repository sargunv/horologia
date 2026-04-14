package foundationcmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

// Output is the JSON response for `horo status`.
type Output struct {
	Server string `json:"server"`
	Health struct {
		OK         bool   `json:"ok"`
		Status     string `json:"status,omitempty"`
		Error      string `json:"error,omitempty"`
		DurationMS int64  `json:"durationMs"`
	} `json:"health"`
	Auth struct {
		Configured bool          `json:"configured"`
		OK         bool          `json:"ok"`
		Skipped    bool          `json:"skipped"`
		Reason     string        `json:"reason,omitempty"`
		DurationMS int64         `json:"durationMs,omitempty"`
		User       *runtime.User `json:"user,omitempty"`
	} `json:"auth"`
}

// NewStatus builds the diagnostic status command.
func NewStatus(flags *support.RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Verify server connectivity and authentication",
		Long: `Verify that the CLI can reach the configured Horologia server and that the
authentication token is valid. Reports server health-check status and latency,
and the authenticated user when a token is configured. Useful for debugging
connection or credential problems after initial setup.`,
		Example: `  # Check connectivity and auth against the configured server
  horo status

  # Check connectivity and output results as JSON
  horo status --json`,
		GroupID: "foundation",
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			out := Output{Server: app.Config.ServerString()}

			healthStart := time.Now()
			health, err := app.Health(cmd.Context())
			if err != nil {
				return err
			}
			out.Health.OK = health.Status == "ok"
			out.Health.Status = health.Status
			out.Health.Error = health.Error
			out.Health.DurationMS = time.Since(healthStart).Milliseconds()

			if app.Config.HasToken() {
				if app.API == nil {
					return runtime.MissingServerError()
				}

				authStart := time.Now()
				user, err := app.API.UsersMe(cmd.Context())
				if err != nil {
					return runtime.NormalizeError(err)
				}
				out.Auth.Configured = true
				out.Auth.OK = true
				out.Auth.DurationMS = time.Since(authStart).Milliseconds()
				out.Auth.User = user
			} else {
				out.Auth.Configured = false
				out.Auth.Skipped = true
				out.Auth.Reason = "token not configured"
			}

			if app.Config.JSON {
				return app.PrintJSON(out)
			}

			app.Printf("Server: %s\n", out.Server)
			app.Printf("Health: %s (%dms)\n", out.Health.Status, out.Health.DurationMS)
			if out.Auth.Skipped {
				app.Printf("API auth: skipped (%s)\n", out.Auth.Reason)
				return nil
			}

			app.Printf(
				"API auth: ok as %s <%s> (%dms)\n",
				out.Auth.User.Name,
				out.Auth.User.Email,
				out.Auth.DurationMS,
			)
			return nil
		}),
	}

	return cmd
}
