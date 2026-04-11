package spacecmd

import (
	"strings"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newMemberCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("member", "Manage space membership and roles")
	cmd.AddCommand(
		newMemberListCmd(flags),
		newMemberAddCmd(flags),
		newMemberSetRoleCmd(flags),
		newMemberRemoveCmd(flags),
	)
	return cmd
}

func newMemberListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list <space>",
		Short: "List members of a space",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			resp, err := api.SpaceMembersList(cmd.Context(), apigen.SpaceMembersListParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printMemberList(app, resp.Items)
		}),
	}
}

func newMemberAddCmd(flags *support.RootFlags) *cobra.Command {
	var role string

	cmd := &cobra.Command{
		Use:   "add <space> <user>",
		Short: "Add a user to a space",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			parsedRole, err := parseSpaceRole(role)
			if err != nil {
				return err
			}

			member, err := api.SpaceMembersCreate(cmd.Context(), &apigen.SpaceMemberCreate{
				UserId: strings.TrimSpace(args[1]),
				Role:   parsedRole,
			}, apigen.SpaceMembersCreateParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(member)
			}

			app.Printf("Added %s to %s as %s\n", member.UserId, args[0], member.Role)
			return nil
		}),
	}

	cmd.Flags().StringVar(&role, "role", "", "Membership role: admin, member, or viewer")
	_ = cmd.MarkFlagRequired("role")
	return cmd
}

func newMemberSetRoleCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-role <space> <user> <role>",
		Short: "Change a member role",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			role, err := parseSpaceRole(args[2])
			if err != nil {
				return err
			}

			member, err := api.SpaceMembersUpdate(cmd.Context(), &apigen.SpaceMemberUpdate{
				Role: role,
			}, apigen.SpaceMembersUpdateParams{
				SpaceSlug: args[0],
				UserId:    strings.TrimSpace(args[1]),
			})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(member)
			}

			app.Printf("Updated %s in %s to role %s\n", member.UserId, args[0], member.Role)
			return nil
		}),
	}
}

func newMemberRemoveCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <space> <user>",
		Short: "Remove a member from a space",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			userID := strings.TrimSpace(args[1])
			if err := api.SpaceMembersDelete(cmd.Context(), apigen.SpaceMembersDeleteParams{
				SpaceSlug: args[0],
				UserId:    userID,
			}); err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(map[string]any{
					"spaceSlug": args[0],
					"userId":    userID,
					"deleted":   true,
				})
			}

			app.Printf("Removed %s from %s\n", userID, args[0])
			return nil
		}),
	}
}
