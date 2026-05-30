-- name: CreatePsqlService :one
INSERT INTO psql_service (id, project_id, type, swarm_service_name, name, db_name, db_user, db_password, internal_url, image, volume)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: GetPsqlServiceById :one
SELECT *
FROM psql_service
WHERE id = ?;

-- name: DeletePsqlService :exec
DELETE FROM psql_service
WHERE id = ?;

-- name: UpdatePsqlServiceDetails :exec
UPDATE psql_service
SET db_name = ?,
    db_user = ?,
    db_password = ?,
    internal_url = ?
WHERE id = ?;

-- name: CreateOrphanVolume :exec
INSERT INTO orphan_volume (id, project_id, volume, type)
VALUES (?, ?, ?, ?);

-- name: GetAllOrphanVolumes :many
SELECT *
FROM orphan_volume
WHERE project_id = ? OR project_id IS NULL;

-- name: GetOrphanVolumeByType :many
SELECT *
FROM orphan_volume
WHERE (project_id = ? OR project_id IS NULL) AND type = ?;

-- name: DisAssociateOrphanVolume :exec
UPDATE orphan_volume
SET project_id = NULL
WHERE volume = ?;

-- name: AssociateOrphanVolume :exec
UPDATE orphan_volume
SET project_id = ?
WHERE volume = ?;

-- name: DeleteOrphanVolume :exec
DELETE FROM orphan_volume
WHERE volume = ?;
