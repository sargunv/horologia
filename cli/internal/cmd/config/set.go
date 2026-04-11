package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newSetCmd() *cobra.Command {
	cmd := support.GroupCommand("set", "Set persisted CLI configuration values")
	cmd.AddCommand(
		support.StubCommand("server <url>", "Set the default Tend server"),
	)
	return cmd
}
