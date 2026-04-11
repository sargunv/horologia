package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newUnsetCmd() *cobra.Command {
	cmd := support.GroupCommand("unset", "Unset persisted CLI configuration values")
	cmd.AddCommand(
		support.StubCommand("server", "Unset the default Tend server"),
	)
	return cmd
}
