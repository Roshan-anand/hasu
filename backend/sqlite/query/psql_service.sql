-- name: CreatePsqlService :one
INSERT INTO psql_service (id, project_id, type, swarm_service_name, name, db_name, db_user, db_password, internal_url, image, volume)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: AssociateVolumeWithPsql :exec
UPDATE psql_service
SET volume = ?
WHERE id = ?;

-- name: GetPsqlServiceById :one
SELECT ps.*, pr.organization_id
FROM psql_service ps
JOIN project pr ON ps.project_id = pr.id
WHERE ps.id = ?;

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
INSERT INTO orphan_volume (id, organization_id, volume, type)
VALUES (?, ?, ?, ?);

-- name: GetAllAttachableOrphanVolumes :many
SELECT *
FROM orphan_volume
WHERE organization_id = ?;

-- name: GetAllOrphanVolumesByOrgID :many
SELECT *
FROM orphan_volume
WHERE organization_id = ?;

-- name: GetOrphanVolumeByType :many
SELECT *
FROM orphan_volume
WHERE organization_id = ? AND type = ?;

-- name: DeleteOrphanVolume :exec
DELETE FROM orphan_volume
WHERE volume = ?;
