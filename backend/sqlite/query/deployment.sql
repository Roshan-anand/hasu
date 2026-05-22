-- name: CreateDeployment :one
INSERT INTO deployments (id, branch_id, commit_msg, is_current)
VALUES (?, ?, ?, ?)
RETURNING id;

-- name: GetDeploymentsByServiceID :many
SELECT d.id, d.status, d.commit_msg, b.branch_name, d.created_at
FROM deployments d
JOIN app_service_branch b ON d.branch_id = b.id
WHERE b.service_id = ?
ORDER BY d.created_at DESC;

-- name: GetDeploymentsByBranchID :many
SELECT d.id, d.is_current, d.image_name, d.status, b.swarm_service_name
FROM deployments d
JOIN app_service_branch b ON d.branch_id = b.id
WHERE d.branch_id = ?
ORDER BY d.created_at DESC;

-- name: GetAllDeploymentImgByServiceID :many
SELECT d.id, d.image_name
FROM deployments d
JOIN app_service_branch b ON d.branch_id = b.id
WHERE b.service_id = ?;

-- name: GetDeploymentImgByID :one
SELECT d.id, d.image_name
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
SET is_current = 0, status = ?
WHERE id = @deployment_id;

-- name: UpgradeDeployment :exec
UPDATE deployments
SET is_current = 1, status = ?
WHERE id = @deployment_id;

-- name: SetDeploymentImageName :exec
UPDATE deployments
SET image_name = ?
WHERE id = ?;

-- name: DeleteDeploymentByID :exec
DELETE FROM deployments
WHERE id = ?;
