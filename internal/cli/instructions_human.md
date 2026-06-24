# mkpdfs — quick guide

mkpdfs turns a Handlebars `.hbs` template (arbitrary HTML + CSS, rendered by
headless Chromium) into a PDF. You author a template, push it, then generate
PDFs from it plus JSON data.

The workflow, end to end:

  1. Log in        mkp auth login            # browser/device flow
  2. Check session  mkp auth whoami           # email, plan, active env
  3. Push template  mkp templates push carta.hbs
  4. Generate PDF   mkp pdf generate -t carta.hbs -d datos.json -o carta.pdf

Read a specific topic with a flag:

  mkp instructions --format        # .hbs format: HTML/CSS, @page, variables, helpers
  mkp instructions --auth          # authentication: login, whoami, --api-key
  mkp instructions --environments  # dev vs prod and how to switch
  mkp instructions --plans         # plans, credits and limits
  mkp instructions --agent         # everything, framed for an AI coding agent

You can combine topic flags, e.g. `mkp instructions --format --plans`.

Building with an AI agent? Run `mkp instructions --agent` and hand the output to
it — a complete, copy-pasteable walkthrough written for coding agents.
