---
date: 2026-03-12T22:52:08-07:00
researcher: CoreyCole
git_commit: 3443a3096d08a3d57f5e685bda529e86ac23e82e
branch: main
repository: go_webserver
topic: "Codebase Cleanup and Full Review"
tags: [cleanup, review, react-removal, s3-migration, documentation]
status: complete
last_updated: 2026-03-12
last_updated_by: CoreyCole
type: implementation_strategy
---

# Handoff: Codebase Cleanup and Full Review

## Task(s)

### Completed
1. **React removal** -- Deleted all React code (`react/` source, `public/react/` 1.1MB bundle, `esbuild.js`, `.eslintrc.js`, `prettier.config.js`). Stripped `package.json` down to only Tailwind CSS deps. The React code was a single unused `Dropdown` component never referenced from any template or Go code.

2. **Bevy WASM games migrated to S3** -- Created public S3 bucket `coreycole-games` in `us-west-2` with public read policy and CORS enabled. Uploaded both games (124MB total). Updated `wasm_game.go` to serve JS/WASM from `https://coreycole-games.s3.us-west-2.amazonaws.com/games/...` via a static `gameDirs` map instead of local filesystem lookups. Removed `public/games/` directory (124MB) and added it to `.gitignore`.

3. **Full codebase review** -- Comprehensive analysis completed (see Learnings below).

### Planned/Discussed
4. **Codebase cleanup** -- Fix issues found during review (see Action Items).
5. **Documentation** -- Write proper documentation for the project.

## Critical References
- `CLAUDE.md` -- Developer guide, architecture overview, build commands
- `thoughts/CoreyCole/handoffs/general/2026-03-12_21-46-17_homepage-redesign.md` -- Prior redesign handoff (Tailwind v4 migration, layout system, Datastar)

## Recent changes
- `webserver/handle/wasm_game.go` -- Replaced local filesystem game lookup with S3 URL construction via static `gameDirs` map. Removed `os`, `path/filepath` imports.
- `webserver/webserver.go:25-29` -- Cleaned up stale comments about local game asset serving.
- `package.json` -- Removed all React/eslint/prettier/typescript/esbuild deps, kept only `@tailwindcss/cli` and `tailwindcss`.
- `.gitignore:11` -- Replaced `public/react/**/*.js` with `public/games/`.
- Deleted: `react/index.ts`, `react/components.tsx`, `public/react/index.js`, `esbuild.js`, `.eslintrc.js`, `prettier.config.js`, `public/games/` (124MB).

## Learnings

### Bugs Found
- **Path traversal in markdown handler** (`handle/md.go:16`): `os.ReadFile("public/md/" + filename)` with unsanitized URL param allows reading arbitrary files (e.g. `/md/../../.env`).
- **Unclosed `<p>` tag** (`lib/resume_json_to_html.go:110`): Closing `<p>` instead of `</p>` produces invalid HTML for every skill entry.
- **Dead error check** (`handle/md.go:28-33`): Checks `err` from a previous call, not from `MarkdownBytesToHTML` which returns only a string.
- **Resume JSON field swap** (`lib/resume_json_to_html.go:24-25`): `Company` and `Position` struct fields are deliberately swapped vs their JSON tags "for pdf export" -- extremely confusing and fragile.

### Dead Code
- `calculateListDepth` function (`lib/markdown_to_html.go:190-200`) -- defined but never called.
- `style` constant `"monokai"` (`handle/welcome.go:15`) -- passed to renderer but immediately discarded; styles are hardcoded to `"github"` / `"github-dark"`.
- Dead `li.Attribute` modification (`lib/markdown_to_html.go:157-162`) -- attributes modified but never assigned back, then hardcoded HTML is returned anyway.

### Stale Config
- `.air.toml:16` -- `include_dir` still references non-existent `react/` directory.
- `.air.toml:17` -- `include_ext` includes `tsx`, `ts`, `jsx` which are no longer relevant.
- `Makefile:22-35` -- `up`, `reset`, `down`, `migration`, `seed` targets reference non-existent `cmd/` directory (leftover from when project had a database).
- `go.mod:3` -- unnecessary `replace` directive mapping module to itself.

### Architecture Notes
- Port `:3001` is hardcoded in `main.go:8` with no env var override.
- Blog posts under `public/md/blog/` are inaccessible via the `/md/:filename` renderer because Echo's `:param` doesn't match slashes.
- Markdown renderer uses `fmt.Println` for errors instead of zerolog.
- `e.Logger.Fatal(err)` after `e.Start()` makes `return err` unreachable in `webserver.go:33-34`.
- The `context/` directory contains two full repo clones (~large) used as reference during redesign; gitignored but still on disk.

## Artifacts
- `thoughts/CoreyCole/handoffs/general/2026-03-12_22-52-08_codebase-cleanup-and-review.md` (this document)

## Action Items & Next Steps

### Cleanup (fix issues found during review)
1. Fix path traversal vulnerability in `handle/md.go:16` -- sanitize filename, reject `..` segments
2. Fix unclosed `<p>` tag in `lib/resume_json_to_html.go:110`
3. Remove dead `calculateListDepth` function from `lib/markdown_to_html.go`
4. Remove dead `style` constant and fix `NewMarkdownToHtmlRenderer` to not take an unused param
5. Clean up dead `li.Attribute` modification code in `markdown_to_html.go`
6. Remove dead error check in `handle/md.go:28-33`
7. Clean up `.air.toml` -- remove `react` from `include_dir`, remove `tsx/ts/jsx` from `include_ext`
8. Remove stale database migration targets from `Makefile`
9. Remove unnecessary `replace` directive from `go.mod`
10. Consider making port configurable via env var
11. Fix blog post routing (support nested paths under `/md/`)
12. Replace `fmt.Println` error logging in markdown renderer with zerolog
13. Fix unreachable `return err` in `webserver.go`

### Documentation
14. Update `CLAUDE.md` to reflect current state (React removed, games on S3)
15. Write README with setup instructions, architecture overview, deployment info
16. Document S3 bucket setup and game deployment process

### Feature Work (from prior handoff)
17. Re-add projects section to homepage as structured templ components (lost during Tailwind v4 redesign)

## Other Notes
- S3 bucket: `coreycole-games` in `us-west-2`, account `975050104386`, user `corey`
- S3 has public read policy + CORS (AllowedOrigins: `*`, AllowedMethods: `GET`)
- Game URLs: `https://coreycole-games.s3.us-west-2.amazonaws.com/games/{gameDir}/...`
- Two games: `giga_platformer-7143ed686304a07e` (114MB) and `nessyclothes-3d0f9d8535e29267` (10MB)
- The `thoughts/` directory is untracked in git and synced via `just sync-thoughts`
- Node deps went from ~hundreds of packages down to 29 (tailwindcss only)
