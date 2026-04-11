package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newUnsetCmd(flags *support.RootFlags) *cobra.Command {
	type output struct {
		Path string `json:"path"`
	}

	cmd := support.GroupCommand("unset", "Unset persisted CLI configuration values")
	cmd.AddCommand(
		&cobra.Command{
			Use:   "server",
			Short: "Unset the default Tend server",
			Args:  cobra.NoArgs,
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
