## Worked example (copy-pasteable)

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

### Push to dev

```bash
mkp --env dev templates push carta.hbs
```

Validates the Handlebars locally, uploads it, prints a `templateId`, and writes a
`.mkpdfs.json` mapping `carta.hbs` → templateId (binding this directory to dev).
Re-running `push` updates the same template.

### Generate a PDF and verify

```bash
mkp --env dev pdf generate -t carta.hbs -d datos.json -o carta.pdf
```

`-t` accepts the local `.hbs` (resolved via `.mkpdfs.json`) or a raw templateId.
**Confirm `carta.pdf` was actually written** before telling the user you're done;
open it if you can. To generate many PDFs at once, make `datos.json` a JSON
**array** of objects (**max 50 items**).
