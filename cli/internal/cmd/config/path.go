package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newPathCmd() *cobra.Command {
	return support.StubCommand("path", "Show the persisted CLI config path")
}
