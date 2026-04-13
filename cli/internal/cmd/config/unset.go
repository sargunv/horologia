package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newUnsetCmd(flags *support.RootFlags) *cobra.Command {
	type output struct {
		Path string `json:"path"`
	}

	cmd := support.GroupCommand("unset", "Remove a persisted configuration value")
	cmd.AddCommand(
		&cobra.Command{
			Use:   "server",
			Short: "Remove the persisted server URL",
			Long: `Remove the persisted server URL from the config file. After removal,
the CLI falls back to the HOROLOGIA_SERVER environment variable. If that
variable is also unset, commands that contact the server will fail.`,
			Example: `  # Remove the server URL from the config file
  horo config unset server`,
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				path, err := runtime.UnsetServer()
				if err != nil {
					return err
				}

				app := runtime.NewApp(runtime.Config{JSON: flags.JSON}, cmd.OutOrStdout(), cmd.ErrOrStderr())
				if flags.JSON {
					return app.PrintJSON(output{Path: path})
				}

				app.Printf("Path: %s\n", path)
				return nil
			},
		},
	)
	return cmd
}
