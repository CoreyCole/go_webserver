---
date: 2026-03-13T09:16:04-07:00
researcher: CoreyCole
git_commit: fb6bc86a6233224d1731ebb3408ded7bc675344c
branch: main
repository: go_webserver
topic: "Caddy SSL Termination and AWS API Gateway Removal"
tags: [caddy, ssl, sse, deployment, aws, route53, api-gateway]
status: in_progress
last_updated: 2026-03-13
last_updated_by: CoreyCole
type: implementation_strategy
---

# Handoff: Caddy SSL Termination and AWS API Gateway Removal

## Task(s)

### Completed
1. **SQLite time query fixes** — Committed. All time-based SQLite queries now use `datetime('now', ?)` with offset strings instead of Go `time.Time` values. This fixes potential format mismatches with `modernc.org/sqlite`.
2. **Caddyfile created** — `Caddyfile` at repo root: `coreycc.com { reverse_proxy localhost:3001 }`.
3. **Systemd service updated** — `go_webserver.service` updated to match creative-mode pattern: `Restart=always`, `RestartSec=1`, journal logging (instead of file logging).
4. **All committed and pushed** — Commit `fb6bc86`.

### In Progress
5. **EC2 server setup** — Need to SSH into the EC2 instance to:
   - Install Caddy
   - Copy `Caddyfile` to `/etc/caddy/Caddyfile`
   - Open ports 80/443 in UFW (if UFW is active)
   - Deploy updated code and restart services
   - Delete old `data/go_webserver.db` so migrations run fresh

6. **AWS infrastructure changes** — Need to:
   - Update Route 53 A record: `coreycc.com` → `52.32.199.228` (currently aliased to API Gateway)
   - Delete API Gateway `3pk16lof2m` (HTTP API v2, named `coreycc.com`)
   - Optionally delete ACM certificate

## Critical References
- `context/creative-mode/thoughts/CoreyCole/plans/2026-02-16_03-41-53_caddy-reverse-proxy-sse.md` — Full implementation plan from creative-mode site (same pattern)
- `context/creative-mode/site/Caddyfile` — Reference Caddyfile
- `context/creative-mode/site/creative-mode-site.service` — Reference systemd service

## Recent changes

### Commit `fb6bc86` — "Use SQLite-native time queries and add Caddy reverse proxy config"
- `webserver/db/queries/metrics_snapshots.sql` — `WHERE created_at >= datetime('now', ?)` instead of `WHERE created_at >= ?`
- `webserver/db/queries/page_views.sql` — Same pattern for `DeletePageViewsOlderThan`
- `webserver/db/sqlc/metrics_snapshots.sql.go` — Regenerated, params changed from `time.Time` to `interface{}`
- `webserver/db/sqlc/page_views.sql.go` — Same
- `webserver/db/sqlc/querier.go` — Same
- `webserver/db/db.go` — `CompactOldMetrics` takes `string` offset instead of `time.Time`
- `webserver/handle/status.go` — Added `sqliteOffset()` helper, `doRetentionCleanup` passes string offsets like `"-3 days"`, `"-30 days"`
- `Caddyfile` — New file: `coreycc.com { reverse_proxy localhost:3001 }`
- `go_webserver.service` — Updated: `Restart=always`, `RestartSec=1`, journal logging, `network-online.target`

## Learnings

### SSE + AWS API Gateway = broken
AWS API Gateway has a ~30s integration timeout and buffers responses, which kills long-lived SSE connections. The creative-mode site hit this exact issue and solved it by switching to Caddy for TLS termination on EC2 directly. The working pattern:
```
Browser → Route 53 → EC2 Elastic IP → Caddy:443 (TLS) → localhost:PORT
```
Caddy handles automatic HTTPS via Let's Encrypt and properly proxies SSE streams without buffering.

### AWS resource IDs discovered
- **API Gateway (HTTP v2)**: `3pk16lof2m` (name: `coreycc.com`)
- **Elastic IP**: `52.32.199.228` (instance: `i-0ec415fe12d69de3f`, tag: `coreycc.com`)
- **Route 53 Hosted Zone**: `Z09323732C979YPP6E902`
- **Current DNS**: A record alias → `d-57mhp83bu9.execute-api.us-west-2.amazonaws.com`
- There is also a `datastar-ui.com` API Gateway (`hgtkln6ux9`) — do NOT touch this

### Order of operations matters
Must install Caddy and deploy code on EC2 BEFORE switching DNS, otherwise the site breaks (no TLS termination between DNS switch and Caddy being ready).

## Artifacts
- `Caddyfile` — Caddy reverse proxy config for coreycc.com
- `go_webserver.service` — Updated systemd unit file
- `context/creative-mode/thoughts/CoreyCole/plans/2026-02-16_03-41-53_caddy-reverse-proxy-sse.md` — Reference implementation plan (Phase 2 and 3 are most relevant)

## Action Items & Next Steps

### 1. SSH into EC2 and set up Caddy
The EC2 instance is at `52.32.199.228`. SSH in and:
```bash
# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install -y caddy

# Open firewall ports (if UFW active)
sudo ufw allow 80/tcp && sudo ufw allow 443/tcp

# Deploy updated code
cd ~/go_webserver && git pull
# Build (just build or however it's built on the server)

# Copy configs
sudo cp ~/go_webserver/Caddyfile /etc/caddy/Caddyfile
sudo cp ~/go_webserver/go_webserver.service /etc/systemd/system/
sudo systemctl daemon-reload

# Delete old database (timestamps may be in incompatible format)
rm -f ~/go_webserver/data/go_webserver.db

# Restart services
sudo systemctl restart go_webserver
sudo systemctl enable --now caddy
```

### 2. Verify Caddy is working locally on EC2
```bash
curl -N http://localhost:3001/status/events  # should stream SSE
sudo systemctl status caddy                   # should be active
sudo systemctl status go_webserver            # should be active
```

### 3. Update Route 53 (from local machine)
```bash
aws route53 change-resource-record-sets --hosted-zone-id Z09323732C979YPP6E902 --change-batch '{
  "Changes": [{"Action": "UPSERT", "ResourceRecordSet": {
    "Name": "coreycc.com", "Type": "A", "TTL": 300,
    "ResourceRecords": [{"Value": "52.32.199.228"}]
  }}]
}'
```

### 4. Delete API Gateway (after DNS propagation, ~5 min)
```bash
aws apigatewayv2 delete-api --api-id 3pk16lof2m
```

### 5. Optionally delete ACM certificate
```bash
aws acm list-certificates --query 'CertificateSummaryList[?DomainName==`coreycc.com`]' --output table
aws acm delete-certificate --certificate-arn <ARN>
```

### 6. Verify everything end-to-end
- `https://coreycc.com/status` shows live metrics via SSE
- Graph populates after 30s
- Traffic counts increment
- Theme toggle works
- `dig coreycc.com` shows `52.32.199.228`

## Other Notes
- The go_webserver port is `3001` (configured via `PORT` env var, default in code). Creative-mode uses `3000`.
- Do NOT touch the `datastar-ui.com` API Gateway or Route 53 zone.
- The `data/` directory is gitignored and auto-created by `db.New()`.
- The creative-mode site's Caddy plan doc (Phase 2 and 3) has the exact commands needed — this handoff adapts them for go_webserver.
- ACM certificate listing command should be run to find the ARN before deletion.
