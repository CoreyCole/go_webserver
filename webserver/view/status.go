package view

// GraphData holds SVG path data for the metrics line graph.
// CPU and Memory share the left 0-100% Y-axis.
// Views uses an independent right Y-axis auto-scaled to min/max.
type GraphData struct {
	CpuPath      string // SVG path d attribute for CPU usage line
	MemPath      string // SVG path d attribute for memory usage line
	ViewsPath    string // SVG path d attribute for page views line
	CpuLast      string // current CPU %, e.g. "12.3%"
	MemLast      string // current memory %, e.g. "45.2%"
	MemBytesLast string // current memory in bytes, e.g. "7.2 GB"
	ViewsLast    string // current cumulative views, e.g. "1,234"
	ViewsAxisMax string // right Y-axis top label
	ViewsAxisMid string // right Y-axis middle label
	ViewsAxisMin string // right Y-axis bottom label
	PointCount   int    // number of data points rendered
	TimeLabel    string // e.g. "Last Hour", "Last 6 Hours"
	TimeStart    string // left edge timestamp, e.g. "14:30"
	TimeEnd      string // right edge timestamp, e.g. "15:30"
}
