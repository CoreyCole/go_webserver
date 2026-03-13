-- name: InsertMetricsSnapshot :exec
INSERT INTO metrics_snapshots (cpu_percent, mem_used_percent, mem_used_bytes, total_visits, unique_visitors)
VALUES (?, ?, ?, ?, ?);

-- name: GetMetricsSince :many
SELECT id, cpu_percent, mem_used_percent, mem_used_bytes, total_visits, unique_visitors, created_at
FROM metrics_snapshots
WHERE created_at >= datetime('now', ?)
ORDER BY created_at ASC;

-- name: DeleteMetricsOlderThan :exec
DELETE FROM metrics_snapshots WHERE created_at < datetime('now', ?);
