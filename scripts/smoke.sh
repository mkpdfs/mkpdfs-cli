#!/usr/bin/env bash
# End-to-end smoke vs dev through the real binary. Requires prior `mkp auth login --env dev`.
set -euo pipefail

BIN="$(cd "$(dirname "$0")/.." && pwd)/mkp-cli"

if [[ ! -x "$BIN" ]]; then
  echo "ERROR: binary not found at $BIN — run 'make build' first" >&2
  exit 1
fi

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
cd "$TMP"

echo '<h1>{{title}}</h1>' > smoke.hbs
echo '{"title":"Smoke"}' > data.json

echo "--- push smoke.hbs ---"
"$BIN" --env dev templates push smoke.hbs --yes

echo "--- generate PDF ---"
"$BIN" --env dev pdf generate -t smoke.hbs -d data.json -o smoke.pdf

[ -s smoke.pdf ] && echo "OK: smoke.pdf generated ($(wc -c < smoke.pdf) bytes)"

echo "--- credits balance (read-only) ---"
"$BIN" --env dev credits

echo "--- credits ledger (read-only) ---"
"$BIN" --env dev credits ledger

echo "--- read templateId from .mkpdfs.json ---"
ID=$(python3 -c "import json;print(json.load(open('.mkpdfs.json'))['templates']['smoke.hbs']['templateId'])")
echo "templateId: $ID"

echo "--- delete template ---"
"$BIN" --env dev templates delete "$ID" --force

echo "SMOKE PASSED"
