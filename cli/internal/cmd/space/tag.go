package spacecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newTagCmd() *cobra.Command {
	cmd := support.GroupCommand("tag", "Manage tags in a space")
	cmd.AddCommand(
		support.StubCommand("list <space>", "List tags in a space"),
		support.StubCommand("create <space>", "Create a tag in a space"),
		support.StubCommand("rename <space> <tag> <new-name>", "Rename a tag"),
		support.StubCommand("delete <space> <tag>", "Delete a tag"),
	)
	return cmd
}
