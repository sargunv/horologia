package authcmd

import (
	"strings"
	"text/tabwriter"

	apigen "github.com/sargunv/horologia/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

const timeFormat = "2006-01-02 15:04:05Z07:00"

func newTokenCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("token", "Manage personal API tokens")
	cmd.AddCommand(
		newTokenListCmd(flags),
		newTokenCreateCmd(flags),
		newTokenRevokeCmd(flags),
	)
	return cmd
}

func newTokenListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List personal API tokens",
		Long: `List all personal API tokens for the authenticated user.

Displays each token's ID, name, kind, and creation timestamp.`,
		Example: `  # List all API tokens
  horo auth token list

  # List all API tokens as JSON
  horo auth token list --json`,
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			resp, err := api.AuthListTokens(cmd.Context())
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			if len(resp.Items) == 0 {
				app.Printf("No API tokens.\n")
				return nil
			}

			w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
			_, _ = w.Write([]byte("ID\tNAME\tKIND\tCREATED\n"))
			for _, token := range resp.Items {
				_, _ = w.Write([]byte(token.ID + "\t" + token.Name + "\t" + string(token.Kind) + "\t" + token.CreatedAt.Format(timeFormat) + "\n"))
			}
			return w.Flush()
		}),
	}
}

func newTokenCreateCmd(flags *support.RootFlags) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a personal API token",
		Long: `Create a new personal API token with the given name.

The token secret is displayed exactly once. Copy it immediately;
it cannot be retrieved later.`,
		Example: `  # Create a token for CI
  horo auth token create --name "CI deploy"

  # Create a token and capture the secret as JSON
  horo auth token create --name "scripting" --json`,
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			resp, err := api.AuthCreateToken(cmd.Context(), &apigen.AuthTokenCreate{Name: strings.TrimSpace(name)})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			app.Printf("Token created.\n")
			app.Printf("ID:      %s\n", resp.AuthToken.ID)
			app.Printf("Name:    %s\n", resp.AuthToken.Name)
			app.Printf("Kind:    %s\n", resp.AuthToken.Kind)
			app.Printf("Created: %s\n", resp.AuthToken.CreatedAt.Format(timeFormat))
			app.Printf("Token:   %s\n", resp.Token)
			app.Printf("Store this token now. It will not be shown again.\n")
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Name for the new API token")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newTokenRevokeCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <token-id>",
		Short: "Revoke a personal API token",
		Long: `Permanently delete the token identified by <token-id>. This action is
irreversible; any client using the token loses access immediately.
Find token IDs with "horo auth token list".`,
		Example: `  # Revoke a token by ID
  horo auth token revoke abc123`,
		Args: cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			if err := api.AuthDeleteToken(cmd.Context(), apigen.AuthDeleteTokenParams{TokenId: args[0]}); err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(map[string]any{
					"tokenId": args[0],
					"deleted": true,
				})
			}

			app.Printf("Revoked token %s\n", args[0])
			return nil
		}),
	}
}
