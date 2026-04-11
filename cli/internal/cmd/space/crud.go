package spacecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newCRUDCmds() []*cobra.Command {
	return []*cobra.Command{
		support.StubCommand("list", "List accessible spaces"),
		support.StubCommand("show <space>", "Show a space"),
		support.StubCommand("create", "Create a space"),
		support.StubCommand("update <space>", "Update a space"),
		support.StubCommand("delete <space>", "Delete a space"),
		support.StubCommand("activity <space>", "Show activity for a space"),
	}
}
