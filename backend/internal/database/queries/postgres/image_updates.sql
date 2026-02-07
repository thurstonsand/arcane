-- name: GetImageUpdateByID :one
SELECT id,
			 repository,
			 tag,
			 has_update,
			 update_type,
			 current_version,
			 latest_version,
			 current_digest,
			 latest_digest,
			 check_time,
			 response_time_ms,
			 last_error,
			 auth_method,
			 auth_username,
			 auth_registry,
			 used_credential,
			 notification_sent,
			 created_at,
			 updated_at
FROM image_updates
WHERE id = $1
LIMIT 1;

-- name: SaveImageUpdate :one
INSERT INTO image_updates (
		id,
		repository,
		tag,
		has_update,
		update_type,
		current_version,
		latest_version,
		current_digest,
		latest_digest,
		check_time,
		response_time_ms,
		last_error,
		auth_method,
		auth_username,
		auth_registry,
		used_credential,
		notification_sent,
		created_at,
		updated_at
)
VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7,
		$8,
		$9,
		$10,
		$11,
		$12,
		$13,
		$14,
		$15,
		$16,
		$17,
		NOW(),
		NOW()
)
ON CONFLICT (id) DO UPDATE SET
		repository = EXCLUDED.repository,
		tag = EXCLUDED.tag,
		has_update = EXCLUDED.has_update,
		update_type = EXCLUDED.update_type,
		current_version = EXCLUDED.current_version,
		latest_version = EXCLUDED.latest_version,
		current_digest = EXCLUDED.current_digest,
		latest_digest = EXCLUDED.latest_digest,
		check_time = EXCLUDED.check_time,
		response_time_ms = EXCLUDED.response_time_ms,
		last_error = EXCLUDED.last_error,
		auth_method = EXCLUDED.auth_method,
		auth_username = EXCLUDED.auth_username,
		auth_registry = EXCLUDED.auth_registry,
		used_credential = EXCLUDED.used_credential,
		notification_sent = EXCLUDED.notification_sent,
		updated_at = NOW()
RETURNING id,
					repository,
					tag,
					has_update,
					update_type,
					current_version,
					latest_version,
					current_digest,
					latest_digest,
					check_time,
					response_time_ms,
					last_error,
					auth_method,
					auth_username,
					auth_registry,
					used_credential,
					notification_sent,
					created_at,
					updated_at;

-- name: ListImageUpdates :many
SELECT id,
			 repository,
			 tag,
			 has_update,
			 update_type,
			 current_version,
			 latest_version,
			 current_digest,
			 latest_digest,
			 check_time,
			 response_time_ms,
			 last_error,
			 auth_method,
			 auth_username,
			 auth_registry,
			 used_credential,
			 notification_sent,
			 created_at,
			 updated_at
FROM image_updates;

-- name: ListImageUpdatesByIDs :many
SELECT id,
			 repository,
			 tag,
			 has_update,
			 update_type,
			 current_version,
			 latest_version,
			 current_digest,
			 latest_digest,
			 check_time,
			 response_time_ms,
			 last_error,
			 auth_method,
			 auth_username,
			 auth_registry,
			 used_credential,
			 notification_sent,
			 created_at,
			 updated_at
FROM image_updates
WHERE id = ANY($1::text[]);

-- name: ListImageUpdatesWithUpdate :many
SELECT id,
			 repository,
			 tag,
			 has_update,
			 update_type,
			 current_version,
			 latest_version,
			 current_digest,
			 latest_digest,
			 check_time,
			 response_time_ms,
			 last_error,
			 auth_method,
			 auth_username,
			 auth_registry,
			 used_credential,
			 notification_sent,
			 created_at,
			 updated_at
FROM image_updates
WHERE has_update = true;

-- name: ListUnnotifiedImageUpdates :many
SELECT id,
			 repository,
			 tag,
			 has_update,
			 update_type,
			 current_version,
			 latest_version,
			 current_digest,
			 latest_digest,
			 check_time,
			 response_time_ms,
			 last_error,
			 auth_method,
			 auth_username,
			 auth_registry,
			 used_credential,
			 notification_sent,
			 created_at,
			 updated_at
FROM image_updates
WHERE has_update = true
	AND notification_sent = false;

-- name: MarkImageUpdatesNotified :exec
UPDATE image_updates
SET notification_sent = true,
		updated_at = NOW()
WHERE id = ANY($1::text[]);

-- name: DeleteImageUpdatesByIDs :execrows
DELETE FROM image_updates
WHERE id = ANY($1::text[]);

-- name: CountImageUpdates :one
SELECT COUNT(*)
FROM image_updates;

-- name: CountImageUpdatesWithUpdate :one
SELECT COUNT(*)
FROM image_updates
WHERE has_update = true;

-- name: CountImageUpdatesWithUpdateType :one
SELECT COUNT(*)
FROM image_updates
WHERE has_update = true
	AND update_type = $1;

-- name: CountImageUpdatesWithErrors :one
SELECT COUNT(*)
FROM image_updates
WHERE last_error IS NOT NULL;

-- name: UpdateImageUpdateHasUpdateByRepositoryTag :exec
UPDATE image_updates
SET has_update = $3,
		updated_at = NOW()
WHERE repository = $1
	AND tag = $2;
