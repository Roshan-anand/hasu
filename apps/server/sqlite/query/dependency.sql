-- name: CreateServiceDependency :one
INSERT INTO service_dependencies (
    id, source_service_id, target_service_id, target_col, env_key, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, datetime('now'), datetime('now')
)
RETURNING *;

-- name: GetServiceDependencyByID :one
SELECT * FROM service_dependencies WHERE id = ?;

-- name: GetServiceDependencies :many
WITH targets AS (
    SELECT id, 'app' AS service_type, name FROM app_service
    UNION ALL
    SELECT id, 'psql' AS service_type, name FROM psql_service
    UNION ALL
    SELECT id, 'redis' AS service_type, name FROM redis_service
)
SELECT
    d.id,
    d.source_service_id,
    d.target_service_id,
    d.target_col,
    d.env_key,
    t.service_type AS target_service_type,
    t.name AS target_service_name,
    d.created_at,
    d.updated_at
FROM service_dependencies d
JOIN targets t ON t.id = d.target_service_id
WHERE d.source_service_id = ?
ORDER BY d.created_at DESC;

-- name: UpdateServiceDependency :one
UPDATE service_dependencies
SET target_service_id = ?, target_col = ?, env_key = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteServiceDependency :exec
DELETE FROM service_dependencies WHERE id = ?;

-- name: GetDependencyTargets :many
SELECT id, 'app' AS service_type, name FROM app_service
WHERE app_service.instance_id = ?1 AND app_service.id != ?2
UNION ALL
SELECT id, 'psql' AS service_type, name FROM psql_service
WHERE psql_service.instance_id = ?1
UNION ALL
SELECT id, 'redis' AS service_type, name FROM redis_service
WHERE redis_service.instance_id = ?1;

-- name: ResolveDependencyEnv :many
WITH dependency_targets AS (
    SELECT
        id,
        instance_id,
        'app' AS service_type,
        name,
        internal_url,
        domain,
        '' AS db_name,
        '' AS db_user,
        '' AS db_password,
        '' AS password
    FROM app_service

    UNION ALL

    SELECT
        id,
        instance_id,
        'psql' AS service_type,
        name,
        internal_url,
        '' AS domain,
        db_name,
        db_user,
        db_password,
        '' AS password
    FROM psql_service

    UNION ALL

    SELECT
        id,
        instance_id,
        'redis' AS service_type,
        name,
        internal_url,
        '' AS domain,
        '' AS db_name,
        '' AS db_user,
        '' AS db_password,
        password
    FROM redis_service
)
SELECT
    d.env_key,
    CASE d.target_col
        WHEN 'internal_url' THEN t.internal_url
        WHEN 'domain' THEN t.domain
        WHEN 'db_name' THEN t.db_name
        WHEN 'db_user' THEN t.db_user
        WHEN 'db_password' THEN t.db_password
        WHEN 'password' THEN t.password
        WHEN 'name' THEN t.name
    END AS resolved_value
FROM service_dependencies d
JOIN app_service source ON source.id = d.source_service_id
JOIN dependency_targets t ON t.id = d.target_service_id
WHERE d.source_service_id = ?
  AND t.instance_id = source.instance_id;

-- name: DeleteIncomingDependencies :exec
DELETE FROM service_dependencies WHERE target_service_id = ?;

-- name: GetDependenciesByTarget :many
SELECT * FROM service_dependencies WHERE target_service_id = ?;

-- name: GetDependencyGraphEdges :many
SELECT
    d.source_service_id,
    d.target_service_id,
    d.env_key,
    d.target_col
FROM service_dependencies d
JOIN app_service source ON source.id = d.source_service_id
WHERE source.instance_id = ?;

-- name: GetDependencyGraphNodes :many
SELECT id, name, 'app' as service_type FROM app_service AS a WHERE a.instance_id = ?
UNION ALL
SELECT id, name, 'psql' as service_type FROM psql_service AS p WHERE p.instance_id = ?
UNION ALL
SELECT id, name, 'redis' as service_type FROM redis_service AS r WHERE r.instance_id = ?;
