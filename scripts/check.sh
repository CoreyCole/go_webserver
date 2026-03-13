#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
failed=0

# Phase 1: Format (modifies files, must run first).
echo "formatting..."
(cd "$ROOT" && templ fmt webserver/ 2>&1)
(cd "$ROOT" && golangci-lint fmt ./... 2>&1)

# Phase 2: Generate (sqlc + templ + tailwind — catches template errors early).
echo "generating..."
(cd "$ROOT" && sqlc generate 2>&1)
(cd "$ROOT" && templ generate -f webserver/view 2>&1)
(cd "$ROOT" && pnpm exec tailwindcss -i webserver/view/css/index.css -o public/build.css --content "./webserver/view/**/*" 2>&1)

# Phase 3: Lint (includes compilation checks).
echo "linting..."
lint_out=$(mktemp)
trap 'rm -f "$lint_out"' EXIT

if (cd "$ROOT" && golangci-lint run ./... 2>&1) >"$lint_out" 2>&1; then
    echo "  ok  golangci-lint"
else
    echo "FAIL  golangci-lint" >&2
    cat "$lint_out" >&2
    failed=1
fi

# Phase 4: Build.
echo "building..."
build_out=$(mktemp)
trap 'rm -f "$lint_out" "$build_out"' EXIT

if (cd "$ROOT" && go build ./... 2>&1) >"$build_out" 2>&1; then
    echo "  ok  go build"
else
    echo "FAIL  go build" >&2
    cat "$build_out" >&2
    failed=1
fi

# Exit code 2 = blocking hook failure (forces Claude to fix issues before stopping)
if [ "$failed" -ne 0 ]; then
    exit 2
fi
