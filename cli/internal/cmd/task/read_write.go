package taskcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newReadWriteCmds() []*cobra.Command {
	return []*cobra.Command{
		support.StubCommand("list", "List tasks in a space"),
		support.StubCommand("mine", "List tasks assigned to the current user"),
		support.StubCommand("show <task>", "Show a task"),
		support.StubCommand("create", "Create a task"),
		support.StubCommand("update <task>", "Update scalar task fields"),
		support.StubCommand("complete <task>", "Mark a task complete"),
		support.StubCommand("delete <task>", "Delete a task"),
		support.StubCommand("activity <task>", "Show activity for a task"),
	}
}
