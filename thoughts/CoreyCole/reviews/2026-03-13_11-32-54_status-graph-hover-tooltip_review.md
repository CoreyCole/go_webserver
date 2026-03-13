---
date: 2026-03-13T11:32:54-0700
reviewer: Claude (Staff Eng Review)
git_commit: b730ad8d389a7ff55aab21987679b3ba5d72a2cf
branch: main
repository: go_webserver
plan_reviewed: thoughts/CoreyCole/plans/2026-03-13_10-21-04_status-graph-hover-tooltip.md
status: complete
type: plan_review
---

# Plan Review: Status Graph Hover Tooltip with Value Display and Dot Indicator

### Summary

Solid plan with a well-reasoned approach — embedding points as a Datastar signal and using a CSS div for the dot avoids the key pitfalls (SVG distortion, large payload, round-trip latency). The main issues are around mouseenter/mouseleave race conditions causing a stale tooltip/dot flash, ambiguous primary vs. fallback code in the plan, and duplicated coordinate computation that could drift.

### Critical Issues (Must Address Before Implementation)

1. **Stale tooltip and dot flash on line-to-line hover transitions**
   - Problem: When the mouse moves from one line to another, `mouseenter` fires on the new line before the first `mousemove`. At that instant `$hoveredLine` updates (making tooltip/dot visible), but `$dotLeft`/`$dotTop`/`$tooltipText` still hold **the previous line's values**. For one frame (~16ms), the dot appears at the old line's Y position with the old line's tooltip text.
   - Risk: Visual glitch — dot jumps to wrong line position momentarily. Tooltip shows "CPU: 12.3% at 14:35" when you just entered the Memory line.
   - Suggestion: Either (a) clear `$dotLeft`, `$dotTop`, `$tooltipText` in the `mouseleave` handler so there's nothing stale to show, or (b) don't show the dot/tooltip until the first `mousemove` fires — e.g., add a `$hoverActive` boolean signal set by mousemove, cleared by mouseleave, and use `data-show="$hoverActive"` instead of `data-show="$hoveredLine !== ''"` for the dot and tooltip.

2. **`mouseleave` handler doesn't clear `$tooltipText`**
   - Problem: The plan updates `mouseenter` to set `$hoverYIdx`/`$hoverVIdx` but leaves `mouseleave` as just `$hoveredLine = ''`. The `$tooltipText` signal retains the last value. If `data-show` ever evaluates before the signal is re-set (race, browser quirk), the tooltip shows stale text.
   - Risk: Stale tooltip content on next hover.
   - Suggestion: Update all `data-on:mouseleave` handlers to: `$hoveredLine = ''; $tooltipText = ''`

