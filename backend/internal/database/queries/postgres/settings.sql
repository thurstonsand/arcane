-- name: ListSettings :many
SELECT key, value
FROM settings;

-- name: GetSetting :one
SELECT key, value
FROM settings
WHERE key = $1
LIMIT 1;

-- name: UpsertSetting :exec
INSERT INTO settings (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- name: InsertSettingIfNotExists :exec
INSERT INTO settings (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO NOTHING;

-- name: DeleteSetting :execrows
DELETE FROM settings
WHERE key = $1;

-- name: UpdateSettingKey :exec
UPDATE settings
SET key = $2
WHERE key = $1;

-- name: DeleteSettingsNotIn :execrows
DELETE FROM settings
WHERE NOT (key = ANY($1::text[]));
