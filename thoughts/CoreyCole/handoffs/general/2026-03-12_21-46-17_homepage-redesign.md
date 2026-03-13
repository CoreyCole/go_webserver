---
date: 2026-03-12T21:46:17-07:00
researcher: CoreyCole
git_commit: 3443a3096d08a3d57f5e685bda529e86ac23e82e
branch: main
repository: go_webserver
topic: "Homepage Redesign — Tailwind v4, Design Tokens, Datastar Header"
tags: [implementation, tailwind, datastar, homepage, layout, templ]
status: complete
last_updated: 2026-03-12
last_updated_by: CoreyCole
type: implementation_strategy
---

# Handoff: Homepage Redesign

## Task(s)

Modernize the go_webserver homepage using patterns from creative-mode/site and datastarui reference projects.

**Completed:**
- Step 1: Added shadcn/ui HSL design tokens (`:root` light / `.dark` dark) to `webserver/view/css/index.css`
- Step 2: Restructured layout — created `Root`/`Head`/`Header`/`Footer` templ components with Datastar theme toggle and links dropdown
- Step 3: Created `HomePage` template wrapping resume JSON content in the new Root layout
- Step 4: Updated markdown renderer link color to `text-primary`
- Upgraded Tailwind v3.4.1 → v4.2.1 with `@tailwindcss/cli`, removed daisyUI
- Added `@theme` block in CSS to bridge CSS variables to Tailwind v4 utilities
- Fixed Makefile: templ generate runs before tailwind, uses `--content` CLI flags
- Switched Datastar from CDN to local `/js/datastar-1-0-0-rc-8-1df5dc253a746506.js`

**Work in Progress / Known Issue:**
- The **projects section** from `welcome.md` was lost during the refactor. The old flow rendered `welcome.md` (which contained project links) + resume JSON. The user removed the markdown rendering from `welcome.go` (see modified handler below) and now only resume JSON is rendered. Projects need to be re-added — either as a templ section in `home.templ` or by re-adding welcome.md rendering.

## Critical References
- `context/creative-mode/site/layouts/header.templ` — Datastar theme toggle pattern (colon syntax: `data-on:click`)
- `context/creative-mode/site/static/css/theme.css` — `@theme` + `@layer base` design token pattern
- `context/datastarui/components/dropdown/dropdown.templ` — Dropdown component pattern (signals, click-outside, escape)

## Recent changes

- `webserver/view/css/index.css` — Replaced `@tailwind` directives with `@import "tailwindcss"` + `@theme` block + `@layer base` design tokens. Primary color: `162 94% 24%` (emerald) in both modes.
- `tailwind.config.js` — Deleted (Tailwind v4 uses CSS-based config via `@theme`)
- `package.json` — Removed daisyui, react, and related deps. Now only `@tailwindcss/cli` + `tailwindcss` v4.2.1.
- `Makefile:1-4` — Reordered: templ generate first, then `pnpm exec tailwindcss` with `--content "./webserver/view/**/*"` flag
- `webserver/view/layout/args.go` — Created: `RootArgs{Title string}`
- `webserver/view/layout/head.templ` — Created: `<head>` with `initTheme()` script + local Datastar JS
- `webserver/view/layout/header.templ` — Created: Sticky header with "Corey Cole Projects & Experience" left, dropdown (🔗 Links with Resume PDF/GitHub/X/LinkedIn) + theme toggle right. Uses Datastar colon syntax.
- `webserver/view/layout/footer.templ` — Created: Footer with GitHub/X/LinkedIn links
- `webserver/view/layout/layout.templ` — Rewritten: `Root(RootArgs)` composing Head+Header+main+Footer. `Default(title)` wraps Root for backward compat.
- `webserver/view/home.templ` — Created: Wraps content in Root layout with `max-w-5xl mx-auto p-4`
- `webserver/view/components.templ` — Removed `WelcomePage` and dead React dropdown code. `MarkdownPage` now uses `bg-background`.
- `webserver/handle/welcome.go` — Modified by user: removed markdown rendering of `welcome.md`, only renders resume JSON via `ResumeJSONToHTML`
- `webserver/lib/markdown_to_html.go:116` — Link color changed from `text-green-600 dark:text-green-400` to `text-primary`

## Learnings

1. **Tailwind v4 does NOT read `tailwind.config.js` by default.** You must use `@theme` directive in CSS or `@config` directive to import it. The color bridge (`text-primary`, `bg-background`, etc.) only works with `@theme`.
2. **Tailwind v4 content scanning uses `--content` CLI flags**, not config file `content` array. Both creative-mode and datastarui use this pattern.
3. **Datastar attribute syntax is colon-based**: `data-on:click`, NOT `data-on-click`. The harness CLAUDE.md explicitly warns: "dashes break the plugin lookup because HTML's dataset API converts `data-bind-foo` → `bindFoo` via camelCase, mangling the plugin name." See `context/creative-mode/harness/CLAUDE.md:294-296`.
4. **Datastar dropdown pattern** (from datastarui): `data-on:click__outside` goes on the ROOT wrapper div, `data-show` + `data-on:keydown__window` go on the content div.
5. **Air hot-reload** only watches `include_dir: ["react", "webserver"]` — root-level file changes (Makefile, package.json, tailwind.config.js) don't trigger rebuilds. Must restart air manually for those.
6. **`initTheme()` in `<head>`** prevents FOUC — reads localStorage/prefers-color-scheme and toggles `dark` class before paint. Returns theme string for Datastar `data-signals="{theme: initTheme()}"` on `<body>`.

## Artifacts

- `webserver/view/css/index.css` — Design tokens + `@theme` block
- `webserver/view/layout/args.go` — RootArgs struct
- `webserver/view/layout/head.templ` — Head with initTheme + Datastar
- `webserver/view/layout/header.templ` — Sticky header + dropdown + theme toggle
- `webserver/view/layout/footer.templ` — Footer
- `webserver/view/layout/layout.templ` — Root + Default layout
- `webserver/view/home.templ` — Homepage template
- `webserver/view/components.templ` — Cleaned up (MarkdownPage + BevyPage only)
- `webserver/handle/welcome.go` — Simplified handler
- `Makefile` — Updated build pipeline
- `package.json` — Cleaned deps
- Implementation plan: `/Users/coreycole/.claude/plans/harmonic-strolling-church.md`

## Action Items & Next Steps

1. **Re-add projects section to homepage.** The `welcome.md` content included project links (creative-mode, WASM game, markdown renderer). These need to be added back — either as a templ section in `webserver/view/home.templ` (preferred, structured cards) or by re-rendering `welcome.md` through the markdown renderer. The user's current `welcome.go` only renders resume JSON.
2. **Verify all pages work end-to-end:**
   - `GET /` — homepage with header, theme toggle, dropdown, projects, resume
   - `GET /md/test.md` — markdown viewer still works with new layout
   - `GET /games/giga_platformer-7143ed686304a07e/game` — WASM game still loads (has its own `<!DOCTYPE html>`)
   - Theme toggle switches light/dark on all pages
3. **Consider committing.** All changes are unstaged — this is a large changeset.

## Other Notes

- The `public/md/welcome.md` file still exists and is served at `/md/welcome.md` — it just isn't rendered on the homepage anymore.
- `BevyPage` is completely unaffected — it has its own `<!DOCTYPE html>` and doesn't use the Root layout.
- The `public/js/datastar-1-0-0-rc-8-1df5dc253a746506.js` file was already present in the repo before this work.
- The `react/` directory and esbuild step still exist in the Makefile but React deps were removed from package.json by the user. The esbuild step may fail or be a no-op now.
