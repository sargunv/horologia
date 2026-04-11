package authcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newLogoutCmd(flags *support.RootFlags) *cobra.Command {
	type output struct {
		Server  string `json:"server,omitempty"`
		Cleared bool   `json:"cleared"`
	}

	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear stored OAuth credentials",
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			if !app.Config.HasServer() {
				return runtime.MissingServerError()
			}

			creds := app.OAuthCredentials()
			if creds != nil {
				_ = runtime.RevokeOAuthToken(cmd.Context(), app.Config.Server, creds.RefreshToken)
			}
			if err := runtime.DeleteOAuthCredentials(app.Config.ServerString()); err != nil {
				return err
			}
			app.ClearOAuthCredentials()

			if app.Config.JSON {
				return app.PrintJSON(output{Server: app.Config.ServerString(), Cleared: true})
			}
			app.Printf("Cleared stored credentials for %s\n", app.Config.ServerString())
			return nil
		}),
	}
}
