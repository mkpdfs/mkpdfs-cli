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

A new top-level command whose content is split into **topic sections**, each
reachable by its own flag:

- `mkp instructions` — short human-readable guide + a menu of the topic flags.
- `mkp instructions --format` — the `.hbs` format (HTML/CSS, `@page`, variables,
  helpers).
- `mkp instructions --auth` — authentication (login, whoami, `--api-key`).
- `mkp instructions --environments` — dev vs prod and how to switch. (Named
  `--environments`, NOT `--env`, because `--env` is the global persistent flag.)
- `mkp instructions --plans` — plans, credits and limits.
- `mkp instructions --agent` — a dense, copy-pasteable doc addressed to an AI
  agent that composes ALL sections (agent intro → environments → auth → format →
  worked example → plans) into a full walkthrough.

Topic flags combine (`--format --plans` prints both sections, joined by a `---`
rule, in canonical order). `--agent` implies everything and ignores topic flags.

Content is **static markdown embedded in the binary** via `go:embed`. No auth, no
network, works offline, and versions in lockstep with the CLI (the commands it
documents always match that build).

### Command shape

- `Use: "instructions"`, `Args: cobra.NoArgs` (so `mkp instructions foo` exits 2
  via the normal usage-error path — do NOT call `requireSubcommand`; this command
  has its own action and no children).
- Five bool flags: `--agent`, `--format`, `--auth`, `--environments`, `--plans`
  (all default false). Request-scoped (bound in `newInstructionsCmd`) so no
  package-level flag state leaks across executions.
- Writes to `cmd.OutOrStdout()`. **No color/ANSI** — output must be clean for
  agents capturing stdout and for `mkp instructions --agent > mkpdfs.md`.

### Files

- `internal/cli/instructions.go` — command definition + `//go:embed` directives +
  `joinSections` composer.
- Embedded section files: `instructions_human.md`, `instr_agent_intro.md`,
  `instr_environments.md`, `instr_auth.md`, `instr_format.md`, `instr_example.md`,
  `instr_plans.md`.
- Registered via `addInstructionsCommands()` in `root.go` `init()`.

### Agent doc content (composed sections)

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

- `instructions` (human) prints the workflow + the topic-flag menu.
- Each topic flag (`--format`, `--auth`, `--environments`, `--plans`) prints its
  section, asserted via section-specific markers (e.g. `--plans` → `1,000 credits`,
  `enterprise`, `INSUFFICIENT_CREDITS`).
- Combining flags (`--format --plans`) prints both sections joined by a `---` rule.
- `instructions --agent` contains markers spanning every composed section
  (helpers, `@page`, `templates push`, `pdf generate`, `--env dev`, `auth whoami`,
  `1,000 credits`) and the explicit "no `auth status`" steer.
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
