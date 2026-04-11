package cmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/runtime"
)

type configOutput struct {
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

func newConfigCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Show the effective CLI configuration",
		GroupID: "foundation",
		RunE: runWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			out := buildConfigOutput(app.Config)
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

func buildConfigOutput(cfg runtime.Config) configOutput {
	var out configOutput
	out.Server.Value = cfg.ServerString()
	out.Server.Source = cfg.ServerSource
	out.APIBase.Value = cfg.APIBaseString()
	out.Token.Configured = cfg.HasToken()
	out.Token.Preview = cfg.RedactedToken()
	out.Token.Source = cfg.TokenSource
	out.Output.JSON = cfg.JSON
	return out
}
