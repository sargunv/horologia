package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newSetCmd(flags *support.RootFlags) *cobra.Command {
	type output struct {
		Path   string `json:"path"`
		Server string `json:"server"`
	}

	cmd := support.GroupCommand("set", "Set persisted CLI configuration values")
	cmd.AddCommand(
		&cobra.Command{
			Use:   "server <url>",
			Short: "Set the default Tend server",
			Args:  cobra.ExactArgs(1),
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
