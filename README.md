# mkp — mkpdfs CLI

`mkp` is the command-line interface for [mkpdfs](https://mkpdfs.com), a multi-tenant PDF generation service. It lets you upload and manage Handlebars templates, generate PDFs from those templates and JSON data, and manage API tokens — all from your terminal or CI pipeline.

---

## Install

### Homebrew (recommended)

```bash
brew install mkpdfs/mkpdfs/mkpdfs
```

This installs the `mkp` binary (macOS/Linux, Intel + Apple Silicon).

### Build from source

Requires Go 1.21+.

```bash
git clone https://github.com/sim4gh/mkpdfs-cli
cd mkpdfs-cli
make build          # produces ./mkp-cli
make dev-link       # symlinks ./mkp-cli → /opt/homebrew/bin/mkp-cli
```

### Binary naming convention

Two binaries coexist deliberately (same convention as nikte-cli):

- **`mkp`** — the production binary, installed from Homebrew (`brew install mkpdfs/mkpdfs/mkpdfs`, built by GoReleaser from `.goreleaser.yml` on tagged releases).
- **`mkp-cli`** — the local dev build (`make build` / `make dev-link`), named differently so it never shadows the brew-installed `mkp`.

---

## Quick start

### 1. Log in

```bash
mkp-cli auth login            # prod (default)
mkp-cli auth login --env dev  # dev environment
```

Login uses a device-code flow. The CLI opens your browser and displays an 8-character code. Approve the prompt in the browser; the CLI polls until authorization completes and stores credentials in the config file. No secrets are ever typed into the terminal.

### 2. Edit loop

```bash
# Pull an existing template by its ID
mkp-cli templates pull <templateId>

# Edit it
$EDITOR mytemplate.hbs

# Push changes back
mkp-cli templates push mytemplate.hbs

# Generate a PDF
mkp-cli pdf generate -t mytemplate.hbs -d data.json --open
```

`templates push` creates the template on first push and updates it on subsequent pushes. The mapping between local file and remote template ID is stored in `.mkpdfs.json` in the current directory.

---

## Commands

```
mkp-cli
├── auth
│   ├── login       Log in via browser (device-code flow)
│   ├── logout      Clear stored credentials for the environment
│   └── whoami      Show current email, plan, and environment
│
├── templates (alias: tpl)
│   ├── list        List all templates (table or --json)
│   ├── get <id>    Show metadata and detected Handlebars variables
│   ├── pull <id>   Download template content to a local .hbs file
│   ├── push <file> Create or update a template from a .hbs file
│   └── delete <id> Delete a template (with confirmation)
│
├── pdf
│   └── generate    Generate a PDF from a template and JSON data file
│       -t <id|file>    template ID or local .hbs file (required)
│       -d <file>       JSON data file (required)
│       -o <path>       output PDF path
│       --open          open the PDF after download
│       --api-key       use server-to-server route with your tlfy_ API key
│
├── tokens
│   ├── list        List API tokens
│   ├── create      Create a new API token (--name required; --save to store in config)
│   └── revoke <id> Revoke an API token
│
├── credits         Show credit balance and auto-recharge status
│   ├── ledger      Show recent credit ledger entries (most recent 50)
│   ├── auto-recharge   Show settings, or --enable [--threshold N] / --disable
│   └── buy         Buy a credit pack (opens Stripe checkout in your browser)
│
├── usage           Show current-month usage stats and credit balance
│
├── instructions    Print the mkpdfs workflow guide
│   └── --agent     Emit a markdown walkthrough for an AI coding agent
│
└── config
    ├── list        List configuration (secrets masked)
    ├── get <key>   Get a config value
    ├── set <key> <value>  Set a config value
    └── path        Print the config file path
```

Global flags available on every command:

| Flag | Description |
|------|-------------|
| `--env dev\|prod` | Override the active environment for this invocation |
| `--json` | Machine-readable JSON output |
| `--yes` | Assume yes for all confirmation prompts |
| `--verbose` / `-v` | Verbose output |

---

## Building templates with an AI agent

If you use an AI coding agent (Claude Code, Cursor, …), point it at the built-in
instructions instead of explaining the format yourself:

```bash
mkp instructions --agent      # dense, copy-pasteable walkthrough for an agent
mkp instructions              # short human guide
```

`mkp instructions --agent` prints a complete, offline markdown doc — the `.hbs`
format (HTML + CSS, `@page` size), the available Handlebars helpers with their
exact signatures, a worked `carta.hbs` + `datos.json` example, and the exact
`templates push` / `pdf generate` commands. Just tell your agent:

> "Create a mkpdfs love-letter template. Get the format from `mkp instructions --agent`."

The agent runs the command, reads the output, and builds + pushes the template
end to end. Output is plain stdout, so you can also save it:
`mkp instructions --agent > mkpdfs.md`.

---

## Credits & billing

mkpdfs is prepaid: **$10 = 1,000 credits, 1 credit = 1 PDF page**, and credits never
expire. PDF generation is blocked with a `402` once your balance is exhausted.

```bash
mkp credits                              # balance + auto-recharge status
mkp credits ledger                       # recent ledger entries (most recent 50)
mkp credits auto-recharge                # show current setting
mkp credits auto-recharge --enable --threshold 100   # recharge when balance < 100
mkp credits auto-recharge --disable
mkp credits buy                          # opens Stripe checkout in your browser
```

> **Note:** the `credits` commands (and `templates`, `tokens`, `usage`) require a
> browser login — they authenticate with your Cognito session, **not** an API key.
> `MKPDFS_API_KEY` / `--api-key` only works for `pdf generate`.

## Environments

mkpdfs has two environments: `prod` (default) and `dev`.

Set a persistent default:

```bash
mkp-cli config set environment dev
mkp-cli config get environment
```

Or override per-command:

```bash
mkp-cli --env dev templates list
```

Credentials are stored separately per environment, so you can be logged in to both simultaneously.

---

## CI usage

For headless pipelines, use an API key instead of a browser login:

```bash
# Generate a PDF — API key passed via environment variable
MKPDFS_API_KEY=tlfy_... mkp-cli pdf generate \
  --api-key \
  -t <templateId> \
  -d data.json \
  -o output.pdf
```

The `--api-key` flag routes the request through the server-to-server endpoint (`POST /v1/pdf/generate`). The template ID must be a UUID (not a local file path), because `.mkpdfs.json` is typically absent in CI.

**Headless CI:** `pdf generate` and **`templates` (list/get/pull/push/delete)** work with an API key. Use `--api-key` (reads `MKPDFS_API_KEY` or the saved key):

```bash
export MKPDFS_API_KEY=tlfy_...
mkp templates push invoice.hbs --api-key          # requires a checked-in .mkpdfs.json entry
mkp templates push invoice.hbs --api-key --new    # create a new template headless
mkp templates pull <templateId> --api-key
```

`tokens`, `auth`, `usage`, and `credits` still require a browser login (Cognito JWT).

---

## Config file locations

The config file (`config.json`) is stored in the OS-standard location:

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/mkpdfs/config.json` |
| Linux | `~/.config/mkpdfs/config.json` (or `$XDG_CONFIG_HOME/mkpdfs/config.json`) |
| Windows | `%APPDATA%\mkpdfs\config.json` |

The file is written with mode `0600`. On Windows, file permissions are advisory — store the config on a personal user profile to avoid unintended access by other accounts.

Print the path for the current machine:

```bash
mkp-cli config path
```

---

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Runtime error (API failure, auth error, I/O error) |
| `2` | Usage error (bad flags, missing required argument, validation failure) |
