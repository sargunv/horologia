# CLI

Download the `horo` CLI from [GitHub Releases](https://github.com/sargunv/horologia/releases)
(macOS, Linux, Windows). Point it at your server:

```sh
horo config set server https://horologia.example.com
horo auth login
```

The CLI authenticates via OAuth and stores credentials in the system keyring.

For non-interactive use (scripts, CI), set these environment variables instead:

```sh
export HOROLOGIA_SERVER="https://horologia.example.com"
export HOROLOGIA_TOKEN="your-api-token"
```

Though the CLI client is available, the API should not yet be considered stable. Ensure your CLI
version matches the server version, or you will encounter issues.
