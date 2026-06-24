## Authentication

`mkp auth login` is an **interactive browser/device flow** — an automated agent
cannot complete it. So:

- **If the user is not logged in, ASK THEM to run** `mkp --env dev auth login`
  and wait for confirmation.
- Verify the session with `mkp --env dev auth whoami` — it prints the email, the
  plan, and the active environment. (There is **no `auth status`**; the command is
  `whoami`.)
- **Headless / CI:** if the user provides an API key, skip the browser entirely.
  Set `MKPDFS_API_KEY=tlfy_…` (or save one with `mkp tokens create --save`) and add
  `--api-key` to `templates` and `pdf` commands.

`mkp auth logout` clears stored credentials for the active environment.
