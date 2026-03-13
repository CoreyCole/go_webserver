# Status Graph Hover Tooltip with Value Display and Dot Indicator

## Overview

Enhance the status page SVG graph tooltip to show the actual data point value and time when hovering a line, and display a colored dot on the line at the nearest data point.

## Current State Analysis

- The graph renders three SVG `<path>` elements (CPU, Memory, Views) with transparent 12px-wide hit-area paths for mouse interaction
- Hovering a line shows a tooltip with just the line name (e.g., "CPU")
- Mouse position is tracked via `$tooltipX`/`$tooltipY` Datastar signals (clientX/clientY)
- All path data is server-rendered; no structured point data is available client-side
- The SVG uses `preserveAspectRatio="none"` which distorts `<circle>` elements

### Key Discoveries:
- SVG viewBox is `0 0 100 100`, mapping X to time (0-100%) and Y to value (inverted: `100 - value`) — `status.go:337-364`
- Graph is replaced entirely via SSE element patch (`datastar.WithModeInner()`) — `status.go:261-265`
- Points data is computed during `buildGraph` but discarded after path string generation — `status.go:274-371`
- Hit-area paths have `pointer-events="stroke"` and `cursor: pointer` — `status.templ:164-175`
- CPU/Memory Y = `100 - percent`, Views Y = normalized to viewsMin/viewsMax range — `status.go:345-356`

## Desired End State

When hovering a line on the graph:
1. Tooltip shows: **"CPU: 12.3% at 14:35"** (line name, value, and timestamp)
2. A colored dot appears on the line at the nearest data point position
3. The dot color matches the line color (primary for CPU, green for Memory, orange for Views)
4. Performance is smooth (~60fps) even on the 3-day range via downsampling to max 360 points

### Verification:
- Hover each line (CPU, Memory, Views) and confirm tooltip shows correct name, value, and time
- Confirm dot appears on the hovered line at the correct position
- Confirm dot color matches line color
- Switch between time ranges (1h, 6h, 24h, 3d) and confirm tooltip still works
- Confirm tooltip and dot disappear when mouse leaves a line
- Confirm no visual distortion on the dot (it should be a circle, not an ellipse)

## What We're NOT Doing

