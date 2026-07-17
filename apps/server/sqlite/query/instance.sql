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

-- name: GetAllProjectIDsByPR :many
SELECT p.id
FROM project p
JOIN instance i ON p.id = i.project_id AND i.is_production
WHERE
    (CAST(@project_name AS TEXT) = ''
    OR p.name = @project_name)
    AND EXISTS(
        SELECT 1 FROM app_service aps
        WHERE aps.instance_id = i.id AND aps.gh_repo_id = @repo_id
    )
    AND NOT EXISTS(
        SELECT 1 FROM instance pi
        WHERE pi.project_id = p.id AND pi.git_source_type = 'pr' AND pi.git_source_value = @pr_number
    );

-- name: GetAllInstanceByPR :many
SELECT i.id
FROM instance i
WHERE i.git_source_type = 'pr' AND i.git_source_value = @pr_number;

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
