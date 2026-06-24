## Plans, credits & limits

mkpdfs is **prepaid credits**: **$10 = 1,000 credits, 1 credit = 1 PDF page**
(the same unit as a generated PDF's page count). Credits never expire. New
accounts get **10 welcome credits**. When the balance hits zero, PDF generation
returns **402 INSUFFICIENT_CREDITS** until you top up.

Two plans:

| | **credits** (default) | **enterprise** (manual) |
|---|---|---|
| Templates | 500 | unlimited |
| API tokens | 10 | unlimited |
| Max PDF size | 50 MB | 100 MB |
| AI generations / month | 15 | unlimited |

AI generation uses a fixed monthly quota (it does NOT spend credits) but still
requires a positive credit balance. `enterprise` is provisioned manually
(Contact Sales).

Manage credits from the CLI:

- `mkp credits` — current balance + auto-recharge status
- `mkp credits buy` — buy a 1,000-credit pack (opens Stripe checkout)
- `mkp credits ledger` — recent credit movements
- `mkp credits auto-recharge --enable [--threshold N]` — auto top-up with a saved card
- `mkp usage` — current-month usage stats

`mkp auth whoami` shows which plan the account is on.
