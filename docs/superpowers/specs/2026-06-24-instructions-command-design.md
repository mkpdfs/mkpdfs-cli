# `mkp instructions` — agent-oriented usage doc command

**Date:** 2026-06-24
**Status:** Approved (brainstormed + Codex-reviewed)

## Problem

Users want to point an AI coding agent (Claude Code, Cursor, …) at the mkpdfs
workflow without hand-writing the format every time. The desired UX:

> "Claude, create a mkpdfs love-letter template. Get the format from
> `mkp instructions --agent`."

The agent runs the command, reads stdout, and follows it end-to-end: authoring
a Handlebars `.hbs`, pushing it, generating a PDF.

## Solution

A new top-level command:

- `mkp instructions` — short human-readable guide (what the workflow is, the key
  commands, a pointer to `--agent`).
- `mkp instructions --agent` — a dense, copy-pasteable markdown doc addressed to
  an AI agent that walks the full workflow.

Content is **static markdown embedded in the binary** via `go:embed`. No auth, no
network, works offline, and versions in lockstep with the CLI (the commands it
documents always match that build).

### Command shape

- `Use: "instructions"`, `Args: cobra.NoArgs` (so `mkp instructions foo` exits 2
  via the normal usage-error path — do NOT call `requireSubcommand`; this command
  has its own action and no children).
- One bool flag: `--agent` (default false).
- Writes to `cmd.OutOrStdout()`. **No color/ANSI** — output must be clean for
  agents capturing stdout and for `mkp instructions --agent > mkpdfs.md`.
- Honors no other global flags meaningfully (`--json` is irrelevant; the body is
  prose). Do not special-case it.

### Files

- `internal/cli/instructions.go` — command definition + `//go:embed` directives.
- `internal/cli/instructions_agent.md` — the agent doc (embedded).
- `internal/cli/instructions_human.md` — the human guide (embedded).
- Registered via `addInstructionsCommands()` in `root.go` `init()`.

### Agent doc content (`instructions_agent.md`)

Addressed in second person to an AI agent. Sections, in order:

1. **What this is** — mkpdfs turns a Handlebars `.hbs` (arbitrary HTML+CSS) into a
   PDF via headless Chromium. You author the template, push it, generate a PDF.
2. **Prerequisites** — confirm the CLI is installed: `mkp version`. If missing,
   tell the user to install it (`brew install mkpdfs/mkpdfs/mkpdfs`).
3. **Environments — prod is the DEFAULT.** State this loudly. Always work on
   **dev** first by passing `--env dev` on each command (preferred — does not
   mutate global config). Mention `mkp config set environment dev` only as the
   "make dev the default" alternative the user can opt into.
4. **Authentication** — `mkp auth login` is an interactive browser/device flow
   that **a headless agent cannot complete**. So:
   - If not authenticated, **ask the user** to run `mkp --env dev auth login`.
   - Verify with `mkp --env dev auth whoami` (shows email, plan, active env).
     (There is no `auth status`.)
   - For CI / fully-headless use, document `--api-key` (set `MKPDFS_API_KEY` or a
     saved key) on `templates`/`pdf` commands — no browser needed.
5. **Template format** — `.hbs` is HTML + CSS. Page size via CSS
   `@page { size: A4; margin: 2cm }`. Parameterize with `{{variable}}` pulled from
   the JSON data file. Mention the 6.5 MiB source cap here (before authoring).
6. **Helpers** — the helpers the platform registers, with one-line syntax each:
   - `{{ifEq a b}}…{{/ifEq}}` (block, equality)
   - `{{gt a b}}…{{/gt}}` (block, greater-than)
   - `{{formatDate date "YYYY-MM-DD"}}`
   - `{{formatCurrency amount "USD"}}`
   - Built-ins: `{{#each}}`, `{{#if}}`, `{{#unless}}`, `{{#with}}`, `{{else}}`.
7. **Worked example** — a complete, valid, copy-pasteable `carta.hbs` (A4 via
   `@page`, inline CSS, `{{vars}}`, a `{{#each}}` over a list, a `formatDate`) and a
   matching valid `datos.json`. Use an explicit output filename downstream.
8. **Push to dev** — `mkp --env dev templates push carta.hbs`. Note it writes
   `.mkpdfs.json` (binds the cwd to dev), returns a `templateId`, validates
   Handlebars locally, and rejects sources over 6.5 MiB.
9. **Generate + verify** — `mkp --env dev pdf generate -t carta.hbs -d datos.json -o carta.pdf`.
   Note batch data is a JSON array (max 50 items). Tell the agent to confirm the
   PDF was written.
10. **Promote to prod** — `.mkpdfs.json` is **env-bound**; cross-env ops are
    rejected, so you cannot just flip `--env prod` and push from a dev-bound dir.
    Safe path: push the same `.hbs` from a **separate/prod-context directory** (or
    a dir with no dev-bound `.mkpdfs.json`) after `mkp --env prod auth login`, e.g.
    `mkp --env prod templates push carta.hbs`. Reiterate prod is live; pushes there
    prompt for confirmation.
11. **Limits & guards (recap)** — 6.5 MiB per template, batch max 50, env-bound
    `.mkpdfs.json` (no cross-env), local Handlebars validation before push.

### Human doc content (`instructions_human.md`)

Short. What the command/workflow is, the four key commands (`auth login`,
`templates push`, `pdf generate`, `auth whoami`), and a final line: "Building with
an AI agent? Run `mkp instructions --agent` and hand the output to it."

## Testing

`internal/cli/instructions_test.go`:

- Executing `instructions` (human) prints non-empty output and contains a known
  marker (e.g. `templates push`).
- Executing `instructions --agent` prints non-empty output and contains key
  markers: each helper name (`ifEq`, `formatDate`, `formatCurrency`), `@page`,
  `templates push`, `pdf generate`, `--env dev`, `auth whoami`.
- `instructions foo` is a usage error (exit 2 / NoArgs).

The embedded docs are static, so the test doubles as a guard against accidentally
deleting a documented command reference.

## Docs to update

- `README.md` — add an "Instructions / AI agents" section pointing at
  `mkp instructions --agent`.
- mkpdfs orchestrator `CLAUDE.md` — one line in the CLI summary noting the new
  command.

## Out of scope (YAGNI)

- `--output <file>` flag (shell redirection already covers it).
- Fetching the doc from the backend (embedded is simpler and offline).
- Per-locale / Spanish variants (English doc; agents handle translation).
