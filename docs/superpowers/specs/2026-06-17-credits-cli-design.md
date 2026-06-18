# Design — `mkp credits` + billing fixes

**Date:** 2026-06-17
**Status:** Approved (user delegated final review to codex)
**Scope:** CLI-only. No backend changes, no new dependencies.

## Background

The mkpdfs billing model switched from monthly subscriptions to **prepaid credits**
on 2026-06-12 ($10 = 1,000 credits, 1 credit = 1 PDF page, never expire; 10 welcome
credits on signup). The CLI (`mkp`, v0.1.0) predates that migration and has two
problems:

1. **No way to see or manage credits.** A user running `mkp pdf generate --api-key`
   in CI can hit `402 INSUFFICIENT_CREDITS` with no CLI command to check balance,
   view history, configure auto-recharge, or buy more.
2. **`mkp usage` is broken.** It reads `subscriptionLimits.pagesPerMonth`, a field
   that no longer exists post-migration (the limit renders as `0`), and frames
   output as a "billing period" — language from the dead subscription model.

This spec adds a `mkp credits` command group and fixes the stale billing surfaces.
Headless API-key support for `templates`/`tokens` (originally raised alongside this)
is **out of scope** — it requires new backend `/v1/*` routes and is captured as a
separate follow-up (see "Out of scope").

## Backend contracts (already exist, JWT-only)

Confirmed by source inspection of mkpdfs-backend:

- **`GET /user/profile`** → `data.subscription` carries `creditBalance` (number),
  `autoRecharge` (bool), `rechargeThreshold` (number), `autoRechargeError` (string?),
  `plan` (`"credits"` | `"enterprise"`). Also `data.subscriptionLimits` =
  `{ templatesAllowed, apiTokensAllowed, maxPdfSizeMB, aiGenerationsPerMonth }`
  (note: **no** `pagesPerMonth`).
- **`GET /billing/ledger`** → `{ success, entries: [...] }`, 50 most recent, no
  pagination. Entry: `{ entryId, type: "debit"|"purchase"|"auto_recharge"|"refund",
  amount, balanceAfter (number|null), description (string|null), createdAt }`.
- **`PUT /billing/auto-recharge`** ← `{ enabled: bool, threshold?: int 1..1000 }` →
  `{ success, autoRecharge, rechargeThreshold }`. Returns **HTTP 400** with
  `{ success:false, error:"NO_PAYMENT_METHOD", message }` if no saved card. No GET
  for settings — read them from `/user/profile`.
- **`POST /stripe/create-checkout-session`** → `{ success, url, sessionId }`. Fixed
  pack (no quantity param). JWT-only.

All four use the Cognito Gateway authorizer (`iamOnlyMiddleware`, no API key). The
CLI already has `jwtClient()` for exactly this, used by `usage`. **Consequence:**
the entire `credits` group requires a browser login (`mkp auth login`) and does
**not** work with `MKPDFS_API_KEY` / `--api-key`. This must be stated in the README
`credits` section, and any "not authenticated" path uses the existing
`jwtClient()` error (`Run "mkp auth login"`).

> **Review note (codex, 2026-06-17):** initial spec was No-Go; the points below are
> folded into the relevant sections. Key corrections: ledger `amount` is **already
> signed** by the backend (debit stored as `-amount`, purchase/recharge positive,
> refund negative — verified in `creditService.ts`), so render as-is, never re-sign;
> `--json` must stay machine-readable everywhere (no bare URLs); the parent `mkp
> credits` cannot reuse `requireSubcommand` (it overwrites `RunE`); credit values
> decode as JSON numbers but are whole credits — format as integers; all leaf
> commands get `cobra.NoArgs`; browser open goes through a package-level indirection
> for testability; credits endpoints are JWT-only — say so in docs and errors.

## Command surface

New file `internal/cli/credits.go`, wired in `root.go` via `addCreditsCommands()`.
Parent command + four behaviors. All accept the global `--json`/`--env` flags and
use `jwtClient()`. Every leaf command sets `Args: cobra.NoArgs` so stray positional
args become usage errors (exit 2) instead of being silently ignored.

```
mkp credits                 # balance + auto-recharge status at a glance
mkp credits ledger          # most recent 50 ledger entries
mkp credits auto-recharge   # show current config (no flags)
mkp credits auto-recharge --enable [--threshold N]
mkp credits auto-recharge --disable
mkp credits buy             # create checkout session, open browser
```

### `mkp credits` (parent, has its own action)

