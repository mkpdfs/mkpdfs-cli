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
CLI already has `jwtClient()` for exactly this, used by `usage`.

## Command surface

New file `internal/cli/credits.go`, wired in `root.go` via `addCreditsCommands()`.
Parent command + four behaviors. All accept the global `--json`/`--env` flags and
use `jwtClient()`.

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
- Parent with an action: unlike pure parents, a bare `mkp credits` runs this (does
  not print help). Unknown subcommands still exit 2 via the existing guard pattern.

### `mkp credits ledger`

Reads `GET /billing/ledger`. Table columns: `DATE  TYPE  AMOUNT  BALANCE  DESCRIPTION`.
- `amount` shown with sign; debits/refunds negative. Color used only when stdout is
  a TTY and `--json` is off (green for credits in, default for debits) — never break
  `--json` or piped output.
- `balanceAfter` null → blank cell. `description` null → blank.
- Empty list → `No ledger entries yet.`
- Exactly 50 rows → footer `(showing most recent 50)`.
- `--json` prints the raw response body.

### `mkp credits auto-recharge`

- **No flags:** read `/user/profile`, print current setting
  (`Auto-recharge is on (threshold 100)` / `... is off`).
- **`--enable`:** `PUT { enabled:true, threshold:<N or omitted> }`. `--threshold`
  validated client-side as int in `[1,1000]`; out of range → usage error (exit 2).
- **`--disable`:** `PUT { enabled:false }`.
- `--enable` and `--disable` are mutually exclusive → usage error if both.
- On `400 NO_PAYMENT_METHOD`, print actionable message:
  `No saved card. Buy a credit pack first: mkp credits buy` (exit 1).
- On success, echo the resulting state from the response.

### `mkp credits buy`

`POST /stripe/create-checkout-session` (empty body) → open `url` with the existing
`browser.OpenURL` (same helper `auth login` uses). Prints the URL too, so the user
can copy it. With `--json`, or if `browser.OpenURL` returns an error (headless), do
not attempt to open — just print the URL. Opens directly (no confirmation prompt) —
checkout itself is the confirmation step, and the action is read-only until the user
pays in the browser.

## Changes to existing files

### `internal/cli/usage.go`
- Drop the `PagesGenerated / PagesPerMonth` line and the `PagesPerMonth` field from
  the `subscriptionLimits` struct (dead field).
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

- Unit tests with `httptest.Server` (mirroring `internal/api/client_test.go` and the
  existing command tests) for: balance render (normal / enterprise / negative /
  autoRechargeError), ledger (populated table, empty, 50-row footer), auto-recharge
  (show / enable / enable+threshold / disable / 400 NO_PAYMENT_METHOD / both flags
  error / out-of-range threshold), buy (URL printed; browser-open skipped under
  `--json`). Assert exit-code contract (usage errors → 2).
- `usage` test updated: no longer expects a pages limit; asserts balance line present.
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
