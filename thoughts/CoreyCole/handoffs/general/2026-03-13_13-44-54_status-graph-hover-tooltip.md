---
date: 2026-03-13T13:44:54-0700
researcher: CoreyCole
git_commit: b730ad8d389a7ff55aab21987679b3ba5d72a2cf
branch: main
repository: go_webserver
topic: "Status Graph Hover Tooltip Enhancement"
tags: [implementation, status-page, svg-graph, datastar, tooltip]
status: complete
last_updated: 2026-03-13
last_updated_by: CoreyCole
type: implementation_strategy
---

# Handoff: Status Graph Hover Tooltip with Value Display and Dot Indicator

## Task(s)

- **Implementation plan creation** — COMPLETED. Created and reviewed a plan to enhance the status page SVG graph tooltip to show point values (time + metric value) and a colored dot on the hovered line.
- **Plan review** — COMPLETED. Staff eng review identified and resolved: stale flash on hover transitions, duplicated coordinate computation, missing signals, ambiguous tooltip approach.
- **Implementation** — PLANNED. Both phases are ready to implement. Start with Phase 1 (server-side), then Phase 2 (template).

## Critical References

- **Implementation plan (reviewed & updated)**: `thoughts/CoreyCole/plans/2026-03-13_10-21-04_status-graph-hover-tooltip.md` — READ THIS FIRST, it contains all implementation details with code snippets
- **Review document**: `thoughts/CoreyCole/reviews/2026-03-13_11-32-54_status-graph-hover-tooltip_review.md`

## Recent changes

No code changes yet — only plan and review documents were created.

## Learnings

### Datastar signal behavior on SSE-patched elements
- `data-signals` (without `__ifmissing`) on elements inside SSE-patched DOM **overwrites** existing signal values. The MutationObserver picks up new `data-signals` attributes after `PatchElementTempl` with `WithModeInner()`.
- This means putting `data-signals="{graphPoints:[...]}"` inside MetricsGraph will update `$graphPoints` on each 30-second SSE graph re-render.

### SVG coordinate mapping for the dot
- The SVG uses `preserveAspectRatio="none"` with viewBox `0 0 100 100`, which makes SVG `<circle>` elements distort into ellipses.
- Solution: use an absolutely-positioned `<div>` outside the SVG with `left`/`top` as percentages. ViewBox 0-100 maps directly to 0%-100% CSS positioning.
- **Critical**: Must add `position: relative` to `<div class="mx-10 h-full">` (`status.templ:157`) for absolute positioning to work correctly. Without it, the nearest positioned ancestor is the wider `relative h-48` parent which includes axis label areas.

### Stale hover flash prevention
- When mouse moves from line A to line B, `mouseenter` on B fires before `mousemove`. If tooltip/dot visibility gates on `$hoveredLine !== ''`, they flash with stale position/text from line A.
- Solution: Gate visibility on a separate `$hoverActive` boolean set only by `mousemove`, cleared by `mouseleave`.

### templ `<script>` tags
- Static JS inside `<script>` tags in templ files is emitted verbatim (no HTML escaping). Safe for the `findNearestPoint` helper function.
- Place the script in `StatusPage` (not inside `MetricsGraph`) so it survives SSE patches.

## Artifacts

- `thoughts/CoreyCole/plans/2026-03-13_10-21-04_status-graph-hover-tooltip.md` — Implementation plan (2 phases, reviewed & updated)
- `thoughts/CoreyCole/reviews/2026-03-13_11-32-54_status-graph-hover-tooltip_review.md` — Staff eng review

## Action Items & Next Steps

### Phase 1: Server-Side (implement first)
1. Add `PointsJSON string` field to `GraphData` struct in `webserver/view/status.go`
2. Refactor `buildGraph` in `webserver/handle/status.go:274-371` — replace the path-building loop with a single loop using a `pointCoords` struct slice. Use it for both SVG path strings AND sampled points JSON (max 360 points, evenly sampled).
3. Verify build succeeds (`just build`) and graph still renders correctly.

### Phase 2: Template (implement second)
1. Add `<script>` with `findNearestPoint` function in `StatusPage` template (after `<h1>`, before `#metrics-graph` div)
2. Add new signals to outer `data-signals`: `graphPoints:[]`, `hoverYIdx:1`, `hoverVIdx:5`, `dotLeft:-10`, `dotTop:-10`, `tooltipText:''`, `hoverActive:false`
3. Add hidden `<div data-signals="{graphPoints:...}">` inside `MetricsGraph` (in else branch, before chart)
4. Update `mouseenter` handlers to set `$hoveredLine`, `$hoverYIdx`, `$hoverVIdx`
5. Update `mouseleave` handlers to clear `$hoveredLine`, `$hoverActive`, `$tooltipText`
6. Update `mousemove` handlers to find nearest point, set `$hoverActive`, `$dotLeft`, `$dotTop`, `$tooltipText`
7. Update tooltip to use `data-show="$hoverActive"` and `data-text="$tooltipText"`
8. Add dot div (absolutely positioned, `overflow-hidden` container, `data-show="$hoverActive"`)
9. Run `just check` to verify everything passes.

## Other Notes

### Key files to modify
- `webserver/view/status.go` — GraphData struct (add PointsJSON field)
- `webserver/handle/status.go` — buildGraph function (refactor loop, add points JSON generation)
- `webserver/view/status.templ` — Template (signals, script, tooltip, dot, event handlers)

### Point data format
Array of arrays: `[x, cpuY, memY, viewsY, "time", "cpuVal", "memVal", "viewsVal"]`
- Indices 1/2/3 are Y coords (selected by `$hoverYIdx`), indices 5/6/7 are display values (selected by `$hoverVIdx`)

### Build commands
- `just build` — templ generate + tailwind + go build
- `just watch` — air hot-reload for development
- `just check` — fmt + generate + lint + build (full validation)
- Tailwind scans `./webserver/**/*.go` and `./webserver/**/*.templ` — classes in `data-class` attributes work because `bg-primary`, `bg-green-500`, `bg-orange-500` are already literal strings in the legend
