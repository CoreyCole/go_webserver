---
date: 2026-03-12T19:46:19-07:00
researcher: CoreyCole
git_commit: 3443a3096d08a3d57f5e685bda529e86ac23e82e
branch: main
repository: go_webserver
topic: "Homepage redesign using creative-mode and datastarui as reference"
tags: [research, codebase, homepage, redesign, datastar, templ, tailwind]
status: complete
last_updated: 2026-03-12
last_updated_by: CoreyCole
---

# Research: Homepage Redesign

**Date**: 2026-03-12T19:46:19-07:00
**Researcher**: CoreyCole
**Git Commit**: 3443a3096d08a3d57f5e685bda529e86ac23e82e
**Branch**: main
**Repository**: go_webserver

## Research Question

Redesign the go_webserver homepage (`webserver/handle/welcome.go`) using patterns from the creative-mode site and datastarui projects as reference.

## Summary

The current homepage is a minimal markdown+resume renderer with a bare layout (jQuery, htmx, Font Awesome CDN, no design tokens, no theme toggle). Both reference projects use a significantly more modern stack: **templ slot-based composition**, **shadcn/ui design tokens** (HSL CSS variables bridged to Tailwind via `@theme`), **Datastar** for client-side reactivity, and **cache-busted CSS loading**. The redesign should adopt these patterns to create a polished personal landing page.

## Current Implementation

### Handler (`webserver/handle/welcome.go`)
- Reads `public/md/welcome.md` -> renders markdown to HTML (with syntax highlighting via chroma)
- Reads `public/resume.json` -> renders resume to HTML (hardcoded tailwind classes in Go strings)
- Concatenates both HTML strings -> wraps in `lib.HTMLToComponent` -> passes to `vi.WelcomePage()`
- The page is entirely server-rendered with no client-side interactivity

### Layout (`webserver/view/layout/layout.templ`)
- Hardcoded dark mode (`class="dark"` on `<html>`)
- Loads jQuery 3.7.1, htmx 1.9.9, Font Awesome 6.5.1 from CDNs
- No design token system -- uses literal colors like `bg-black text-slate-300`
- No theme toggle, no responsive header, no footer

### Page Template (`webserver/view/components.templ`)
- `WelcomePage` wraps content in `<div class="p-4 max-w-5xl mx-auto">`
- Content is a single `templ.Component` blob (markdown + resume concatenated)
- Has commented-out React dropdown code

### CSS (`webserver/view/css/index.css`)
- Tailwind v3 directives (`@tailwind base/components/utilities`)
- Custom `.myh` heading classes with hardcoded green color `#047857`
- Custom bullet style for `ul`

### Resume Renderer (`webserver/lib/resume_json_to_html.go`)
- Builds HTML strings in Go with inline Tailwind classes
- No semantic color tokens -- uses literal classes like `text-xl`, `font-bold`
- Note: `position` and `company` JSON fields are swapped intentionally (for PDF export compatibility)

## Reference Project: creative-mode/site

### Architecture
- Go/Echo + templ + Tailwind CSS v4 + Datastar (no React, no htmx, no jQuery)
- Slot-based layout composition: `Root` -> `Head` + `Header` + `{children}` + `Footer`
- `RootArgs` struct controls layout behavior (hide footer, hide CTA, fixed viewport)

### Key Patterns
- **Design tokens**: shadcn/ui HSL variables in `:root`/`.dark`, bridged via `@theme`
- **Cache-busted CSS**: `filepath.Glob("static/css/out.*.css")` at render time
- **Flash-free theme init**: synchronous `initTheme()` script before body renders
- **Datastar reactivity**: `data-signals` for state, `data-show` for visibility, `data-on:*` for events
- **Responsive header**: sticky with backdrop blur, mobile-hidden nav items shown in footer
- **Homepage structure**: Hero section + Features grid (3-col) + CTA section + Discord OAuth modal

