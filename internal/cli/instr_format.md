## Template format

A `.hbs` file is **plain HTML with inline CSS** plus Handlebars `{{placeholders}}`,
rendered by headless Chromium (so flexbox, grid, `box-shadow`, and web fonts all
work).

- **Page size** is set in CSS, not a flag:
  `@page { size: A4; margin: 2cm; }` (use `size: Letter` for US Letter).
- **Variables**: `{{nombre}}` is replaced with the value of `nombre` from the JSON
  data file you pass to `pdf generate`.
- **Source size cap: 6.5 MiB** per `.hbs` (`templates push` rejects larger files).

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
yourself in the JSON data and emit the string with a plain `{{variable}}` — don't
rely on `formatDate`/`formatCurrency` for that.
