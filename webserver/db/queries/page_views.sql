-- name: InsertPageView :exec
INSERT INTO page_views (path, visitor_hash) VALUES (?, ?);

-- name: CountTotalPageViews :one
SELECT COUNT(*) FROM page_views;

-- name: CountUniqueVisitors :one
SELECT COUNT(DISTINCT visitor_hash) FROM page_views WHERE visitor_hash != '';

-- name: DeletePageViewsOlderThan :exec
DELETE FROM page_views WHERE created_at < ?;