3. **Plan presents two conflicting tooltip implementations — which one?**
   - Problem: Step 6 shows an IIFE in `data-text` as the **primary** code block, then says "prefer this [signal] approach" for the fallback. An implementer reading top-down will use the IIFE. The plan should be unambiguous.
   - Risk: Implementer uses the wrong approach; the IIFE may have reactivity issues (Datastar's dependency tracking may not trigger re-evaluation when `$dotLeft` or `$hoverVIdx` change inside an IIFE, since the signal access happens inside a closure the compiler can't statically analyze).
   - Suggestion: Remove the IIFE approach entirely from the plan. Show only the `$tooltipText` signal approach. Add `$tooltipText` to the new signals list (it's currently missing from the signals declaration in step 2). Update the mousemove handler in step 5 to include the `$tooltipText` assignment. Update the tooltip HTML to `data-text="$tooltipText"`.

### Concerns (Should Address)

1. **Duplicated coordinate computation will drift**
   - Observation: The plan acknowledges that the points JSON loop duplicates the X/Y computation from the path-building loop (line 170), but dismisses refactoring as "optional." These two loops compute the same values from the same inputs — if one is changed (e.g., a new clamping rule, a different Y mapping), the other must change too. Two identical 20-line computations is a maintenance risk.
   - Suggestion: Compute coordinates once in a single loop and store them in a `[]pointData` slice. Use that slice for both path string building and points JSON generation. This eliminates the duplication with minimal added complexity.

2. **Dot initial position (0%, 0%) visible on first hover**
   - Observation: `$dotLeft` and `$dotTop` are initialized to `0`. On the very first hover, `mouseenter` fires and sets `$hoveredLine` (making the dot visible via `data-show`), but `$dotLeft`/`$dotTop` are still `0` — the dot will appear at top-left corner for one frame before `mousemove` fires and positions it correctly.
   - Suggestion: Initialize `$dotLeft` to `-10` (off-screen) so even if the dot becomes visible before mousemove fires, it's not in the viewport. Or gate visibility on a separate flag set by mousemove (see Critical Issue 1).

3. **`$tooltipText` signal is missing from the signals list**
   - Observation: The plan recommends using `$tooltipText` as the preferred approach (step 6 fallback), but it's not listed in the new signals declaration in step 2. The updated `data-signals` string on line 223 doesn't include `tooltipText`.
   - Suggestion: Add `tooltipText:''` to the outer `data-signals`.

4. **Points JSON inside `data-signals` attribute could be large for edge cases**
   - Observation: The plan caps at 360 points with ~15KB for 3-day range. But each point includes formatted strings with potential special characters. The entire JSON is embedded in an HTML attribute. While templ handles escaping correctly (and the browser decodes entities when reading attributes), very large attribute values can impact DOM parsing performance and make the page source hard to debug.
   - Suggestion: This is acceptable for 360 points but worth monitoring. If it becomes an issue, consider using `MarshalAndPatchSignals` from the server to update `graphPoints` via SSE signal patching instead of embedding in the DOM.

5. **Dot at container edges may be clipped or overflow**
   - Observation: The plan notes "half may overflow; acceptable" for the dot at `0%` or `100%`. But with `transform: translate(-50%, -50%)`, the dot at `left: 100%` would extend 5px past the container's right edge into the axis label area.
   - Suggestion: Add `overflow: hidden` to the `<div class="mx-10 h-full relative">` container to clip the dot cleanly at edges.

### Questions (Need Clarification)

1. Has `data-signals` on elements within SSE-patched inner HTML been verified to work with this specific Datastar version (1.0.0-rc-8)? The MutationObserver approach should pick it up, but this is a key assumption worth a quick manual test before committing to this architecture.
2. When the SVG graph gets SSE-patched every 30 seconds, the hit-area `<path>` elements are replaced. If the user is actively hovering, does Datastar properly fire `mouseleave` on the removed element and `mouseenter` on the new element? Or could the hover state get stuck (tooltip visible but no line hovered)?

### Suggestions (Nice to Have)

1. **Binary search for `findNearestPoint`**: The points are sorted by X. A binary search would be O(log n) vs O(n). At 360 points this barely matters (~microseconds either way), but it's a trivial improvement if anyone feels like it.
2. **Animate the dot**: A subtle `transition: left 50ms, top 50ms` on the dot div would smooth the snapping between discrete data points as the mouse moves along a line.
3. **Show memory bytes in tooltip for Memory line**: The current plan shows "Memory: 45.2% at 14:35". It could additionally show the bytes value (e.g., "Memory: 45.2% (7.2 GB) at 14:35") since `MemUsedBytes` is available in the snapshot. This would match the legend which already shows bytes.

### What's Good

- **Dot-as-CSS-div approach** is the right call — avoids the `preserveAspectRatio="none"` distortion problem cleanly. The coordinate mapping analysis (viewBox 0-100 → CSS %) is correct and the plan correctly identifies the need to add `position: relative` to the `mx-10 h-full` container.
- **Downsampling strategy** is pragmatic — 360 points is a sensible cap that keeps payload reasonable while maintaining tooltip precision.
- **`<script>` placement** outside the SSE-patched area is correct. Independent research confirms templ emits static script content verbatim with no escaping.
- **Compact array-of-arrays format** is a good trade-off between payload size and code readability. The index-based access via `$hoverYIdx`/`$hoverVIdx` signals is clever and keeps the mousemove handler generic across all three lines.
- **Edge cases section** covers the important scenarios (empty graph, single point, re-render during hover).
- **Tailwind class availability** was verified — `bg-primary`, `bg-green-500`, `bg-orange-500` are all already literal strings in the template, so the JIT scanner will include them.

### Recommended Next Steps

1. Resolve the three critical issues (stale flash, mouseleave cleanup, consolidate to one tooltip approach)
2. Add `tooltipText` to the signals declaration
3. Consider the single-loop refactoring for coordinate computation to prevent drift
4. Manually verify `data-signals` on SSE-patched elements works in Datastar rc-8 (quick sanity test before starting implementation)
5. Implement Phase 1, verify PointsJSON in browser devtools, then proceed to Phase 2
