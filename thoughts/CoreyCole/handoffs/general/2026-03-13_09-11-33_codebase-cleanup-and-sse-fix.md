---
date: 2026-03-13T09:11:33-07:00
researcher: CoreyCole
git_commit: 4b8338d3bc6a91e9a435ad3f175bf0785e56c124
branch: main
repository: go_webserver
topic: "Codebase Cleanup, Bug Fixes, and SSE Deployment Fix"
tags: [cleanup, bugfix, sse, deployment, caddy, sqlite, lint]
status: in_progress
last_updated: 2026-03-13
last_updated_by: CoreyCole
type: implementation_strategy
---

# Handoff: Codebase Cleanup and SSE Deployment Fix

## Task(s)

### Completed
1. **Bug fixes** -- Fixed path traversal vulnerability in markdown handler (`handle/md.go`), unclosed `<p>` tag in resume HTML (`lib/resume_json_to_html.go:110`), dead error check in markdown handler.
2. **Dead code removal** -- Removed unused `calculateListDepth` function, dead `style` constant `"monokai"`, dead `li.Attribute` modification code.
3. **Config cleanup** -- Removed `react/` from `.air.toml` include_dir, removed stale database migration targets from Makefile (Makefile then deleted entirely), removed unnecessary `replace` directive from `go.mod`.
4. **Feature: nested markdown paths** -- Changed route from `/md/:filename` to `/md/*` with `c.Param("*")` so blog posts at `public/md/blog/` are now accessible.
5. **Feature: configurable port** -- Added `kelseyhightower/envconfig` with `PORT` env var (default `3001`).
6. **Favicon** -- Replaced favicon with rocket from creative-mode (`public/favicons/favicon-32x32.png`).
7. **CLAUDE.md** -- Complete rewrite reflecting current architecture.
8. **Lint fixes** -- Fixed all 55 golangci-lint issues: errcheck, naming conventions (Cpu→CPU), magic numbers, naked returns, variable shadowing, gosec nolint placement, strings.Builder for path concat, etc.
9. **Status page UI** -- Removed debug "N points" display, added x-axis padding, matched content width to header (`max-w-5xl`).
10. **SQLite time queries** -- Changed all SQLite queries to use `datetime('now', ?)` with offset strings instead of Go `time.Time` values. The `modernc.org/sqlite` driver serializes `time.Time` in a format that may not match SQLite's `CURRENT_TIMESTAMP` text format.
11. **Git history** -- Created 7 clean commits and pushed to origin/main.

### In Progress / Blocked
12. **SSE not working on Linux deployment** -- The status page renders but SSE events never arrive. System metrics show `--`, graph shows "Collecting data...", traffic shows 0. **Root cause: AWS API Gateway kills SSE connections** (same issue hit with creative-mode site). Need to switch from API Gateway to Caddy for SSL termination, matching the creative-mode deployment pattern.

## Critical References
- `context/creative-mode/site/CLAUDE.md` -- Creative-mode site deployment docs showing Caddy + Route 53 + EC2 pattern that works with SSE
- `thoughts/CoreyCole/handoffs/general/2026-03-12_22-52-08_codebase-cleanup-and-review.md` -- Prior handoff with full codebase review findings

## Recent changes

### Commits pushed (7 total, in order)
- `019c194` Remove React code and frontend tooling
- `9fd5d3d` Migrate WASM game assets to S3
- `da1f3ce` Redesign layout with templ, Datastar, and Tailwind v4
- `8071ce8` Add status page with SSE metrics, SQLite database, and build tooling
- `438e3c5` Fix bugs, remove dead code, and add envconfig
- `4d84b2e` Update favicon, CLAUDE.md, and resume
- `4b8338d` Fix lint issues and improve status page layout

### Uncommitted changes (need to commit + push + deploy)
- `webserver/db/queries/metrics_snapshots.sql` -- Changed `WHERE created_at >= ?` to `WHERE created_at >= datetime('now', ?)`
- `webserver/db/queries/page_views.sql` -- Same pattern for `DeletePageViewsOlderThan`
- `webserver/db/sqlc/*.go` -- Regenerated (params changed from `time.Time` to `interface{}`)
- `webserver/db/db.go` -- `CompactOldMetrics` now takes string offset instead of `time.Time`, uses `datetime('now', ?)` in raw SQL
- `webserver/handle/status.go` -- Added `sqliteOffset()` helper, `buildGraph` and `doRetentionCleanup` pass SQLite offset strings

