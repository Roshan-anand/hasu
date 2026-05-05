-- name: GetAllService :many
SELECT ps.id, ps.type, ps.name, '' AS gh_repo_name, '' AS gh_repo_url, '' AS git_provider, '' AS branch_name, ps.created_at
FROM psql_service ps
WHERE ps.organization_id = @org_id
UNION ALL
SELECT aps.id, aps.type, aps.name, aps.gh_repo_url, aps.gh_repo_url, aps.git_provider, b.branch_name, aps.created_at
FROM app_service aps
JOIN app_service_branch b ON aps.default_branch_id = b.id
WHERE aps.organization_id = @org_id;

-- name: CreatePsqlService :one
INSERT INTO psql_service (id, organization_id, type, service_id, name, app_name, description, db_name, db_user, db_password, image, internal_url)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, type;

-- name: GetPsqlServiceById :one
SELECT *
FROM psql_service
WHERE id = ?;

-- name: SetPsqlServiceId :exec
UPDATE psql_service
SET service_id = ?
WHERE id = ?;

-- name: DeletePsqlService :exec
DELETE FROM psql_service
WHERE id = ?;

-- name: CreateAppService :one
INSERT INTO app_service (id, organization_id, type, service_id, name, app_name, git_provider, gh_app_id, gh_repo_id, gh_repo_name, gh_repo_url, build_path, watch_path, env, build_args, build_secrets, default_branch_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, type;

-- name: CreateAppServiceBranch :exec
INSERT INTO app_service_branch (id, branch_name, file_path, app_service_id)
VALUES (?, ?, ?, ?);

-- name: GetAppServiceById :one
SELECT
    a.id, a.type, a.name, a.gh_repo_name, a.gh_repo_url, b.branch_name
FROM app_service a
JOIN app_service_branch b ON app.default_branch_id = branch.id
WHERE a.id = ?;

-- name: SetAppServiceId :exec
UPDATE app_service
SET service_id = ?
WHERE id = ?;

-- name: DeleteAppService :exec
DELETE FROM app_service
WHERE id = ?;
