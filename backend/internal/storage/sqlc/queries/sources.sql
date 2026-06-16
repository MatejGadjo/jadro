-- name: ListActiveSources :many
SELECT id, name, base_url, rss_url, logo_url, active, rate_limit_per_min, created_at
FROM sources
WHERE active = TRUE
ORDER BY created_at ASC;

-- name: GetSourceByID :one
SELECT id, name, base_url, rss_url, logo_url, active, rate_limit_per_min, created_at
FROM sources
WHERE id = $1;

-- name: UpdateSourceLogoURL :exec
UPDATE sources
SET logo_url = $2
WHERE id = $1
  AND logo_url IS DISTINCT FROM $2;

-- name: DisableSource :exec
UPDATE sources
SET active = FALSE
WHERE id = $1;
