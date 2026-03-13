package handle

import (
	"context"
	"fmt"
	"runtime"
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

// memUsed returns meaningful memory usage figures.
// On macOS, vm.Used includes compressed memory (the VM compressor) which
// inflates the number to nearly 100% of physical RAM. Active + Wired
// matches what Activity Monitor reports as actual app memory usage.
func memUsed(vm *mem.VirtualMemoryStat) (usedBytes uint64, usedPercent float64) {
	if runtime.GOOS == "darwin" && vm.Active > 0 {
		usedBytes = vm.Active + vm.Wired
		if vm.Total > 0 {
			usedPercent = float64(usedBytes) / float64(vm.Total) * 100
		}
		return
	}
	return vm.Used, vm.UsedPercent
}

// systemSignals holds scalar metrics pushed via MarshalAndPatchSignals.
// JSON tags match the signal names declared in the template's data-signals.
type systemSignals struct {
	MemTotal        string `json:"memTotal,omitempty"`
	MemUsed         string `json:"memUsed,omitempty"`
	MemUsedPercent  string `json:"memUsedPercent,omitempty"`
	CpuPercent      string `json:"cpuPercent,omitempty"`
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
	"24h": {24 * time.Hour, "Last 24 Hours"},
	"3d":  {72 * time.Hour, "Last 3 Days"},
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
	ticker := time.NewTicker(30 * time.Second)
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
	ticker := time.NewTicker(1 * time.Hour)
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
	threeDaysAgo := time.Now().Add(-72 * time.Hour)
	if err := h.db.CompactOldMetrics(ctx, threeDaysAgo); err != nil {
		log.Error().Err(err).Msg("failed to compact old metrics")
	}

	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	if err := h.db.DeleteMetricsOlderThan(ctx, thirtyDaysAgo); err != nil {
		log.Error().Err(err).Msg("failed to delete old metrics")
	}
	if err := h.db.DeletePageViewsOlderThan(ctx, thirtyDaysAgo); err != nil {
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

	sysTicker := time.NewTicker(2 * time.Second)
	defer sysTicker.Stop()
	graphTicker := time.NewTicker(30 * time.Second)
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
		return echo.NewHTTPError(400, "invalid signals")
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
		signals.CpuPercent = fmt.Sprintf("%.1f%%", cpuPct[0])
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

func (h *StatusHandler) buildGraph(ctx context.Context, dur time.Duration) vi.GraphData {
	now := time.Now()
	cutoff := now.Add(-dur)

	snaps, err := h.db.GetMetricsSince(ctx, cutoff)
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
	viewsMid := viewsMin + viewsRange/2

	data := vi.GraphData{
		PointCount: n,
		CpuLast: fmt.Sprintf(
			"%.1f%%", snaps[n-1].CPUPercent,
		),
		MemLast: fmt.Sprintf(
			"%.1f%%", snaps[n-1].MemUsedPercent,
		),
		//nolint:gosec // stored as positive value
		MemBytesLast: humanize.Bytes(
			uint64(snaps[n-1].MemUsedBytes),
		),
		ViewsLast: fmt.Sprintf(
			"%d", snaps[n-1].TotalVisits,
		),
		ViewsAxisMax: fmt.Sprintf("%d", viewsMax),
		ViewsAxisMid: fmt.Sprintf("%d", viewsMid),
		ViewsAxisMin: fmt.Sprintf("%d", viewsMin),
		TimeStart:    cutoff.Format("15:04"),
		TimeEnd:      now.Format("15:04"),
	}

	// Build SVG paths.
	// CPU and Memory: left Y-axis, 0-100%, Y = 100 - value.
	// Views: right Y-axis, auto-scaled to min/max range.
	totalSeconds := now.Sub(cutoff).Seconds()
	cpuPath := ""
	memPath := ""
	viewsPath := ""

	for i, s := range snaps {
		x := 0.0
		if totalSeconds > 0 {
			x = s.CreatedAt.Sub(cutoff).Seconds() /
				totalSeconds * 100
		}
		if x < 0 {
			x = 0
		} else if x > 100 {
			x = 100
		}

		cpuY := 100 - s.CPUPercent
		memY := 100 - s.MemUsedPercent
		if cpuY < 0 {
			cpuY = 0
		} else if cpuY > 100 {
			cpuY = 100
		}
		if memY < 0 {
			memY = 0
		} else if memY > 100 {
			memY = 100
		}

		// Views: normalize to 0-100 range based on min/max.
		var viewsY float64
		if viewsRange > 0 {
			viewsY = 100 - float64(
				s.TotalVisits-viewsMin,
			)/float64(viewsRange)*100
		} else {
			viewsY = 50 // flat line when all values equal
		}

		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		cpuPath += fmt.Sprintf("%s%.1f,%.1f ", cmd, x, cpuY)
		memPath += fmt.Sprintf("%s%.1f,%.1f ", cmd, x, memY)
		viewsPath += fmt.Sprintf(
			"%s%.1f,%.1f ", cmd, x, viewsY,
		)
	}

	data.CpuPath = cpuPath
	data.MemPath = memPath
	data.ViewsPath = viewsPath
	return data
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
