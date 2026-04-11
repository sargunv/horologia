package configcmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newPathCmd(flags *support.RootFlags) *cobra.Command {
	type output struct {
		Path   string `json:"path"`
		Exists bool   `json:"exists"`
	}

	return &cobra.Command{
		Use:   "path",
		Short: "Show the persisted CLI config path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := runtime.ConfigPath()
			if err != nil {
				return err
			}

			_, err = os.Stat(path)
			exists := err == nil
			if err != nil && !os.IsNotExist(err) {
				return err
			}

			app := runtime.NewApp(runtime.Config{JSON: flags.JSON}, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if flags.JSON {
				return app.PrintJSON(output{Path: path, Exists: exists})
			}

			app.Printf("%s\n", path)
			return nil
		},
	}
}
