-- name: CreateDeployment :one
INSERT INTO deployments (id, branch_id, commit_msg)
VALUES (?, ?, ?)
RETURNING id;

-- name: GetDeploymentsByServiceID :many
SELECT d.id, d.status, d.commit_msg, b.branch_name, d.created_at
FROM deployments d
JOIN app_service_branch b ON d.branch_id = b.id
WHERE b.service_id = ?
ORDER BY created_at DESC;

-- name: GetAllDeploymentIdsByServiceID :many
SELECT d.id
FROM deployments d
JOIN app_service_branch b ON d.branch_id = b.id
WHERE b.service_id = ?;

-- name: GetDeploymentStatus :one
SELECT status
FROM deployments
WHERE id = ?;

-- name: UpdateDeploymentStatus :exec
UPDATE deployments
SET status = ?
WHERE id = ?;

-- name: SetDeploymentImageID :exec
UPDATE deployments
SET image_id = ?
WHERE id = ?;

-- name: DeleteDeploymentByID :exec
DELETE FROM deployments
WHERE id = ?;
