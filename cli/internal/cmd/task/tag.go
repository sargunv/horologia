package taskcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newTagCmd() *cobra.Command {
	cmd := support.GroupCommand("tag", "Manage task tags")
	cmd.AddCommand(
		support.StubCommand("set <task>", "Replace task tags"),
		support.StubCommand("add <task> <tag>", "Add a tag to a task"),
		support.StubCommand("remove <task> <tag>", "Remove a tag from a task"),
		support.StubCommand("clear <task>", "Clear task tags"),
	)
	return cmd
}