- Vertical crosshair showing all three lines at once (different UX pattern)
- Click-to-pin tooltip behavior
- Tooltip on mobile/touch (current hit areas don't support touch)
- Custom tooltip styling beyond the existing design

## Implementation Approach

Embed a compact JSON array of sampled data points as a Datastar signal inside the graph template. On mousemove, convert the cursor's clientX to SVG viewBox X coordinates using `getBoundingClientRect`, then find the nearest point by X. Use the point's data to populate the tooltip and position the dot.

The dot is an absolutely-positioned `<div>` outside the SVG (to avoid `preserveAspectRatio="none"` distortion). Since the SVG viewBox is 0-100 and fills its container, viewBox coordinates map directly to CSS percentages.

## Phase 1: Server-Side — Add Points Data to GraphData

### Overview
Compute a compact, downsampled JSON array of data points during `buildGraph` and include it in `GraphData`. Refactor the path-building loop to compute coordinates once and reuse them for both path strings and the points JSON.

### Changes Required:

#### 1. Add PointsJSON to GraphData
**File**: `webserver/view/status.go`
**Changes**: Add `PointsJSON` field

```go
type GraphData struct {
	CPUPath      string
	MemPath      string
	ViewsPath    string
	CPULast      string
	MemLast      string
	MemBytesLast string
	ViewsLast    string
	ViewsAxisMax string
	ViewsAxisMid string
	ViewsAxisMin string
	PointCount   int
	TimeLabel    string
	TimeStart    string
	TimeEnd      string
	PointsJSON   string // JSON array for client-side tooltip lookup
}
```

#### 2. Refactor buildGraph to compute coordinates once, generate paths and points JSON
**File**: `webserver/handle/status.go`
**Changes**: Replace the path-building loop with a single loop that stores computed coordinates in a struct slice. Use that slice for both SVG path generation and sampled points JSON output.

Point format (array of arrays for compactness):
```
// [x, cpuY, memY, viewsY, "time", "cpuVal", "memVal", "viewsVal"]
[[0.0, 87.7, 54.8, 23.1, "14:35", "12.3%", "45.2%", "1234"], ...]
```

Indices:
- 0: x (SVG X coordinate, 0-100)
- 1: cpuY (SVG Y coordinate, 0-100)
- 2: memY (SVG Y coordinate, 0-100)
- 3: viewsY (SVG Y coordinate, 0-100)
- 4: time string (e.g., "14:35")
- 5: CPU value string (e.g., "12.3%")
- 6: Memory value string (e.g., "45.2%")
- 7: Views value string (e.g., "1,234")

Sampling logic:
- Max 360 points
- Compute `step = ceil(n / 360)`
- Take every `step`-th point, always including the last point

Implementation — replace the existing path-building loop in `buildGraph`:

```go
// Compute coordinates for each snapshot once.
type pointCoords struct {
	x, cpuY, memY, viewsY float64
	timeStr                string
	cpuVal, memVal         string
	viewsVal               string
}

points := make([]pointCoords, n)
for i, s := range snaps {
	x := 0.0
	if totalSeconds > 0 {
		x = s.CreatedAt.Sub(cutoff).Seconds() / totalSeconds * percentScale
	}
	x = clamp(x)
	cpuY := clamp(percentScale - s.CPUPercent)
	memY := clamp(percentScale - s.MemUsedPercent)

	var viewsY float64
	if viewsRange > 0 {
		viewsY = percentScale - float64(s.TotalVisits-viewsMin)/float64(viewsRange)*percentScale
	} else {
		viewsY = percentMid
	}

	points[i] = pointCoords{
		x: x, cpuY: cpuY, memY: memY, viewsY: viewsY,
		timeStr:  s.CreatedAt.Format("15:04"),
		cpuVal:   fmt.Sprintf("%.1f%%", s.CPUPercent),
		memVal:   fmt.Sprintf("%.1f%%", s.MemUsedPercent),
		viewsVal: strconv.FormatInt(s.TotalVisits, 10),
	}
}

// Build SVG path strings from computed coordinates.
var cpuBuf, memBuf, viewsBuf strings.Builder
for i, p := range points {
	cmd := "L"
	if i == 0 {
		cmd = "M"
	}
	fmt.Fprintf(&cpuBuf, "%s%.1f,%.1f ", cmd, p.x, p.cpuY)
	fmt.Fprintf(&memBuf, "%s%.1f,%.1f ", cmd, p.x, p.memY)
	fmt.Fprintf(&viewsBuf, "%s%.1f,%.1f ", cmd, p.x, p.viewsY)
}
data.CPUPath = cpuBuf.String()
data.MemPath = memBuf.String()
data.ViewsPath = viewsBuf.String()

// Build sampled points JSON for client-side tooltip lookup.
const maxTooltipPoints = 360
step := 1
if n > maxTooltipPoints {
	step = (n + maxTooltipPoints - 1) / maxTooltipPoints // ceil division
}

var pointsBuf strings.Builder
pointsBuf.WriteByte('[')
first := true
for i := 0; i < n; i += step {
	p := points[i]
	if !first {
		pointsBuf.WriteByte(',')
	}
	first = false
	fmt.Fprintf(&pointsBuf, "[%.1f,%.1f,%.1f,%.1f,\"%s\",\"%s\",\"%s\",\"%s\"]",
		p.x, p.cpuY, p.memY, p.viewsY,
		p.timeStr, p.cpuVal, p.memVal, p.viewsVal,
	)
}
// Always include last point if not already included.
if (n-1)%step != 0 {
	p := points[n-1]
	fmt.Fprintf(&pointsBuf, ",[%.1f,%.1f,%.1f,%.1f,\"%s\",\"%s\",\"%s\",\"%s\"]",
		p.x, p.cpuY, p.memY, p.viewsY,
		p.timeStr, p.cpuVal, p.memVal, p.viewsVal,
	)
}
pointsBuf.WriteByte(']')
data.PointsJSON = pointsBuf.String()
```

This single-loop approach computes coordinates once in `pointCoords` and reuses them for both SVG paths and tooltip JSON, eliminating the risk of the two drifting out of sync.

### Success Criteria:

#### Automated Verification:
- [ ] Build succeeds: `just build`
- [ ] `GraphData.PointsJSON` is populated with valid JSON for all time ranges

#### Manual Verification:
- [ ] No visible change to the graph (this phase is data-only)

---

## Phase 2: Template — Add Helper Function, Dot, and Enhanced Tooltip

### Overview
Add a `findNearestPoint` JavaScript helper, new Datastar signals, the hover dot element, and update mousemove handlers to perform point lookups.

### Changes Required:

#### 1. Add helper script to StatusPage
**File**: `webserver/view/status.templ`
**Changes**: Add a `<script>` tag inside `StatusPage`, outside the `#metrics-graph` div (so it executes on page load and survives SSE patches).

Place after the `<h1>` and before the metrics graph div:

```html
<script>
function findNearestPoint(pts, xPct) {
  if (!pts || !pts.length) return null;
  var best = 0, bestD = Math.abs(xPct - pts[0][0]);
  for (var i = 1; i < pts.length; i++) {
    var d = Math.abs(xPct - pts[i][0]);
    if (d < bestD) { bestD = d; best = i; }
  }
  return pts[best];
}
</script>
```

#### 2. Add new signals
**File**: `webserver/view/status.templ`
**Changes**: Add signals to the outer `data-signals` attribute on line 14.

New signals:
- `graphPoints:[]` — points data array, updated by inner `data-signals` on graph re-render
- `hoverYIdx:1` — index into point array for the hovered line's Y coordinate (1=CPU, 2=Mem, 3=Views)
- `hoverVIdx:5` — index into point array for the hovered line's value string (5=CPU, 6=Mem, 7=Views)
- `dotLeft:-10` — dot X position as percentage of chart width (-10 = off-screen by default to prevent flash at 0,0 on first hover)
- `dotTop:-10` — dot Y position as percentage of chart height
- `tooltipText:''` — formatted tooltip string, set by mousemove handler
- `hoverActive:false` — true only after first mousemove fires; prevents stale dot/tooltip flash on mouseenter

Updated signals string:
```
data-signals="{memTotal:'',memUsed:'',memUsedPercent:'',cpuPercent:'',uptime:'',diskTotal:'',diskUsed:'',diskUsedPercent:'',graphRange:'1h',hoveredLine:'',tooltipX:0,tooltipY:0,graphPoints:[],hoverYIdx:1,hoverVIdx:5,dotLeft:-10,dotTop:-10,tooltipText:'',hoverActive:false}"
```

#### 3. Embed points data signal inside MetricsGraph
**File**: `webserver/view/status.templ`
**Changes**: Add a hidden div inside `MetricsGraph` (inside the `else` branch, before the chart) that updates the `graphPoints` signal when the graph is patched.

```html
<div style="display:none" data-signals={ fmt.Sprintf("{graphPoints:%s}", data.PointsJSON) }></div>
```

This ensures `$graphPoints` is updated each time the graph is re-rendered via SSE.

#### 4. Update mouseenter handlers to set index signals
**File**: `webserver/view/status.templ`
**Changes**: Update each hit-area path's `data-on:mouseenter` to set `$hoveredLine`, `$hoverYIdx`, and `$hoverVIdx`. Note: `mouseenter` does NOT set `$hoverActive` — that is deferred to the first `mousemove` to prevent a stale dot/tooltip flash.

CPU hit area (line 172):
```
data-on:mouseenter="$hoveredLine = 'CPU'; $hoverYIdx = 1; $hoverVIdx = 5"
```

Memory hit area (line 194):
```
data-on:mouseenter="$hoveredLine = 'Memory'; $hoverYIdx = 2; $hoverVIdx = 6"
```

Views hit area (line 216):
```
data-on:mouseenter="$hoveredLine = 'Views'; $hoverYIdx = 3; $hoverVIdx = 7"
```

#### 5. Update mouseleave handlers to clear all hover state
**File**: `webserver/view/status.templ`
**Changes**: Update all three hit-area `data-on:mouseleave` handlers to clear `$hoveredLine`, `$hoverActive`, and `$tooltipText`. This prevents stale values from persisting into the next hover.

Same handler for all three hit areas:
```
data-on:mouseleave="$hoveredLine = ''; $hoverActive = false; $tooltipText = ''"
```

#### 6. Update mousemove handlers to find nearest point
**File**: `webserver/view/status.templ`
**Changes**: Replace all three hit-area `data-on:mousemove` handlers with a version that performs the lookup, sets `$hoverActive`, computes `$tooltipText`, and positions the dot.

Same handler for all three hit areas:
```
data-on:mousemove__throttle.16ms="
  var svg = evt.currentTarget.closest('svg');
  var rect = svg.getBoundingClientRect();
  var xPct = (evt.clientX - rect.left) / rect.width * 100;
  var pt = findNearestPoint($graphPoints, xPct);
  $tooltipX = evt.clientX;
  $tooltipY = evt.clientY;
  $hoverActive = true;
  if (pt) {
    $dotLeft = pt[0];
    $dotTop = pt[$hoverYIdx];
    $tooltipText = $hoveredLine + ': ' + pt[$hoverVIdx] + ' at ' + pt[4];
  }
"
```

Key: `$hoverActive` is set here (not in mouseenter), ensuring the dot and tooltip only become visible after the first mousemove computes the correct position.

#### 7. Update tooltip content
**File**: `webserver/view/status.templ`
**Changes**: Replace the tooltip inner content (line 134-136) to use `$tooltipText` signal.

Update the tooltip `data-show` to gate on `$hoverActive` (not just `$hoveredLine`):
```html
<div
  style="display: none; position: fixed; z-index: 50; pointer-events: none;"
  data-show="$hoverActive"
  data-style:left="$tooltipX + 'px'"
  data-style:top="($tooltipY - 40) + 'px'"
>
  <div style="transform: translateX(-50%); white-space: nowrap;" class="rounded-md border border-border bg-popover px-3 py-1.5 text-sm text-popover-foreground shadow-md">
    <span data-text="$tooltipText"></span>
  </div>
</div>
```

Using `$hoverActive` instead of `$hoveredLine !== ''` ensures the tooltip is only visible after the first mousemove fires (which computes the correct position and text), preventing a stale flash on hover transitions.

#### 8. Add the hover dot
**File**: `webserver/view/status.templ`
**Changes**: Inside the chart container `<div class="mx-10 h-full">`, add `relative` and `overflow-hidden` to its classes and add the dot element after the `</svg>`.

Update the div:
```html
<div class="mx-10 h-full relative overflow-hidden">
```

`overflow-hidden` clips the dot at container edges so it doesn't extend into the axis label areas.

Add dot after `</svg>` (inside the same div):
```html
<!-- Hover dot -->
<div
  style="display: none; position: absolute; width: 10px; height: 10px; border-radius: 50%; transform: translate(-50%, -50%); pointer-events: none;"
  data-show="$hoverActive"
  data-style:left="$dotLeft + '%'"
  data-style:top="$dotTop + '%'"
  data-class="{'bg-primary': $hoveredLine === 'CPU', 'bg-green-500': $hoveredLine === 'Memory', 'bg-orange-500': $hoveredLine === 'Views'}"
>
  <div style="position: absolute; inset: -2px; border-radius: 50%; border: 2px solid hsl(var(--background));"></div>
</div>
```

The dot uses `data-show="$hoverActive"` (same as tooltip) so it only appears after the first mousemove computes the correct position.

The inner div provides a background-colored ring for contrast against the line.

### Success Criteria:

#### Automated Verification:
- [ ] Build succeeds: `just build`
- [ ] `just check` passes (fmt + generate + lint + build)

#### Manual Verification:
- [ ] Hover CPU line: tooltip shows "CPU: X.X% at HH:MM", dot appears in primary color on the line
- [ ] Hover Memory line: tooltip shows "Memory: X.X% at HH:MM", green dot appears on the line
- [ ] Hover Views line: tooltip shows "Views: NNNN at HH:MM", orange dot appears on the line
- [ ] Dot position matches the actual line position (not offset)
- [ ] Moving mouse along a line smoothly updates dot and tooltip
- [ ] Mouse leaving a line hides dot and tooltip
- [ ] Moving mouse directly from one line to another: no stale flash (dot/tooltip don't appear at old position)
- [ ] Switch time ranges — tooltip works correctly after graph re-render
- [ ] 3-day range doesn't feel sluggish (downsampled to ~360 points)
- [ ] Dot appears as a circle, not an ellipse (not distorted by SVG scaling)
- [ ] Dot at chart edges is cleanly clipped (no overflow into axis labels)

---

## Testing Strategy

### Manual Testing Steps:
1. Navigate to `/status`
2. Wait for graph to populate (may take 30s for first data point)
3. Hover each line and verify tooltip content and dot appearance
4. Move mouse slowly along a line and verify smooth tracking
5. Switch to each time range (1h, 6h, 24h, 3d) and repeat hover tests
6. Verify graph SSE updates (every 30s) don't break hover state
7. Test in both light and dark mode (dot border should adapt)

## Performance Considerations

- Points JSON for 1h range (~120 points): ~5KB — negligible
- Points JSON for 3d range (downsampled to ~360 points): ~15KB — acceptable
- `findNearestPoint` is O(n) linear scan on max 360 points — sub-millisecond
- Mousemove throttled to 16ms (~60fps) — matches current behavior
- Points signal update on graph SSE patch: Datastar merges signals efficiently

## Edge Cases

- **Empty graph** (no data yet): `$graphPoints` is `[]`, `findNearestPoint` returns null, `$tooltipText` stays empty, `$hoverActive` stays false — tooltip and dot remain hidden
- **Single data point**: Dot and tooltip work correctly (only one point to snap to)
- **Graph re-render during hover**: SSE replaces inner HTML of `#metrics-graph`, destroying the hit-area paths. Browser fires `mouseleave` on the removed element (clearing `$hoverActive`), so tooltip/dot hide cleanly. The new elements pick up on next `mouseenter`.
- **Dot at extreme edges**: `overflow-hidden` on the chart container clips the dot cleanly
- **Line-to-line hover transition**: `mouseenter` on the new line fires before `mousemove`. Because `$hoverActive` was cleared by the previous `mouseleave`, the dot and tooltip stay hidden until the first `mousemove` fires with the correct position for the new line — no stale flash.

## References

- Current tooltip: `webserver/view/status.templ:128-137`
- Hit-area pattern: `webserver/view/status.templ:164-175`
- Path building: `webserver/handle/status.go:337-364`
- Graph SSE patching: `webserver/handle/status.go:253-266`
- GraphData struct: `webserver/view/status.go:6-21`
