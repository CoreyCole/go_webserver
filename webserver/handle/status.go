package handle

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/coreycole/go_webserver/webserver/db"
	"github.com/coreycole/go_webserver/webserver/db/sqlc"
	vi "github.com/coreycole/go_webserver/webserver/view"
	"github.com/coreycole/go_webserver/webserver/view/layout"
)

const (
	percentScale       = 100 // Y-axis percentage scale
	percentMid         = 50  // midpoint for flat-line rendering
	snapshotIntervalS  = 30  // seconds between metric snapshots
	metricsIntervalS   = 2   // seconds between SSE metric pushes
	hoursPerDay        = 24  // hours in a day
	minutesPerHour     = 60  // minutes in an hour
	retentionDays      = 30  // days to keep metrics/page views
	compactionHours    = 72  // hours before compaction kicks in
	halfDivisor        = 2   // divisor for midpoint calculation
	retentionCheckHour = 1   // hours between retention cleanup runs
)

// memUsed returns meaningful memory usage figures.
// On macOS, vm.Used includes compressed memory (the VM compressor) which
// inflates the number to nearly 100% of physical RAM. Active + Wired
// matches what Activity Monitor reports as actual app memory usage.
func memUsed(vm *mem.VirtualMemoryStat) (usedBytes uint64, usedPercent float64) {
	if runtime.GOOS == "darwin" && vm.Active > 0 {
		usedBytes = vm.Active + vm.Wired
		if vm.Total > 0 {
			usedPercent = float64(usedBytes) / float64(vm.Total) * percentScale
		}
		return usedBytes, usedPercent
	}
	return vm.Used, vm.UsedPercent
}

// systemSignals holds scalar metrics pushed via MarshalAndPatchSignals.
// JSON tags match the signal names declared in the template's data-signals.
type systemSignals struct {
	MemTotal        string `json:"memTotal,omitempty"`
	MemUsed         string `json:"memUsed,omitempty"`
	MemUsedPercent  string `json:"memUsedPercent,omitempty"`
	CPUPercent      string `json:"cpuPercent,omitempty"`
	Uptime          string `json:"uptime,omitempty"`
	DiskTotal       string `json:"diskTotal,omitempty"`
	DiskUsed        string `json:"diskUsed,omitempty"`
	DiskUsedPercent string `json:"diskUsedPercent,omitempty"`
}

type graphRangeSignals struct {
	GraphRange string `json:"graphRange"`
}

var graphRanges = map[string]struct {
	duration time.Duration
	label    string
}{
	"1h":  {1 * time.Hour, "Last Hour"},
	"6h":  {6 * time.Hour, "Last 6 Hours"},
	"24h": {hoursPerDay * time.Hour, "Last 24 Hours"},
	"3d":  {compactionHours * time.Hour, "Last 3 Days"},
}

// StatusHandler serves the /status page and SSE events.
// Metrics snapshots are persisted to SQLite via the db package.
type StatusHandler struct {
	db        *db.DB
	startedAt time.Time
	cancel    context.CancelFunc
}

// NewStatusHandler creates a handler and starts background goroutines.
func NewStatusHandler(database *db.DB) *StatusHandler {
	ctx, cancel := context.WithCancel(context.Background())
	h := &StatusHandler{
		db:        database,
		startedAt: time.Now(),
		cancel:    cancel,
	}
	go h.runSnapshotWriter(ctx)
	go h.runRetentionCleanup(ctx)
	return h
}

func (h *StatusHandler) runSnapshotWriter(ctx context.Context) {
	h.takeSnapshot(ctx)
	ticker := time.NewTicker(snapshotIntervalS * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.takeSnapshot(ctx)
		}
	}
}

func (h *StatusHandler) takeSnapshot(ctx context.Context) {
	var cpuPct, memPct float64
	var memBytes uint64

	if vm, err := mem.VirtualMemory(); err == nil {
		memBytes, memPct = memUsed(vm)
	}
	if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
		cpuPct = pct[0]
	}

	totalVisits, _ := h.db.CountTotalPageViews(ctx)
	uniqueVisitors, _ := h.db.CountUniqueVisitors(ctx)

	err := h.db.InsertMetricsSnapshot(ctx, sqlc.InsertMetricsSnapshotParams{
		CPUPercent:     cpuPct,
		MemUsedPercent: memPct,
		//nolint:gosec // memory bytes won't overflow int64
		MemUsedBytes:   int64(memBytes),
		TotalVisits:    totalVisits,
		UniqueVisitors: uniqueVisitors,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to insert metrics snapshot")
	}
}

