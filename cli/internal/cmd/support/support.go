package support

import (
	"fmt"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/runtime"
)

// RootFlags holds persistent flags shared across the CLI.
type RootFlags struct {
	JSON bool
}

// AppRunner executes a command with an initialized runtime app.
type AppRunner func(app *runtime.App, cmd *cobra.Command, args []string) error

// RunWithApp resolves runtime configuration and constructs the shared app.
func RunWithApp(flags *RootFlags, fn AppRunner) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := runtime.ResolveConfig(runtime.ResolveInput{
			JSON: flags.JSON,
		})
		if err != nil {
			return err
		}

		app := runtime.NewApp(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		cmd.SetContext(runtime.WithApp(cmd.Context(), app))
		return fn(app, cmd, args)
	}
}

// GroupCommand creates a command that defaults to printing help.
func GroupCommand(use string, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}

// DefaultSubcommand runs the named subcommand when the parent is invoked bare.
func DefaultSubcommand(name string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		child, _, err := cmd.Find([]string{name})
		if err != nil {
			return err
		}

		child.SetContext(cmd.Context())
		child.SetOut(cmd.OutOrStdout())
		child.SetErr(cmd.ErrOrStderr())
		if child.RunE != nil {
			return child.RunE(child, args)
		}
		return child.Help()
	}
}

// StubCommand creates a placeholder leaf command.
func StubCommand(use string, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("%s is not implemented yet", cmd.CommandPath())
		},
	}
}

// RequireAPI returns the initialized generated API client.
func RequireAPI(app *runtime.App) (*apigen.Client, error) {
	if app.API == nil {
		return nil, runtime.MissingServerError()
	}
	return app.API, nil
}
