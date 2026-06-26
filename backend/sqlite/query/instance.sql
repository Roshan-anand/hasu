-- name: GetProductionInstanceByProject :one
SELECT id, project_id, is_production, name, network, created_by, created_at
FROM instance
WHERE project_id = ? AND is_production = true
LIMIT 1;

-- name: CreatePreviewInstance :exec
INSERT INTO instance (
    id, project_id, is_production, name, network,
    git_source_type, git_source_value, status, created_by
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetProjectIDByRepoID :one
SELECT i.project_id
FROM app_service a
JOIN instance i ON i.id = a.instance_id
WHERE a.gh_repo_id = ?
LIMIT 1;

-- name: GetPreviewInstancesByProject :many
SELECT id, project_id, is_production, name, network,
    git_source_type, git_source_value, status, created_by,
    created_at
FROM instance
WHERE project_id = ? AND is_production = false
ORDER BY created_at DESC;

-- name: GetActivePreviewByPR :one
SELECT id, project_id, is_production, name, network,
    git_source_type, git_source_value, status, created_by,
    created_at
FROM instance
WHERE project_id = ? AND git_source_type = 'pr' AND git_source_value = ?
    AND status NOT IN ('deleting', 'error')
LIMIT 1;

-- name: GetPreviewInstanceByID :one
SELECT id, project_id, is_production, name, network,
    git_source_type, git_source_value, status, created_by,
    created_at
FROM instance
WHERE id = ?;

-- name: UpdateInstanceStatus :exec
UPDATE instance
SET status = ?
WHERE id = ?;

-- name: DeletePreviewInstance :exec
DELETE FROM instance
WHERE id = ?;
