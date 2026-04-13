package authcmd

import (
	apigen "github.com/sargunv/horologia/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

type statusOutput struct {
	Server struct {
		Configured bool                `json:"configured"`
		Value      string              `json:"value,omitempty"`
		Source     runtime.ValueSource `json:"source"`
	} `json:"server"`
	Token struct {
		Configured bool                `json:"configured"`
		Preview    string              `json:"preview,omitempty"`
		Source     runtime.ValueSource `json:"source"`
	} `json:"token"`
	Identity struct {
		Checked bool         `json:"checked"`
		OK      bool         `json:"ok"`
		Skipped bool         `json:"skipped"`
		Reason  string       `json:"reason,omitempty"`
		User    *apigen.User `json:"user,omitempty"`
	} `json:"identity"`
}

func newStatusCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		Long: `Display the configured server URL and its source, a redacted token
preview, and the authenticated user. If the server or token is missing,
the identity check is skipped with a reason.`,
		Example: `  # Check authentication status
  horo auth status

  # Check authentication status as JSON
  horo auth status --json`,
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			out := buildStatusOutput(app)

			if out.Token.Configured && out.Server.Configured {
				api, err := support.RequireAPI(app)
				if err != nil {
					return err
				}

				user, err := api.UsersMe(cmd.Context())
				if err != nil {
					return runtime.NormalizeError(err)
				}
				out.Identity.Checked = true
				out.Identity.OK = true
				out.Identity.User = user
			} else {
				out.Identity.Skipped = true
				if !out.Token.Configured {
					out.Identity.Reason = "token not configured"
				} else {
					out.Identity.Reason = "server not configured"
				}
			}

			if app.Config.JSON {
				return app.PrintJSON(out)
			}

			serverLine := "<unset>"
			if out.Server.Configured {
				serverLine = out.Server.Value
			}
			tokenLine := "not configured"
			if out.Token.Configured {
				tokenLine = out.Token.Preview
			}

			app.Printf("Server:   %s (%s)\n", serverLine, out.Server.Source)
			app.Printf("Token:    %s (%s)\n", tokenLine, out.Token.Source)
			if out.Identity.Skipped {
				app.Printf("Identity: skipped (%s)\n", out.Identity.Reason)
				return nil
			}

			app.Printf(
				"Identity: %s <%s>\n",
				out.Identity.User.Name,
				out.Identity.User.Email,
			)
			return nil
		}),
	}
}

func buildStatusOutput(app *runtime.App) statusOutput {
	var out statusOutput
	out.Server.Configured = app.Config.HasServer()
	out.Server.Value = app.Config.ServerString()
	out.Server.Source = app.Config.ServerSource
	out.Token.Configured = app.Config.HasToken()
	out.Token.Preview = app.Config.RedactedToken()
	out.Token.Source = app.Config.TokenSource
	return out
}
