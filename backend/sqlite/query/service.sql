-- name: GetAllService :many
SELECT ps.id, ps.type, ps.name, '' AS gh_repo_name, '' AS gh_repo_url, '' AS git_provider, '' AS branch_name, ps.created_at
FROM psql_service ps
WHERE ps.project_id = @project_id
UNION ALL
SELECT aps.id, aps.type, aps.name, aps.gh_repo_url, aps.gh_repo_url, aps.git_provider, b.branch_name, aps.created_at
FROM app_service aps
JOIN app_service_branch b ON aps.id = b.service_id AND b.is_default_branch = 1
WHERE aps.project_id = @project_id;

-- name: ServiceNameExists :one
SELECT CAST(
    (SELECT EXISTS (
        SELECT 1
        FROM psql_service ps
        WHERE ps.project_id = @project_id AND ps.name = @name
        UNION ALL
        SELECT 1
        FROM app_service aps
        WHERE aps.project_id = @project_id AND aps.name = @name
    ))
AS BOOLEAN);

-- name: CreatePsqlService :one
INSERT INTO psql_service (id, project_id, type, swarm_service_name, name, db_name, db_user, db_password, internal_url, image_name)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: GetPsqlServiceById :one
SELECT *
FROM psql_service
WHERE id = ?;

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

-- name: CreateAppService :one
INSERT INTO app_service (id, project_id, type, name, git_provider, gh_app_id, gh_repo_id, gh_repo_name, gh_repo_url, build_path, watch_path, env, build_args, build_secrets, docker_filepath, docker_contextpath, docker_buildstage)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, type;

-- name: GetServiceEnv :one
SELECT env, build_args, build_secrets
FROM app_service
WHERE id = ?;

-- name: UpdateAppServiceEnv :exec
UPDATE app_service
SET env = ?, build_args = ?, build_secrets = ?
WHERE id = ?;

-- name: GetAppServiceById :one
SELECT
    a.*,
    d.status, d.commit_msg,
    b.id AS branch_id, b.branch_name, b.domain
FROM app_service a
JOIN app_service_branch b ON b.service_id = a.id AND b.is_default_branch = 1
JOIN deployments d ON d.branch_id = b.id AND d.is_current = 1
WHERE a.id = ?;

-- name: DeleteAppService :exec
DELETE FROM app_service
WHERE id = ?;

-- name: CreateAppServiceBranch :one
INSERT INTO app_service_branch (id, is_default_branch, is_public, branch_name, swarm_service_name, service_id, port)
VALUES (?, ?, ?, ?, ?, ?, 80)
RETURNING id;

-- name: CheckBranchExists :one
SELECT CAST(
    (SELECT EXISTS (
        SELECT 1
        FROM app_service_branch
        WHERE service_id = @service_id AND branch_name = @branch_name
    ))
AS BOOLEAN);

-- name: GetDefaultBranchByServiceId :one
SELECT *
FROM app_service_branch
WHERE service_id = ? AND is_default_branch = 1;

-- name: GetAppServiceByBranchId :one
SELECT a.id AS service_id, a.name,a.gh_repo_id, a.gh_repo_url, a.gh_app_id,
    a.build_path, a.env, a.build_args, a.build_secrets,
    a.docker_filepath, a.docker_contextpath, a.docker_buildstage,
    b.id AS branch_id, b.branch_name, b.swarm_service_name, b.domain, b.port,
    d.id AS deployment_id, d.status AS deployment_status
FROM app_service a
JOIN app_service_branch b ON b.service_id = a.id
JOIN deployments d ON d.branch_id = b.id AND d.is_current = 1
WHERE b.id = @branch_id;

-- name: GetAllAppServicesByRepo :many
SELECT a.id AS service_id, a.name, a.gh_repo_url, a.gh_app_id,
    a.build_path, a.env, a.build_args, a.build_secrets,
    a.docker_filepath, a.docker_contextpath, a.docker_buildstage,
    b.id AS branch_id, b.branch_name, b.swarm_service_name, b.domain, b.port,
    d.id AS deployment_id, d.status AS deployment_status
FROM app_service a
JOIN app_service_branch b ON b.service_id = a.id
JOIN deployments d ON d.branch_id = b.id AND d.is_current = 1
WHERE a.gh_repo_id = ? AND b.branch_name = ?;

-- name: GetSwarmServiceByBranchId :one
SELECT swarm_service_name
FROM app_service_branch
WHERE id = @branch_id;

-- name: GetAllSwarmServiceAndImagesByAppServiceId :many
SELECT b.swarm_service_name, d.id AS deployment_id, d.image_name
FROM app_service_branch b
JOIN deployments d ON d.branch_id = b.id
WHERE b.service_id = ?;

-- name: GetAllSwarmServiceByAppServiceId :many
SELECT swarm_service_name
FROM app_service_branch
WHERE service_id = ?;

-- name: GetDefaultBranchSwarmService :one
SELECT swarm_service_name
FROM app_service_branch
WHERE service_id = ? AND is_default_branch = 1;

-- name: SetDomianAndPortForBranch :exec
UPDATE app_service_branch
SET domain = ?, port = ?
WHERE id = ?;

-- name: GetBranchesDomainByServiceId :many
SELECT b.id AS branch_id, b.branch_name, b.domain, b.port
FROM app_service_branch b
WHERE b.service_id = ?;
