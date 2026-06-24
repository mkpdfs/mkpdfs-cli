# mkpdfs — quick guide

mkpdfs turns a Handlebars `.hbs` template (arbitrary HTML + CSS, rendered by
headless Chromium) into a PDF. You author a template, push it, then generate
PDFs from it plus JSON data.

The workflow, end to end:

  1. Log in        mkp auth login            # browser/device flow
  2. Check session  mkp auth whoami           # email, plan, active env
  3. Push template  mkp templates push carta.hbs
  4. Generate PDF   mkp pdf generate -t carta.hbs -d datos.json -o carta.pdf

Notes:

  * The default environment is PROD. Add `--env dev` to any command to work on
    dev first (recommended while iterating), e.g. `mkp --env dev templates push …`.
  * Page size lives in the template CSS: `@page { size: A4; margin: 2cm }`.
  * `--api-key` (with MKPDFS_API_KEY) runs templates/pdf commands headless, with
    no browser login — for CI and servers.

Building with an AI agent? Run `mkp instructions --agent` and hand the output to
it — it's a complete, copy-pasteable walkthrough written for coding agents.
