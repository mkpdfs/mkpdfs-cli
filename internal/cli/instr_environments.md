## Environments — dev vs prod

**The default environment is PROD (live).** While building and iterating, always
work on **dev** by passing `--env dev` to every command. Do not push or generate
against prod until the template is verified on dev.

- `--env dev` / `--env prod` — pick the environment for a single command.
- `mkp config set environment dev` — make dev the default (optional; mutates your
  global CLI config). Prefer explicit `--env dev` when acting on someone's behalf.
- `.mkpdfs.json` (written by `templates pull`/`push`) **binds the current
  directory to one environment**; the CLI rejects cross-env operations.

**Promoting a template to prod.** Because `.mkpdfs.json` is env-bound, you cannot
just switch to `--env prod` and push from the same dev-bound directory. Safe path:

1. Ask the user to log in to prod: `mkp --env prod auth login`.
2. From a **separate directory** (no dev-bound `.mkpdfs.json`), push the same
   file: `mkp --env prod templates push carta.hbs`.

Prod is live and `mkp` prompts for confirmation before writing there. Treat every
prod command as publishing.