Reads `GET /user/profile`. Human output:

```
Credits — 1,240 remaining

  Auto-recharge:  on  (when balance < 100)
  Plan:           credits
```

- `enterprise` plan → balance shows `unlimited`.
- If `autoRechargeError` is non-empty, append a red warning line:
  `Auto-recharge failed: <msg>. Update your card: mkp credits buy`.
- A negative balance (possible after a refund clawback) renders literally
  (`-30 remaining`) plus a hint: `Buy credits to continue: mkp credits buy`.
- `--json` prints the relevant profile subset (balance, autoRecharge, threshold,
  autoRechargeError, plan).
- Balance renders as a whole integer with thousands separators (`1,240`), never a
  float (the JSON number decodes to `float64` but credits are whole pages).
- **Parent with an action — and `requireSubcommand` must NOT be used here.**
  `requireSubcommand` (root.go) overwrites `RunE`, which would clobber this command's
  own balance action. Instead `mkp credits` defines its own `RunE`: known
  subcommands (`ledger`, `auto-recharge`, `buy`) are routed by Cobra; a bare
  `mkp credits` runs the balance view; and `mkp credits <unknown>` arrives in this
  `RunE` as a positional arg, so `RunE` returns the same `unknown command … : %w
  ErrUsage` error (exit 2) when `len(args) > 0`.

### `mkp credits ledger`

Reads `GET /billing/ledger`. Table columns: `DATE  TYPE  AMOUNT  BALANCE  DESCRIPTION`.
- **`amount` is already signed by the backend** (debit/refund negative,
  purchase/recharge positive). Render it verbatim — do **not** infer or re-apply a
  sign from `type` (that would double-negate debits). Format as a whole integer with
  thousands separators and an explicit `+`/`-`.
- Color used only when stdout is a TTY and `--json` is off (green for positive
  amounts, default for negative) — never break `--json` or piped output.
- `balanceAfter` null → blank cell. `description` null → blank.
- Empty list → `No ledger entries yet.`
- Exactly 50 rows → footer `(showing most recent 50)`.
- `--json` prints the raw response body.

### `mkp credits auto-recharge`

- **No flags:** read `/user/profile`, print current setting
  (`Auto-recharge is on (threshold 100)` / `... is off`). `--json` → the
  `{autoRecharge, rechargeThreshold}` subset.
- **`--enable`:** `PUT { enabled:true, threshold:<N or omitted> }`. `--threshold`
  validated client-side as int in `[1,1000]`; out of range → usage error (exit 2).
- **`--disable`:** `PUT { enabled:false }`.
- **Flag-combination rules (all usage errors → exit 2):** `--enable` + `--disable`
  together; `--threshold` without `--enable`; `--disable` + `--threshold`.
- **`NO_PAYMENT_METHOD` handling (implementation detail).** `client.Put` returns
  `(resp, err)` where `err` is the generic parsed message on HTTP ≥400. The handler
  must inspect `resp` even when `err != nil`: if `resp.StatusCode == 400` and the
  body decodes to `error == "NO_PAYMENT_METHOD"`, print the actionable message
  `No saved card. Buy a credit pack first: mkp credits buy` (exit 1); otherwise
  surface the generic `err`.
- On success, echo the resulting state from the response. `--json` → the raw
  `{autoRecharge, rechargeThreshold}` response.

### `mkp credits buy`

`POST /stripe/create-checkout-session` (empty body) → returns `{success, url,
sessionId}`.
- **`--json`:** print the raw response (machine-readable), do **not** open a browser.
- **Human mode:** print the URL (so it is always visible/copyable), then attempt to
  open it. Browser-open is indirected through a package-level `var openURL =
  browser.OpenURL` so tests can stub it; `auth login` calls `browser.OpenURL`
  directly today, so this introduces the seam. If `openURL` returns an error, the
  command still succeeds (exit 0) — the URL was already printed; emit a one-line
  note that the browser could not be opened.
- Opens directly (no confirmation prompt) — checkout is the confirmation step and
  nothing is charged until the user pays in the browser.

## Changes to existing files

### `internal/cli/usage.go`
- Drop only the dead **limit** `PagesPerMonth` (and remove that field from the
  `subscriptionLimits` struct). **Keep** the current-month count itself — render it
  as a plain `PDF pages generated: N` line (no `/ limit` denominator). The monthly
  stats are still useful; only the obsolete cap is removed.
- Drop the "billing period" framing; header becomes `Usage — YYYY-MM` (the month is
  still meaningful for the stats counters).
