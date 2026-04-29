-- name: GetAllServicesByProjectId :many
SELECT id, type, name, description, created_at
FROM psql_service ps
WHERE ps.project_id = @projectId
UNION ALL
SELECT id, type, name, description, created_at
FROM app_service aps
WHERE aps.project_id = @projectId;

-- name: GetAllServicesByOrgId :many
SELECT ps.id, ps.type, ps.name, ps.description, ps.created_at
FROM psql_service ps
JOIN project p ON p.id = ps.project_id
WHERE p.organization_id = @org_id
UNION ALL
SELECT aps.id, aps.type, aps.name, aps.description, aps.created_at
FROM app_service aps
JOIN project p ON p.id = aps.project_id
WHERE p.organization_id = @org_id;

-- name: CreatePsqlService :one
INSERT INTO psql_service (id, project_id, type, service_id, name, app_name, description, db_name, db_user, db_password, image, internal_url)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, type;

-- name: GetPsqlServiceById :one
SELECT *
FROM psql_service
WHERE id = ?;

-- name: CreateAppService :one
INSERT INTO app_service (id, project_id, type, name, app_name, description, git_provider, git_repo_id, git_repo_name, git_branch, build_path)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, type;

-- name: GetAppServiceById :one
SELECT *
FROM app_service
WHERE id = ?;

-- name: SetPsqlServiceId :exec
UPDATE psql_service
SET service_id = ?
WHERE id = ?;

-- name: DeletePsqlService :exec
DELETE FROM psql_service
WHERE id = ?;

-- name: DeleteAppService :exec
DELETE FROM app_service
WHERE id = ?;

-- name: GetAllPsqlServicesByProjectId :many
SELECT *
FROM psql_service
WHERE project_id = ?;

-- name: CreateDeployment :one
INSERT INTO deployments (id, service_id, name, status)
VALUES (?, ?, ?, ?)
RETURNING id;

-- name: GetDeploymentByID :one
SELECT id, service_id, name, status, created_at
FROM deployments
WHERE id = ?;

-- name: GetDeploymentsByServiceID :many
SELECT id, service_id, name, status, created_at
FROM deployments
WHERE service_id = ?
ORDER BY created_at DESC;

-- name: UpdateDeploymentStatus :exec
UPDATE deployments
SET status = ?
WHERE id = ?;

-- name: DeleteDeploymentByID :exec
DELETE FROM deployments
WHERE id = ?;