func (h *StatusHandler) runRetentionCleanup(ctx context.Context) {
	ticker := time.NewTicker(retentionCheckHour * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.doRetentionCleanup(ctx)
		}
	}
}

func (h *StatusHandler) doRetentionCleanup(ctx context.Context) {
	if err := h.db.CompactOldMetrics(ctx, "-3 days"); err != nil {
		log.Error().Err(err).Msg("failed to compact old metrics")
	}

	if err := h.db.DeleteMetricsOlderThan(ctx, "-30 days"); err != nil {
		log.Error().Err(err).Msg("failed to delete old metrics")
	}
	if err := h.db.DeletePageViewsOlderThan(ctx, "-30 days"); err != nil {
		log.Error().Err(err).Msg("failed to delete old page views")
	}
}

// GetStatus renders the status page.
func (h *StatusHandler) GetStatus(c echo.Context) error {
	return vi.StatusPage(layout.RootArgs{Title: "Status"}).
		Render(c.Request().Context(), c.Response().Writer)
}

// GetStatusEvents streams SSE events for system metrics and the metrics graph.
func (h *StatusHandler) GetStatusEvents(c echo.Context) error {
	sse := datastar.NewSSE(c.Response().Writer, c.Request())

	// Fire immediately on connection.
	h.sendSystemMetrics(sse)
	h.sendGraph(sse, c.Request().Context(), 1*time.Hour, "Last Hour")
	h.sendStatsOverview(sse, c.Request().Context())

	sysTicker := time.NewTicker(metricsIntervalS * time.Second)
	defer sysTicker.Stop()
	graphTicker := time.NewTicker(snapshotIntervalS * time.Second)
	defer graphTicker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-sysTicker.C:
			h.sendSystemMetrics(sse)
			h.sendStatsOverview(sse, c.Request().Context())
		case <-graphTicker.C:
			h.sendGraph(sse, c.Request().Context(), 1*time.Hour, "Last Hour")
		}
	}
}

// PostGraphUpdate handles POST /status/graph — reads graphRange signal, returns updated graph.
func (h *StatusHandler) PostGraphUpdate(c echo.Context) error {
	var signals graphRangeSignals
	if err := datastar.ReadSignals(c.Request(), &signals); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid signals")
	}

	entry, ok := graphRanges[signals.GraphRange]
	if !ok {
		entry = graphRanges["1h"]
	}

	sse := datastar.NewSSE(c.Response().Writer, c.Request())
	h.sendGraph(sse, c.Request().Context(), entry.duration, entry.label)
	return nil
}

func (h *StatusHandler) sendSystemMetrics(sse *datastar.ServerSentEventGenerator) {
	signals := systemSignals{
		Uptime: formatDuration(time.Since(h.startedAt)),
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		usedBytes, usedPercent := memUsed(vm)
		signals.MemTotal = humanize.Bytes(vm.Total)
		signals.MemUsed = humanize.Bytes(usedBytes)
		signals.MemUsedPercent = fmt.Sprintf("%.1f%%", usedPercent)
	}
	if cpuPct, err := cpu.Percent(0, false); err == nil && len(cpuPct) > 0 {
		signals.CPUPercent = fmt.Sprintf("%.1f%%", cpuPct[0])
	}
	if du, err := disk.Usage("/"); err == nil {
		signals.DiskTotal = humanize.Bytes(du.Total)
		signals.DiskUsed = humanize.Bytes(du.Used)
		signals.DiskUsedPercent = fmt.Sprintf("%.1f%%", du.UsedPercent)
	}
	_ = sse.MarshalAndPatchSignals(signals)
}

func (h *StatusHandler) sendStatsOverview(
	sse *datastar.ServerSentEventGenerator,
	ctx context.Context,
) {
	totalVisits, _ := h.db.CountTotalPageViews(ctx)
	uniqueVisitors, _ := h.db.CountUniqueVisitors(ctx)
	_ = sse.PatchElementTempl(
		vi.StatsOverview(totalVisits, uniqueVisitors),
		datastar.WithSelectorID("stats-overview"),
		datastar.WithModeInner(),
	)
}

