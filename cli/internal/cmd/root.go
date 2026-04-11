package cmd

import (
	"context"
	"io"
	"os"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/runtime"
)

type commandOptions struct {
	stdout io.Writer
	stderr io.Writer
}

type rootFlags struct {
	json bool
}

var (
	version = ""
	commit  = ""
)

// NewRootCmd builds the root cobra command with all subcommands.
func NewRootCmd() *cobra.Command {
	return newRootCmd(commandOptions{
		stdout: os.Stdout,
		stderr: os.Stderr,
	})
}

func newRootCmd(opts commandOptions) *cobra.Command {
	flags := rootFlags{}

	rootCmd := &cobra.Command{
		Use:           "tend",
		Short:         "Tend command-line client",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.SetOut(opts.stdout)
	rootCmd.SetErr(opts.stderr)

	rootCmd.PersistentFlags().BoolVar(&flags.json, "json", false, "Print machine-readable JSON output")

	rootCmd.AddGroup(
		&cobra.Group{ID: "foundation", Title: "Foundation Commands:"},
	)

	rootCmd.AddCommand(
		newConfigCmd(&flags),
		newPingCmd(&flags),
		newWhoamiCmd(&flags),
	)

	return rootCmd
}

func runWithApp(flags *rootFlags, fn func(app *runtime.App, cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := runtime.ResolveConfig(runtime.ResolveInput{
			JSON: flags.json,
		})
		if err != nil {
			return err
		}

		app := runtime.NewApp(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		cmd.SetContext(runtime.WithApp(cmd.Context(), app))
		return fn(app, cmd, args)
	}
}

// Execute runs the root command with Fang's CLI UX.
func Execute(ctx context.Context) error {
	return fang.Execute(
		ctx,
		NewRootCmd(),
		fang.WithVersion(version),
		fang.WithCommit(commit),
	)
}
