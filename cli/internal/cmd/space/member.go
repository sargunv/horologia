package spacecmd

import (
	"strings"

	apigen "github.com/sargunv/horologia/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
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
		Long:  `List all members of the given space, including each member's role (admin, member, or viewer).`,
		Example: `  # List members of the "eng" space
  horo space member list eng`,
		Args: cobra.ExactArgs(1),
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
		Long: `Add a user to the given space with the specified role. If the user
already belongs to the space, the command fails; use set-role to change
an existing member's role.`,
		Example: `  # Add alice as a member
  horo space member add eng alice --role member

  # Add bob as an admin
  horo space member add eng bob --role admin`,
		Args: cobra.ExactArgs(2),
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
		Short: "Change a member's role",
		Long: `Change the role of an existing space member. The <role> argument must be
one of admin, member, or viewer. The user must already belong to the space;
use add to grant initial access.`,
		Example: `  # Promote alice to admin
  horo space member set-role eng alice admin

  # Downgrade bob to viewer
  horo space member set-role eng bob viewer`,
		Args: cobra.ExactArgs(3),
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
		Long: `Remove a user from the given space, revoking all access immediately.
To restore access later, use add.`,
		Example: `  # Remove alice from the "eng" space
  horo space member remove eng alice`,
		Args: cobra.ExactArgs(2),
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
