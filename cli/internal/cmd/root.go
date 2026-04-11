package cmd

import (
	"context"
	"io"
	"os"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"

	authcmd "github.com/sargunv/tend/cli/internal/cmd/auth"
	configcmd "github.com/sargunv/tend/cli/internal/cmd/config"
	foundationcmd "github.com/sargunv/tend/cli/internal/cmd/foundation"
	spacecmd "github.com/sargunv/tend/cli/internal/cmd/space"
	"github.com/sargunv/tend/cli/internal/cmd/support"
	taskcmd "github.com/sargunv/tend/cli/internal/cmd/task"
	usercmd "github.com/sargunv/tend/cli/internal/cmd/user"
)

type commandOptions struct {
	stdout io.Writer
	stderr io.Writer
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
	flags := &support.RootFlags{}

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

	rootCmd.PersistentFlags().BoolVar(&flags.JSON, "json", false, "Print machine-readable JSON output")

	rootCmd.AddGroup(
		&cobra.Group{ID: "foundation", Title: "Foundation Commands:"},
		&cobra.Group{ID: "auth", Title: "Authentication Commands:"},
		&cobra.Group{ID: "workspace", Title: "Workspace Commands:"},
		&cobra.Group{ID: "account", Title: "Account Commands:"},
	)

	rootCmd.AddCommand(
		foundationcmd.NewStatus(flags),
		configcmd.New(flags),
		authcmd.New(flags),
		spacecmd.New(flags),
		taskcmd.New(flags),
		usercmd.New(flags),
	)
	return rootCmd
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
