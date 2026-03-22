package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/config"
)

// skipAuth overrides PersistentPreRunE for commands that don't require authentication.
func skipAuth(*cobra.Command, []string) error { return nil }

// readToken reads the API token from the user. If stdin is a terminal,
// it prompts interactively with echo suppressed. If stdin is a pipe,
// it reads the first line from stdin.
func readToken(cmd *cobra.Command) (string, error) {
	fd := os.Stdin.Fd()
	if term.IsTerminal(fd) {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Token: ")
		raw, err := term.ReadPassword(fd)
		_, _ = fmt.Fprintln(cmd.ErrOrStderr())
		if err != nil {
			return "", fmt.Errorf("read token: %w", err)
		}
		token := strings.TrimSpace(string(raw))
		if token == "" {
			return "", fmt.Errorf("token must not be empty")
		}
		return token, nil
	}

	// Stdin is a pipe — read the first line.
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("read token from stdin: %w", err)
		}
		return "", fmt.Errorf("token must not be empty")
	}
	token := strings.TrimSpace(scanner.Text())
	if token == "" {
		return "", fmt.Errorf("token must not be empty")
	}
	return token, nil
}

func newLoginCmd() *cobra.Command {
	var serverURL string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a Tend server",
		Long: `Authenticate with a Tend server by providing an API token.

In a terminal, you will be prompted to enter the token with hidden input.
You can also pipe a token from stdin or set the TEND_TOKEN environment variable.`,
		Example: `  # Interactive prompt (echo suppressed)
  tend login --server-url https://tend.example.com

  # Pipe from a secret manager or file
  cat ~/.tend-token | tend login --server-url https://tend.example.com

  # Use environment variables
  export TEND_URL=https://tend.example.com
  export TEND_TOKEN=<TOKEN>`,
		PersistentPreRunE: skipAuth,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := readToken(cmd)
			if err != nil {
				return err
			}

			// Use LoadFile (not Load) to avoid persisting env-var values to disk.
			cfg, err := config.LoadFile()
			if err != nil {
				return err
			}

			if serverURL != "" {
				cfg.ServerURL = serverURL
			}
			if cfg.ServerURL == "" {
				return fmt.Errorf("server URL required; use --server-url or set TEND_URL")
			}

			// Validate the token by calling /users/me.
			client, err := config.NewClient(cfg.ServerURL, token)
			if err != nil {
				return err
			}
			user, err := client.UsersMe(cmd.Context())
			if err != nil {
				return fmt.Errorf("token validation failed: %w", FormatAPIError(err))
			}

			// Save server URL to config file and token to keychain.
			if err := config.Save(cfg); err != nil {
				return err
			}
			if err := config.SetToken(token); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s (%s)\n", user.Name, user.Email)
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server-url", "", "Tend server URL")

	return cmd
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "logout",
		Short:             "Remove stored credentials",
		Long:              "Remove the stored API token from the config file. The server URL is preserved.",
		PersistentPreRunE: skipAuth,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.GetToken()
			if err != nil {
				return err
			}
			if token == "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No stored credentials to remove")
				return nil
			}
			if err := config.ClearToken(); err != nil {
				return err
			}
			cfg, err := config.LoadFile()
			if err != nil {
				return err
			}
			if cfg.ServerURL != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Logged out from %s\n", cfg.ServerURL)
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Logged out")
			}
			return nil
		},
	}
}
