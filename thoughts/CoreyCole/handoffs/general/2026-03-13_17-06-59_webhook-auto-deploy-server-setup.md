---
date: 2026-03-13T17:06:59-07:00
researcher: CoreyCole
git_commit: 49416d13709b8e78bed1c6361b70f03fb0f76db6
branch: main
repository: go_webserver
topic: "GitHub Webhook Auto-Deploy Server Setup"
tags: [deployment, webhook, systemd, server-setup]
status: complete
last_updated: 2026-03-13
last_updated_by: CoreyCole
type: implementation_strategy
---

# Handoff: GitHub Webhook Auto-Deploy — Server Configuration

## Task(s)

1. **Webhook handler implementation** — COMPLETED. A GitHub webhook handler was added to go_webserver that listens for push events on `main`, pulls latest code, rebuilds, and restarts via SIGTERM + systemd.
2. **Server-side configuration** — PLANNED. The server needs the environment file created, the systemd service reloaded, and the GitHub webhook configured.

## Critical References

- `webserver/webhook/handler.go` — the webhook handler (adapted from creative-mode site pattern at `/Users/coreycole/cdev/creative-mode/site/internal/webhook/handler.go`)
- `go_webserver.service` — updated systemd unit with `EnvironmentFile` directive

## Recent changes

- `webserver/webhook/handler.go` — new file: webhook handler with HMAC-SHA256 verification, async rebuild (git fetch/reset, sqlc, templ, tailwind, go build), atomic binary replacement, SIGTERM for systemd restart
- `webserver/webserver.go:14` — added webhook import, wired `POST /webhook/github` route
- `main.go:13` — added `WebhookSecret` to Config struct, passed to `Start()`
- `go_webserver.service:11` — added `EnvironmentFile=/home/ubuntu/.config/go_webserver/env`
- `justfile:7` — fixed Tailwind `--content` flag to scan `./webserver/**/*.go` and `./webserver/**/*.templ` (was only scanning `./webserver/view/**/*`, missing classes in `webserver/lib/`)
- `public/build.css` — regenerated with correct content scan (removed stale unused classes)

## Learnings

- The rebuild process calls tools directly (`git`, `sqlc`, `templ`, `tailwindcss`, `go`) rather than using `just build`, to avoid snap/systemd scope issues (same pattern as creative-mode site).
- Binary replacement uses `os.Remove` (unlink) then `os.Rename` to avoid ETXTBSY errors on running executables.
- The `noctx` linter requires `exec.CommandContext` instead of `exec.Command` — a 5-minute timeout constant (`cmdTimeout`) was added.
- The `mnd` (magic number detector) linter flags inline numeric literals in function arguments.

## Artifacts

- `webserver/webhook/handler.go` — complete webhook handler
- `go_webserver.service` — updated systemd unit
- `main.go` — updated config
- `webserver/webserver.go` — updated route registration

## Action Items & Next Steps

These steps need to be performed **on the server** (the EC2 instance running coreycc.com):

1. **Pull latest code**:
   ```bash
   cd /home/ubuntu/go_webserver && git pull origin main
   ```

2. **Create environment file with webhook secret**:
   ```bash
   mkdir -p /home/ubuntu/.config/go_webserver
   WEBHOOK_SECRET=$(openssl rand -hex 32)
   echo "WEBHOOK_SECRET=$WEBHOOK_SECRET" > /home/ubuntu/.config/go_webserver/env
   echo "Save this secret for GitHub config: $WEBHOOK_SECRET"
   ```

3. **Rebuild the binary**:
   ```bash
   cd /home/ubuntu/go_webserver && just build
   ```
   Note: `sqlc`, `templ`, `tailwindcss`, `go`, and `pnpm` must be installed. Check `just install` if any are missing.

4. **Reload and restart the systemd service**:
   ```bash
   sudo cp go_webserver.service /etc/systemd/system/go_webserver.service
   sudo systemctl daemon-reload
   sudo systemctl restart go_webserver
   sudo systemctl status go_webserver
   ```

5. **Configure GitHub webhook** (GitHub.com > CoreyCole/go_webserver > Settings > Webhooks > Add webhook):
   - Payload URL: `https://coreycc.com/webhook/github`
   - Content type: `application/json`
   - Secret: the `WEBHOOK_SECRET` value from step 2
   - Events: "Just the push event"

6. **Verify**: Push a trivial change and watch logs:
   ```bash
   journalctl -u go_webserver -f
   ```

## Other Notes

- The Caddy reverse proxy (`Caddyfile`) already proxies `coreycc.com` to `localhost:3001` — no Caddy changes needed, the webhook endpoint is served by the Go app itself.
- If `WEBHOOK_SECRET` is empty, signature verification is skipped (not recommended for production).
- The handler uses a mutex (`buildMu.TryLock()`) to prevent overlapping rebuilds — if a second push arrives during a build, it's skipped.
- The creative-mode site's webhook handler (`/Users/coreycole/cdev/creative-mode/site/internal/webhook/handler.go`) was the reference implementation. Key difference: creative-mode filters by changed file paths (`site/` or `pkg/`), while go_webserver deploys on any push to main since it's a single-project repo.
