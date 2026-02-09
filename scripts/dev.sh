#!/usr/bin/env bash
# Build and launch the devcontainer using the @devcontainers/cli.
# All container config (build args, mounts, features) lives in devcontainer.json.
# Usage: dev.sh
set -euo pipefail

WORKSPACE="$(cd "$(dirname "$0")/.." && pwd)"

# Resolve the devcontainer CLI — prefer a local install, fall back to npx
if command -v devcontainer &>/dev/null; then
  DC="devcontainer"
elif command -v npx &>/dev/null; then
  DC="npx --yes @devcontainers/cli"
else
  echo "ERROR: Neither 'devcontainer' nor 'npx' found on PATH." >&2
  echo "       Install the CLI: npm install -g @devcontainers/cli" >&2
  exit 1
fi

echo "==> Starting devcontainer..."
$DC up --workspace-folder "$WORKSPACE"

echo "==> Attaching shell..."
exec $DC exec --workspace-folder "$WORKSPACE" bash
