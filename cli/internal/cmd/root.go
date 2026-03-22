package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/sargunv/tend/server/api/gen"

	"github.com/sargunv/tend/cli/internal/config"
	"github.com/sargunv/tend/cli/internal/output"
)

type contextKey int

const appContextKey contextKey = iota

// AppContext holds the API client and printer for use by subcommands.
// Note: Client may be nil when Printer.IsSchemaMode() is true.
type AppContext struct {
	Client    gen.Invoker
	Printer   *output.Printer
	ServerURL string
}

// GetAppContext retrieves the AppContext from a cobra command.
// Panics if not set — only call this from commands that do not override
// PersistentPreRunE (such as login and logout).
func GetAppContext(cmd *cobra.Command) *AppContext {
	ctx, ok := cmd.Context().Value(appContextKey).(*AppContext)
	if !ok || ctx == nil {
		panic("AppContext not set: command must not override PersistentPreRunE without setting up its own context")
	}
	return ctx
}

// NewRootCmd builds the root cobra command with all subcommands.
func NewRootCmd() *cobra.Command {
	var jsonFlag, jsonSchemaFlag bool

	rootCmd := &cobra.Command{
		Use:   "tend",
		Short: "Manage spaces and tasks",
		Long: `Tend is a CLI for managing spaces and tasks on a Tend server.

To get started, authenticate with your server:

  tend login --server-url https://tend.example.com

Then explore available commands:

  tend spaces list
  tend tasks list --space <slug>`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if jsonFlag && jsonSchemaFlag {
				return fmt.Errorf("--json and --json-schema are mutually exclusive")
			}

			mode := output.ModeTable
			if jsonFlag {
				mode = output.ModeJSON
			} else if jsonSchemaFlag {
				mode = output.ModeJSONSchema
			}
			printer := output.New(mode)

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			appCtx := &AppContext{
				Printer:   printer,
				ServerURL: cfg.ServerURL,
			}

			// Schema mode doesn't need a client — skip construction.
			if mode != output.ModeJSONSchema {
				token, err := config.GetToken()
				if err != nil {
					return err
				}
				client, err := config.NewClient(cfg.ServerURL, token)
				if err != nil {
					return err
				}
				appCtx.Client = client
			}

			cmd.SetContext(context.WithValue(cmd.Context(), appContextKey, appCtx))
			return nil
		},
	}

	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&jsonSchemaFlag, "json-schema", false, "Output the JSON Schema for this command's output")

	rootCmd.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newWhoamiCmd(),
		newSpacesCmd(),
		newTasksCmd(),
	)

	return rootCmd
}

// FormatAPIError extracts a user-friendly message from an API error.
// It strips control characters from the server message to prevent terminal injection.
func FormatAPIError(err error) error {
	var apiErr *gen.ApiErrorStatusCode
	if errors.As(err, &apiErr) {
		msg := strings.Map(func(r rune) rune {
			if unicode.IsControl(r) && r != '\n' {
				return -1
			}
			return r
		}, apiErr.Response.Message)
		if len(msg) > 512 {
			msg = msg[:512] + "..."
		}
		return fmt.Errorf("[%s] %s", apiErr.Response.Code, msg)
	}
	return err
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
