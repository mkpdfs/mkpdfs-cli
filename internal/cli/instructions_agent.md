# mkpdfs — instructions for an AI coding agent

You are an AI coding agent. The user asked you to build a PDF template with
mkpdfs and gave you this document as the source of truth. Follow it end to end.

mkpdfs turns a **Handlebars `.hbs` template** (arbitrary **HTML + CSS**, rendered
by **headless Chromium**) into a PDF. Your job: author a `.hbs`, push it, then
generate a PDF from it plus a JSON data file. The CLI is `mkp`.

---

## 0. Environments — read this first

**The default environment is PROD (live).** While building and iterating, always
work on **dev** by passing `--env dev` to every command (shown throughout below).
Do not push or generate against prod until the template is verified on dev.

(Optional: the user can make dev the default with `mkp config set environment dev`,
but prefer explicit `--env dev` so you never mutate their global config.)

## 1. Confirm the CLI is installed

```bash
mkp version
```

If that fails, ask the user to install it: `brew install mkpdfs/mkpdfs/mkpdfs`.
Do not try to install it yourself.

## 2. Authentication

`mkp auth login` is an **interactive browser/device flow** — you (an agent)
cannot complete it. So:

- **If the user is not logged in, ASK THEM to run** `mkp --env dev auth login`
  and wait for them to confirm.
- Verify the session yourself with:

  ```bash
  mkp --env dev auth whoami      # prints email, plan, active env
  ```

  (There is no `auth status` — the command is `whoami`.)

- **Headless / CI alternative:** if the user gives you an API key, you can skip
  the browser entirely. Set `MKPDFS_API_KEY=tlfy_…` (or a saved key) and add
  `--api-key` to `templates` and `pdf` commands. No login needed.

## 3. The template format

A `.hbs` file is **plain HTML with inline CSS** plus Handlebars `{{placeholders}}`.

- **Page size** is set in CSS, not a flag:
  `@page { size: A4; margin: 2cm; }` (use `size: Letter` for US Letter).
- **Variables**: `{{nombre}}` is replaced with the value of `nombre` from the
  JSON data file you pass to `pdf generate`.
- Anything valid in Chromium works: flexbox, grid, `box-shadow`, web fonts, etc.
- **Source size cap: 6.5 MiB** per `.hbs` (the push will reject larger files).

### Helpers (exact signatures — do NOT invent arguments)

| Helper | Usage | Notes |
|---|---|---|
| `ifEq` | `{{#ifEq a b}}…{{else}}…{{/ifEq}}` | block; loose `==` equality |
| `gt` | `{{#if (gt a b)}}…{{/if}}` | returns a boolean; use as a subexpression |
| `formatDate` | `{{formatDate someDate}}` | **no format argument**; renders `toLocaleDateString()` |
| `formatCurrency` | `{{formatCurrency amount}}` | **always USD**; no currency argument |
| `mkpdfsQR` | `{{{mkpdfsQR "https://example.com"}}}` | inline SVG QR code; triple-stache (raw) |

Built-in Handlebars helpers also work: `{{#each list}}…{{/each}}`,
`{{#if x}}…{{/if}}`, `{{#unless x}}…{{/unless}}`, `{{#with obj}}…{{/with}}`,
`{{else}}`. Inside `{{#each}}`, the current item is `{{this}}`.

If you need a formatted date or non-USD currency in a specific style, format it
yourself in the JSON data and emit the string with a plain `{{variable}}` —
don't rely on `formatDate`/`formatCurrency` for that.

## 4. Worked example (copy-pasteable)

Write these two files. They are complete and valid.

**`carta.hbs`** — an A4 love letter:

```hbs
<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<style>
  @page { size: A4; margin: 2.5cm; }
  body { font-family: Georgia, "Times New Roman", serif; color: #2b2b2b; line-height: 1.7; }
  .fecha { text-align: right; color: #777; font-size: 12px; margin-bottom: 2rem; }
  .saludo { font-size: 22px; color: #b23a48; margin-bottom: 1rem; }
  p { margin: 0 0 1rem; text-align: justify; }
  .firma { margin-top: 3rem; font-style: italic; font-size: 18px; }
</style>
</head>
<body>
  <div class="fecha">{{formatDate fecha}}</div>
  <div class="saludo">Querida {{para}},</div>
  {{#each parrafos}}
  <p>{{this}}</p>
  {{/each}}
  <div class="firma">Siempre tuyo,<br>{{de}}</div>
</body>
</html>
```

**`datos.json`** — the data fed to the template:

```json
{
  "para": "Mariana",
  "de": "Alejandro",
  "fecha": "2026-06-24",
  "parrafos": [
    "Cada mañana desde que te conocí amanece distinta, más clara.",
    "No sé escribir versos, así que te escribo la verdad: me haces feliz.",
    "Guarda esta carta; es el recibo de todo lo que no sé decirte en voz alta."
  ]
}
```

## 5. Push the template to dev

```bash
mkp --env dev templates push carta.hbs
```

This validates the Handlebars locally, uploads it, prints a `templateId`, and
writes a `.mkpdfs.json` in the current directory that maps `carta.hbs` →
templateId **and binds this directory to dev**. Re-running `push` updates the
same template.

## 6. Generate a PDF and verify

```bash
mkp --env dev pdf generate -t carta.hbs -d datos.json -o carta.pdf
```

`-t` accepts the local `.hbs` (resolved via `.mkpdfs.json`) or a raw templateId.
**Confirm `carta.pdf` was actually written** before telling the user you're done;
open it if you can. To generate many PDFs at once, make `datos.json` a JSON
**array** of objects (**max 50 items**).

## 7. Promote to prod (only after dev looks right)

`.mkpdfs.json` is **bound to one environment**; the CLI rejects cross-env
operations, so you cannot just switch to `--env prod` and push from this same
dev-bound directory. Safe path:

1. Ask the user to log in to prod: `mkp --env prod auth login`.
2. From a **separate directory** (or one with no dev-bound `.mkpdfs.json`), push
   the same file: `mkp --env prod templates push carta.hbs`.

Prod is live and `mkp` prompts for confirmation before writing there. Treat every
prod command as publishing.

## 8. Limits & guards (recap)

- Template source: **6.5 MiB** max per `.hbs`.
- Batch generation: **50** items max per `pdf generate`.
- `.mkpdfs.json` is **env-bound** — no cross-env push/pull/generate.
- Handlebars is **validated locally** before every push (syntax errors block it).
- `formatDate` has no format arg; `formatCurrency` is USD-only — pre-format in the
  data when you need anything else.