## Learnings

### SSE + AWS API Gateway = broken
AWS API Gateway kills long-lived SSE connections (30-second timeout). The creative-mode site hit this exact same issue and solved it by switching to **Caddy** for SSL/TLS termination on the EC2 instance directly. The working pattern from creative-mode:
```
Browser → Route 53 → EC2 Elastic IP → Caddy:443 (TLS) → localhost:3000
```
Caddy handles automatic HTTPS via Let's Encrypt and properly proxies SSE streams without buffering or timeout issues.

### SQLite time comparison gotcha
`modernc.org/sqlite` (pure Go SQLite) may serialize Go `time.Time` differently than how `CURRENT_TIMESTAMP` stores timestamps as text. The creative-mode site avoids this by using `datetime('now', ?)` with SQLite offset strings like `"-1 hour"`, `"-30 days"` in all time-based queries. This keeps time comparison logic entirely within SQLite.

### Lint patterns established
The codebase now passes `golangci-lint` clean. Key patterns: `_, _ = w.Write(...)` for render hook writes, `//nolint:gosec` on the line ABOVE the call (not same line -- formatter moves it), constants for all magic numbers, `strings.Builder` for loop concatenation, explicit returns (no naked returns).

## Artifacts
- `CLAUDE.md` -- Comprehensive developer guide (rewritten)
- `webserver/handle/status.go` -- Status page handler with all lint fixes + SQLite offset queries
- `webserver/view/status.templ` -- Status page template (UI fixes, `CPUPath`/`CPULast` naming)
- `webserver/view/status.go` -- GraphData struct (renamed fields)
- `webserver/db/queries/metrics_snapshots.sql` -- SQLite-native time queries
- `webserver/db/queries/page_views.sql` -- SQLite-native time queries
- `webserver/db/db.go` -- CompactOldMetrics with string offset
- `webserver/lib/markdown_to_html.go` -- All lint fixes, renamed `NewMarkdownToHTMLRenderer`
- `go_webserver.service` -- Systemd unit file (current deployment config)

## Action Items & Next Steps

### 1. Commit and push SQLite time query changes
The `datetime('now', ?)` changes are uncommitted. Commit and push them.

### 2. Switch SSL from AWS API Gateway to Caddy
This is the critical fix for SSE. Follow the creative-mode site pattern:
- Install Caddy on the EC2 instance
- Configure Caddy to handle TLS (automatic HTTPS via Let's Encrypt) for `coreycc.com`
- Proxy to `localhost:3001` (or whatever PORT is configured)
- Remove AWS API Gateway from the traffic path
- Update Route 53 to point directly at the EC2 Elastic IP

Reference: `context/creative-mode/site/CLAUDE.md` has the full deployment topology.

### 3. Delete the old database on the server
After deploying the SQLite query changes, delete `data/go_webserver.db` on the server so migrations run fresh with the corrected schema. The old data may have timestamps in an incompatible format.

### 4. Update systemd service
Consider updating `go_webserver.service` to match creative-mode patterns:
- `Restart=always` instead of `Restart=on-failure`
- `RestartSec=1` instead of `RestartSec=10`
- Logs to `journal` instead of file (easier with `journalctl`)

### 5. Verify on Linux after deployment
After Caddy is set up and the app is redeployed, verify:
- SSE connection establishes (system metrics update every 2s)
- Graph populates (data points appear after 30s)
- Traffic counts increment on page visits
- Theme toggle works
- Markdown pages render (including nested blog paths)

## Other Notes
- The creative-mode site deployment docs mention the exact same SSE issue was root-caused to API Gateway: `thoughts/CoreyCole/handoffs/general/2026-02-16_03-38-09_status-page-sse-root-cause.md` (in the creative-mode repo)
- The go_webserver uses `modernc.org/sqlite` (pure Go, no CGO) which is the same as creative-mode
- Datastar client version is v1.0.0-RC.8 (vendored locally), creative-mode uses RC.7 from CDN
- Server has no CORS middleware (shouldn't matter for same-origin SSE)
- The `data/` directory is gitignored; it's created automatically by `db.New()`
- Port is configurable via `PORT` env var (default 3001)
- Current domain: `coreycc.com` pointing to AWS infrastructure