func (h *StatusHandler) sendGraph(
	sse *datastar.ServerSentEventGenerator,
	ctx context.Context,
	dur time.Duration,
	label string,
) {
	graphData := h.buildGraph(ctx, dur)
	graphData.TimeLabel = label
	_ = sse.PatchElementTempl(
		vi.MetricsGraph(graphData),
		datastar.WithSelectorID("metrics-graph"),
		datastar.WithModeInner(),
	)
}

// sqliteOffset converts a time.Duration to a SQLite datetime modifier string.
func sqliteOffset(dur time.Duration) string {
	secs := int(dur.Seconds())
	return fmt.Sprintf("-%d seconds", secs)
}

func (h *StatusHandler) buildGraph(ctx context.Context, dur time.Duration) vi.GraphData {
	now := time.Now()
	cutoff := now.Add(-dur)

	snaps, err := h.db.GetMetricsSince(ctx, sqliteOffset(dur))
	if err != nil {
		log.Error().Err(err).Msg("failed to get metrics for graph")
		return vi.GraphData{}
	}

	if len(snaps) == 0 {
		return vi.GraphData{}
	}

	// Pad at range start using first real value's data.
	if snaps[0].CreatedAt.Sub(cutoff) > time.Minute {
		padded := snaps[0]
		padded.CreatedAt = cutoff
		snaps = append([]sqlc.MetricsSnapshot{padded}, snaps...)
	}

	n := len(snaps)

	// Find views min/max for right Y-axis scaling.
	viewsMin := snaps[0].TotalVisits
	viewsMax := snaps[0].TotalVisits
	for _, s := range snaps[1:] {
		if s.TotalVisits < viewsMin {
			viewsMin = s.TotalVisits
		}
		if s.TotalVisits > viewsMax {
			viewsMax = s.TotalVisits
		}
	}
	viewsRange := viewsMax - viewsMin
	viewsMid := viewsMin + viewsRange/halfDivisor

	data := vi.GraphData{
		PointCount: n,
		CPULast: fmt.Sprintf(
			"%.1f%%", snaps[n-1].CPUPercent,
		),
		MemLast: fmt.Sprintf(
			"%.1f%%", snaps[n-1].MemUsedPercent,
		),
		//nolint:gosec // stored as positive value
		MemBytesLast: humanize.Bytes(
			uint64(snaps[n-1].MemUsedBytes),
		),
		ViewsLast:    strconv.FormatInt(snaps[n-1].TotalVisits, 10),
		ViewsAxisMax: strconv.FormatInt(viewsMax, 10),
		ViewsAxisMid: strconv.FormatInt(viewsMid, 10),
		ViewsAxisMin: strconv.FormatInt(viewsMin, 10),
		TimeStart:    cutoff.Format("15:04"),
		TimeEnd:      now.Format("15:04"),
	}

	// Compute coordinates for each snapshot once.
	totalSeconds := now.Sub(cutoff).Seconds()

	type pointCoords struct {
		x, cpuY, memY, viewsY float64
		timeStr               string
		cpuVal, memVal        string
		viewsVal              string
	}

	points := make([]pointCoords, n)
	for i, s := range snaps {
		x := 0.0
		if totalSeconds > 0 {
			x = s.CreatedAt.Sub(cutoff).Seconds() /
				totalSeconds * percentScale
		}
		x = clamp(x)

		cpuY := clamp(percentScale - s.CPUPercent)
		memY := clamp(percentScale - s.MemUsedPercent)

		var viewsY float64
		if viewsRange > 0 {
			viewsY = percentScale - float64(
				s.TotalVisits-viewsMin,
			)/float64(viewsRange)*percentScale
		} else {
			viewsY = percentMid
		}

		points[i] = pointCoords{
			x: x, cpuY: cpuY, memY: memY, viewsY: viewsY,
			timeStr:  s.CreatedAt.Local().Format("15:04"),
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

	return data
}

// clamp restricts a value to the [0, 100] range.
func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > percentScale {
		return percentScale
	}
	return v
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / hoursPerDay
	hours := int(d.Hours()) % hoursPerDay
	mins := int(d.Minutes()) % minutesPerHour
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
