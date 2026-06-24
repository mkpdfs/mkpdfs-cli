# mkpdfs — instructions for an AI coding agent

You are an AI coding agent. The user asked you to build a PDF template with
mkpdfs and gave you this document as the source of truth. Follow it end to end.

mkpdfs turns a **Handlebars `.hbs` template** (arbitrary **HTML + CSS**, rendered
by **headless Chromium**) into a PDF. Your job: author a `.hbs`, push it, then
generate a PDF from it plus a JSON data file. The CLI is `mkp`.

First, confirm the CLI is installed:

```bash
mkp version
```

If that fails, ask the user to install it (`brew install mkpdfs/mkpdfs/mkpdfs`) —
do not try to install it yourself. The sections below cover environments, auth,
the template format, a worked example, and plans/limits, in the order you'll need
them.