- Show credit balance at the top (pulled from the same `/user/profile` call
  `fetchLimits` already makes — extend it to also return `creditBalance`). If the
  profile call fails, fall back to printing stats without the balance line.
- Keep templates/tokens limit lines and data-generated. Add footer:
  `Detailed credit history: mkp credits ledger`.

### `internal/api/client.go`
- `parseAPIError`: the `402` case currently appends `— see https://mkpdfs.com/pricing`.
  Change to `— buy credits: mkp credits buy`. Remove the loose
  `strings.Contains(lower, "subscription")` match (dead model). Keep the limit/auth
  cases unchanged.

### `README.md`
- Remove "No Homebrew release has been tagged yet; until then, build from source."
- Add `brew install mkpdfs/mkpdfs/mkpdfs` as the primary install path (the tap
  `mkpdfs/homebrew-mkpdfs` and release `v0.1.0` already exist and publish the formula).
- Add a `credits` section documenting the new commands.
- Update the command tree and the `usage` description (stats, not "billing period").

### Enterprise plan

`enterprise` is unlimited/manually-billed. The CLI does **not** add client-side
gating for `buy` / `auto-recharge` on enterprise accounts — it forwards the request
and lets the backend respond (keeps the CLI simple and avoids encoding billing
policy in the client). The only enterprise special-case is display: both
`mkp credits` and the balance line in `mkp usage` show `unlimited` rather than a
number when `plan == "enterprise"`.

## Go structs

- Extend the profile-decode struct (currently inline in `fetchLimits`) to also read
  `data.subscription.{creditBalance, autoRecharge, rechargeThreshold, autoRechargeError, plan}`.
  Consider a shared `profileResponse` type in `credits.go` reused by `usage.go` to
  avoid two divergent decoders.
- `ledgerEntry{ EntryID, Type string; Amount float64; BalanceAfter *float64;
  Description *string; CreatedAt string }`.
- `autoRechargeRequest{ Enabled bool; Threshold *int }` (omitempty on threshold) and
  `autoRechargeResponse{ AutoRecharge bool; RechargeThreshold int }`.

## Testing

The repo currently has unit tests at the API and push-logic level
(`internal/api/client_test.go`, `internal/cli/push_logic_test.go`) but **no Cobra
command harness**. This work adds a thin one: a helper that builds the command with
an `httptest.Server` base URL and captured stdout/stderr, returning output + error,
reused across credits tests. Pure formatting (balance line, ledger row, amount sign)
is also factored into small functions tested directly without HTTP.

- Tests with `httptest.Server` for: balance render (normal / enterprise=`unlimited` /
  negative / autoRechargeError warning), ledger (populated table with signed amounts
  rendered verbatim, empty, exactly-50 footer), auto-recharge (show / enable /
  enable+threshold / disable / 400 NO_PAYMENT_METHOD via `resp` inspection / both
  flags error / `--threshold` without `--enable` / `--disable --threshold` /
  out-of-range threshold), buy (human mode prints URL + calls stubbed `openURL`;
  `--json` prints raw body and does **not** call `openURL`; `openURL` error still
  exits 0). Assert the exit-code contract (usage errors → 2) via `errors.Is(err,
  ErrUsage)`.
- The `openURL` package var is stubbed in buy tests (no real browser launch).
- `usage` test updated: no longer expects a pages *limit*; asserts the
  `PDF pages generated: N` count line and the credit-balance line are present.
- `scripts/smoke.sh`: add `mkp credits` and `mkp credits ledger` read-only checks vs
  dev.
- `make test` and `make build` green before commit.

## Out of scope (follow-up)

**#2 Headless CI for templates/tokens.** All `templates`/`tokens` routes are JWT-only
behind the Cognito Gateway authorizer (`iamOnlyMiddleware`, `allowApiToken:false`).
Enabling API-key auth requires new backend routes (`/v1/templates/*`, possibly
`/v1/tokens`) with `apiKeyOnlyMiddleware` and **no** Gateway authorizer — mirroring
`/v1/pdf/generate` and its security note (a forged JWT must not pass on an
authorizer-less route). This is primarily backend work and gets its own spec; the CLI
side (a `--api-key` path on `templates`/`tokens`) is trivial once those routes exist.

## Non-goals

- No pagination for the ledger (backend returns a fixed 50; documented as such).
- No quantity selection for `buy` (backend pack is fixed).
- No offline/cached balance.
