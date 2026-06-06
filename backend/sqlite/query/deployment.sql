-- name: CreateDeployment :one
INSERT INTO deployments (id, service_id, commit_hash, commit_msg, is_current)
VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: GetDeploymentsByServiceID :many
SELECT d.*, aps.swarm_service
FROM deployments d
JOIN app_service aps ON d.service_id = aps.id
WHERE d.service_id = ?
ORDER BY d.created_at DESC;

-- name: GetDeploymentImgByID :one
SELECT d.id, d.image
FROM deployments d
WHERE d.id = ?;

-- name: GetDeploymentStatus :one
SELECT status
FROM deployments
WHERE id = ?;

-- name: UpdateDeploymentStatus :exec
UPDATE deployments
SET status = ?
WHERE id = ?;

-- name: DownGradeDeployment :exec
UPDATE deployments
SET is_current = FALSE, status = ?
WHERE id = @deployment_id;

-- name: UpgradeDeployment :exec
UPDATE deployments
SET is_current = TRUE, status = ?
WHERE id = @deployment_id;

-- name: SetDeploymentImageName :exec
UPDATE deployments
SET image = ?
WHERE id = ?;

-- name: DeleteDeploymentByID :exec
DELETE FROM deployments
WHERE id = ?;
