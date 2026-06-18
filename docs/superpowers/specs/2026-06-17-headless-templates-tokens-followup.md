# Follow-up stub — headless API-key access for templates/tokens (#2)

**Date:** 2026-06-17
**Status:** Deferred (captured so it isn't lost; not yet planned in detail)
**Primary repo:** `mkpdfs-backend` (CLI side is trivial once routes exist)
**Related:** `2026-06-17-credits-cli-design.md` (this was carved out of that round)

## Problem

The CLI can only do `pdf generate` headlessly (with `MKPDFS_API_KEY` / `--api-key`).
`templates` (list/pull/push/delete), `tokens` (create/list/revoke), `usage`, and the
new `credits` group all require a browser/Cognito login, so a CI pipeline can't, e.g.,
push a template or rotate a token without a human at a workstation.

## Root cause (verified 2026-06-17)

All those routes are JWT-only behind the Cognito **Gateway authorizer** and use
`iamOnlyMiddleware()` (`dualAuthMiddleware({ requireAuth:true, allowApiToken:false })`
in `src/libs/middleware/dualAuth.ts`). Two layers reject API keys:

1. The Gateway authorizer rejects any request without a valid Cognito JWT *before*
   the lambda runs — so even swapping the in-lambda middleware to `dualAuth` would
   still 401 at the gateway for an API-key request.
2. `allowApiToken:false` in the middleware.

API tokens *do* carry `userId` (SHA256 lookup in the tokens table returns it), so
ownership filtering would work identically to JWT — the blocker is purely routing/auth,
not data scoping.

## Sketch of the work (backend)

Mirror the `/v1/pdf/generate` pattern: add parallel **`/v1/*`** routes with **no
Gateway authorizer** and `apiKeyOnlyMiddleware()` (NOT dualAuth — the
`/v1/pdf/generate` security note explains why: on an authorizer-less route a forged
JWT must not be accepted, so the route must require an API key).

Candidate routes (server-to-server subset; pick what CI actually needs):
- `GET /v1/templates`, `GET /v1/templates/{id}`, `POST /v1/templates`,
  `PUT /v1/templates/{id}`, `DELETE /v1/templates/{id}`
- Possibly `POST /v1/tokens` etc. — but minting tokens from an API key is a
  privilege-escalation surface; consider leaving token management JWT-only.

Notes / decisions to make when this is planned for real:
- Reuse the existing handlers (they already key off `event.userId`); only the
  middleware chain and route wiring change.
- This is the same shape as the known backlog item for `/v1/jobs/submit`.
- Subscription/usage middleware should still apply on the `/v1` variants.

## CLI side (small)

Add an `--api-key` path to `templates` (and any chosen `tokens`) commands that targets
the `/v1/*` route and uses `client.WithAPIKey()` + `PostWithKey`-style calls (the
client already supports `x-api-key`). Update the README CI section to broaden what's
headless.

## Why deferred

It's primarily a backend change with a security surface (authorizer-less routes), so
it warrants its own design pass rather than riding along with the CLI-only credits
work. No code written yet.
