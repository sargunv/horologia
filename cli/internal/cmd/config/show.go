package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

// Output is the JSON response for `horo config show`.
type Output struct {
	Server struct {
		Value  string              `json:"value"`
		Source runtime.ValueSource `json:"source"`
	} `json:"server"`
	APIBase struct {
		Value string `json:"value"`
	} `json:"apiBase"`
	Token struct {
		Configured bool                `json:"configured"`
		Preview    string              `json:"preview,omitempty"`
		Source     runtime.ValueSource `json:"source"`
	} `json:"token"`
	Output struct {
		JSON bool `json:"json"`
	} `json:"output"`
}

func newShowCmd(flags *support.RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the effective CLI configuration",
		Long: `Show the effective CLI configuration after merging all sources.

The server URL is resolved from the persisted config file, overridden by
the HOROLOGIA_SERVER environment variable if set. The token is read from the
HOROLOGIA_TOKEN environment variable only. The output labels every value with
its source.`,
		Example: `  # Show current configuration
  horo config show

  # Show current configuration as JSON
  horo config show --json`,
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			out := BuildOutput(app.Config)
			if app.Config.JSON {
				return app.PrintJSON(out)
			}

			serverValue := out.Server.Value
			if serverValue == "" {
				serverValue = "<unset>"
			}
			apiBase := out.APIBase.Value
			if apiBase == "" {
				apiBase = "<unavailable>"
			}
			tokenLine := "not configured"
			if out.Token.Configured {
				tokenLine = out.Token.Preview
			}

			app.Printf("Server:   %s (%s)\n", serverValue, out.Server.Source)
			app.Printf("API base: %s\n", apiBase)
			app.Printf("Token:    %s (%s)\n", tokenLine, out.Token.Source)
			app.Printf("Output:   json=%t\n", out.Output.JSON)
			return nil
		}),
	}

	return cmd
}

// BuildOutput formats resolved runtime configuration for display.
func BuildOutput(cfg runtime.Config) Output {
	var out Output
	out.Server.Value = cfg.ServerString()
	out.Server.Source = cfg.ServerSource
	out.APIBase.Value = cfg.APIBaseString()
	out.Token.Configured = cfg.HasToken()
	out.Token.Preview = cfg.RedactedToken()
	out.Token.Source = cfg.TokenSource
	out.Output.JSON = cfg.JSON
	return out
}
