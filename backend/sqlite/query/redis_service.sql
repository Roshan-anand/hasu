-- name: GetRedisServicesByInstanceID :many
SELECT * FROM redis_service WHERE instance_id = ?;

-- name: CreateRedisService :one
INSERT INTO redis_service (id, instance_id, type, status, swarm_service, name, password, internal_url, image, volume)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, type;

-- name: AssociateVolumeWithRedis :exec
UPDATE redis_service
SET volume = ?
WHERE id = ?;

-- name: GetRedisServiceById :one
SELECT rs.*, p.organization_id
FROM redis_service rs
JOIN instance i ON i.id = rs.instance_id
JOIN project p ON p.id = i.project_id
WHERE rs.id = @service_id;

-- name: DeleteRedisService :exec
DELETE FROM redis_service
WHERE id = ?;

-- name: UpdateRedisServiceDetails :exec
UPDATE redis_service
SET password = ?,
    internal_url = ?
WHERE id = ?;
