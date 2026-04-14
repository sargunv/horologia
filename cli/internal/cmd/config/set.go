package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newSetCmd(flags *support.RootFlags) *cobra.Command {
	type output struct {
		Path   string `json:"path"`
		Server string `json:"server"`
	}

	cmd := support.GroupCommand("set", "Set a persisted configuration value")
	cmd.AddCommand(
		&cobra.Command{
			Use:   "server <url>",
			Short: "Set the default Horologia server URL",
			Long: `Write the given URL to the persisted config file as the default Horologia
server. The URL is normalized before saving. This value can still be
overridden by the HOROLOGIA_SERVER environment variable.`,
			Example: `  # Point the CLI at a local dev server
  horo config set server http://localhost:8080

  # Point the CLI at a hosted instance
  horo config set server https://horologia.example.com`,
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				path, server, err := runtime.SaveServer(args[0])
				if err != nil {
					return err
				}

				app := runtime.NewApp(runtime.Config{JSON: flags.JSON}, cmd.OutOrStdout(), cmd.ErrOrStderr())
				if flags.JSON {
					return app.PrintJSON(output{Path: path, Server: server})
				}

				app.Printf("Server: %s\n", server)
				app.Printf("Path:   %s\n", path)
				return nil
			},
		},
	)
	return cmd
}
