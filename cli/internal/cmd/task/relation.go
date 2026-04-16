package taskcmd

import (
	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newRelationCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("relation", "Manage task relations")
	cmd.AddCommand(
		newRelationAddCmd(flags),
		newRelationRemoveCmd(flags),
	)
	return cmd
}

func newRelationAddCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "add <space> <task> <kind> <related-task>",
		Short: "Add a relation between two tasks",
		Long: `Add a directed relation from one task to another within a space.
The <kind> argument describes how the two tasks relate. Valid values:
parent_of, child_of, blocks, blocked_by, relates_to, duplicates,
triggers, triggered_by, spawns, spawned_by.`,
		Example: `  # Mark SV-42 as blocking SV-43
  horo task relation add my-project SV-42 blocks SV-43

  # Record that SV-10 relates to SV-11
  horo task relation add my-project SV-10 relates_to SV-11`,
		Args: cobra.ExactArgs(4),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			kind, err := parseRelationKind(args[2])
			if err != nil {
				return err
			}

			relation, err := api.SpaceTaskRelationsCreate(cmd.Context(), &apigen.TaskRelationCreate{
				Kind:          kind,
				RelatedTaskId: args[3],
			}, apigen.SpaceTaskRelationsCreateParams{
				SpaceSlug: args[0],
				TaskId:    args[1],
			})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(relation)
			}
			app.Printf("Added relation %s -> %s (%s)\n", args[1], relation.RelatedTaskId, relation.Kind)
			return nil
		}),
	}
}

func newRelationRemoveCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <space> <task> <kind> <related-task>",
		Short: "Remove a relation between two tasks",
		Long: `Remove an existing relation from one task to another. The <kind>
argument must match the relation to delete. Valid values: parent_of,
child_of, blocks, blocked_by, relates_to, duplicates, triggers,
triggered_by, spawns, spawned_by.`,
		Example: `  # Remove the blocks relation from SV-42 to SV-43
  horo task relation remove my-project SV-42 blocks SV-43`,
		Args: cobra.ExactArgs(4),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			kind, err := parseRelationKind(args[2])
			if err != nil {
				return err
			}

			if err := api.SpaceTaskRelationsDelete(cmd.Context(), apigen.SpaceTaskRelationsDeleteParams{
				SpaceSlug:     args[0],
				TaskId:        args[1],
				Kind:          kind,
				RelatedTaskId: args[3],
			}); err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(map[string]any{
					"spaceSlug":     args[0],
					"taskId":        args[1],
					"kind":          kind,
					"relatedTaskId": args[3],
					"deleted":       true,
				})
			}
			app.Printf("Removed relation %s -> %s (%s)\n", args[1], args[3], kind)
			return nil
		}),
	}
}
