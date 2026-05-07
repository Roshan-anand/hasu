-- name: GetAllService :many
SELECT ps.id, ps.type, ps.name, '' AS gh_repo_name, '' AS gh_repo_url, '' AS git_provider, '' AS branch_name, ps.created_at
FROM psql_service ps
WHERE ps.organization_id = @org_id
UNION ALL
SELECT aps.id, aps.type, aps.name, aps.gh_repo_url, aps.gh_repo_url, aps.git_provider, b.branch_name, aps.created_at
FROM app_service aps
JOIN app_service_branch b ON aps.default_branch_id = b.id
WHERE aps.organization_id = @org_id;

-- name: ServiceNameExists :one
SELECT CAST(
    (SELECT EXISTS (
        SELECT 1
        FROM psql_service ps
        WHERE ps.organization_id = @org_id AND ps.name = @name
        UNION ALL
        SELECT 1
        FROM app_service aps
        WHERE aps.organization_id = @org_id AND aps.name = @name
    )) 
AS BOOLEAN);

-- name: CreatePsqlService :one
INSERT INTO psql_service (id, organization_id, type, swarm_service_name, name, db_name, db_user, db_password, internal_url)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, type;

-- name: GetPsqlServiceById :one
SELECT *
FROM psql_service
WHERE id = ?;

-- name: SetPsqlSwarmServiceId :exec
UPDATE psql_service
SET swarm_service_id = ?
WHERE id = ?;

-- name: DeletePsqlService :exec
DELETE FROM psql_service
WHERE id = ?;

-- name: CreateAppService :one
INSERT INTO app_service (id, organization_id, type, name, git_provider, gh_app_id, gh_repo_id, gh_repo_name, gh_repo_url, build_path, watch_path, env, build_args, build_secrets)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, type;

-- name: DeleteAppService :exec
DELETE FROM app_service
WHERE id = ?;

-- name: GetAppServiceById :one
SELECT
    a.id, a.type, a.name, a.gh_repo_name, a.gh_repo_url, b.branch_name
FROM app_service a
JOIN app_service_branch b ON b.service_id = a.id AND b.is_default_branch = 1
WHERE a.id = ?;

-- name: CreateAppServiceBranch :one
INSERT INTO app_service_branch (id, is_default_branch, branch_name, swarm_service_name, service_id)
VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: SetAppSwarmServiceId :exec
UPDATE app_service_branch
SET swarm_service_id = ?
WHERE id = ?;

