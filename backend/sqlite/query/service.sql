-- name: GetAllService :many
SELECT ps.id, ps.type, ps.name, '' AS gh_repo_name, '' AS gh_repo_url, '' AS git_provider, '' AS branch_name, ps.created_at
FROM psql_service ps
WHERE ps.instance_id = @instance_id
UNION ALL
SELECT aps.id, aps.type, aps.name, aps.gh_repo_url, aps.gh_repo_url, aps.git_provider, aps.branch, aps.created_at
FROM app_service aps
WHERE aps.instance_id = @instance_id;

-- name: GetServiceID :one
SELECT ps.id 
FROM psql_service ps
WHERE ps.instance_id = @instance_id AND ps.name = @name
UNION ALL
SELECT aps.id 
FROM app_service aps
WHERE aps.instance_id = @instance_id AND aps.name = @name;

-- name: ServiceNameExists :one
SELECT CAST(
    (SELECT EXISTS (
        SELECT 1
        FROM psql_service ps
        WHERE ps.instance_id = @instance_id AND ps.name = @name
        UNION ALL
        SELECT 1
        FROM app_service aps
        WHERE aps.instance_id = @instance_id AND aps.name = @name
    ))
AS BOOLEAN);

-- name: CreateAppService :one
INSERT INTO app_service (id, instance_id, type, name, git_provider, gh_app_id, gh_repo_id, gh_repo_name, gh_repo_url, build_path, watch_path, env, build_args, build_secrets, docker_filepath, docker_contextpath, docker_buildstage, is_public, branch, swarm_service, port, internal_url)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, type;

-- name: CheckServiceIsProduction :one
SELECT is_production
FROM app_service aps
JOIN instance i ON i.id = aps.instance_id
WHERE aps.id = @service_id;

-- name: CheckProjectHasService :one
SELECT CAST(EXISTS(
    SELECT 1
    FROM app_service aps
    JOIN instance i ON i.id = aps.instance_id
    WHERE i.project_id = @project_id
    UNION ALL
    SELECT 1
    FROM psql_service ps
    JOIN instance i ON i.id = ps.instance_id
    WHERE i.project_id = @project_id
) AS BOOLEAN);

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
    a.id, a.name, a.gh_repo_name, a.gh_repo_url, a.is_public, a.branch, a.swarm_service, a.domain, a.internal_url, a.port, a.created_at,
    d.status, d.commit_msg
FROM app_service a
JOIN deployments d ON d.service_id = a.id AND d.is_current
WHERE a.id = ?;

-- name: GetAppServiceForRebuild :one
SELECT
    a.id, a.name, a.gh_repo_url, a.gh_app_id, a.gh_repo_id, a.branch, a.build_path, a.docker_filepath, a.docker_contextpath, a.docker_buildstage, a.env, a.build_args, a.build_secrets, a.swarm_service,
    d.id AS deployment_id, d.status AS deployment_status
FROM app_service a
JOIN deployments d ON d.service_id = a.id AND d.is_current
WHERE a.id = ?;

-- name: DeleteAppService :exec
DELETE FROM app_service
WHERE id = ?;

-- name: GetAllSwarmServiceAndImgByAppServiceId :many
SELECT aps.swarm_service, d.id AS deployment_id, d.image
FROM app_service aps
JOIN deployments d ON d.service_id = aps.id
WHERE aps.id = @service_id;

-- name: GetSwarmServiceByServiceId :one
SELECT swarm_service
FROM app_service aps
WHERE aps.id = @service_id
UNION ALL
SELECT swarm_service
FROM psql_service ps
WHERE ps.id = @service_id;

-- name: GetAllAppServicesByRepo :many
SELECT a.id AS service_id, a.name, a.gh_repo_url, a.gh_app_id,
    a.build_path, a.watch_path, a.env, a.build_args, a.build_secrets,
    a.docker_filepath, a.docker_contextpath, a.docker_buildstage,
    a.branch, a.swarm_service, a.domain, a.port,
    d.id AS deployment_id, d.status AS deployment_status
FROM app_service a
JOIN deployments d ON d.service_id = a.id AND d.is_current
WHERE a.gh_repo_id = ? AND a.branch = ?;

-- name: GetDomainAndPortByServiceId :one
SELECT domain, port
FROM app_service
WHERE id = @service_id;

-- name: UpdateDomianAndPort :exec
UPDATE app_service
SET domain = ?, port = ?
WHERE id = @service_id;
