-- name: CreatePsqlService :one
INSERT INTO psql_service (id, instance_id, type, swarm_service, name, db_name, db_user, db_password, internal_url, image, volume)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, type;

-- name: AssociateVolumeWithPsql :exec
UPDATE psql_service
SET volume = ?
WHERE id = ?;

-- name: GetPsqlServiceById :one
SELECT ps.*, p.organization_id
FROM psql_service ps
JOIN instance i ON i.id = ps.instance_id
JOIN project p ON p.id = i.project_id
WHERE ps.id = @service_id;

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

-- name: GetOrphanVolumeByName :one
SELECT *
FROM orphan_volume
WHERE volume = ?;

-- name: GetOrphanVolumeById :one
SELECT *
FROM orphan_volume
WHERE id = ?;

-- name: ClaimOrphanVolume :execrows
DELETE FROM orphan_volume
WHERE volume = ? AND organization_id = ?;

-- name: UpdateOrphanVolumeName :exec
UPDATE orphan_volume
SET display_name = ?
WHERE id = ? AND organization_id = ?;

-- name: DeleteOrphanVolume :exec
DELETE FROM orphan_volume
WHERE volume = ? AND organization_id = ?;

-- name: TransferOrphanVolume :exec
UPDATE orphan_volume
SET organization_id = ?
WHERE id = ? AND organization_id = ?;

-- name: GetOrphanVolumesByOrgId :many
SELECT *
FROM orphan_volume
WHERE organization_id = ?;