### Key Files
- `creative-mode/site/layouts/root.templ` -- document skeleton with slot composition
- `creative-mode/site/layouts/head.templ` -- cache-busted CSS, theme init, Datastar CDN
- `creative-mode/site/layouts/header.templ` -- sticky header with theme toggle
- `creative-mode/site/pages/home.templ` -- hero + features grid + CTA
- `creative-mode/site/static/css/theme.css` -- shadcn/ui design tokens

## Reference Project: datastarui

### Architecture
- Go/Echo + templ + Tailwind CSS v4 + Datastar
- Same slot-based `Root` layout pattern
- `RootArgs` with `CurrentPage` + `CurrentPath` for nav state
- Conditional layout: home gets full-width (no sidebar), other pages get sidebar

### Key Patterns
- **Same design token system** as creative-mode (shadcn/ui HSL variables)
- **Same cache-busted CSS** approach
- **Same theme initialization** pattern
- **Signal namespacing**: `SignalManager` utility creates namespaced Datastar state
- **Homepage structure**: Hero with gradient text + Features grid (3-col responsive)
- **Sidebar navigation**: shared data source (`GetSidebarSections()`) powers both sidebar and component listing
- **Animation system**: full `tailwindcss-animate` implementation in CSS

### Key Files
- `datastarui/layouts/root.templ` -- root layout with conditional sidebar
- `datastarui/pages/home.templ` -- hero + features grid
- `datastarui/static/css/index.css` -- design tokens + animations
- `datastarui/utils/signals.go` -- Datastar signal management

## Redesign Recommendations

### 1. Upgrade Stack
- **Drop**: jQuery, htmx, Font Awesome CDNs
- **Add**: Datastar (single CDN import as ES module)
- **Upgrade**: Tailwind v3 -> v4 (`@import "tailwindcss"` replaces `@tailwind` directives)

### 2. Adopt Design Token System
- Add `theme.css` with shadcn/ui HSL variables (`:root` + `.dark`)
- Bridge to Tailwind via `@theme` block
- Replace hardcoded colors (`.myh` green `#047857`, `bg-black`, `text-slate-300`) with semantic tokens (`text-foreground`, `bg-background`, `text-primary`)

### 3. Improve Layout
- Add `RootArgs` struct for layout configuration
- Add sticky header with nav links and Datastar-powered theme toggle
- Add footer with mobile-responsive nav duplication
- Add flash-free `initTheme()` script
- Add cache-busted CSS loading via `filepath.Glob`

### 4. Restructure Homepage Content
Instead of a single markdown+resume HTML blob:
- **Hero section**: Name, title, social links (GitHub, X, LinkedIn)
- **Projects section**: Cards grid (creative-mode, WASM game, etc.)
- **Resume section**: Keep the JSON-to-HTML renderer but use semantic color tokens
- Consider making resume a separate page/route

### 5. File Structure Changes
```
webserver/
  view/
    layout/
      layout.templ      -> root.templ (rename, add RootArgs)
      head.templ         (new: cache-busted CSS, theme init, Datastar)
      header.templ       (new: sticky header with theme toggle)
      footer.templ       (new: footer with mobile nav)
      args.go            (new: RootArgs struct)
    pages/               (new directory, replaces components.templ)
      home.templ         (hero + projects + resume)
    css/
      index.css          (update to Tailwind v4)
      theme.css          (new: shadcn/ui design tokens)
```

### 6. Context Directory
Both reference projects are already cloned inside `go_webserver/context/`:
- `context/creative-mode/`
- `context/datastarui/`

This makes them available for direct reference during implementation.

## Open Questions
- Should the resume remain on the homepage or become a separate `/resume` route?
- Which Datastar version to pin? creative-mode uses `v1.0.0-RC.7`
- Should we add a sidebar for future pages, or keep the single-page layout?
- Do we want the full `tailwindcss-animate` system from datastarui, or just the basics?
