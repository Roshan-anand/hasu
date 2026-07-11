-- name: CreateOrg :one
INSERT INTO organization (id, name)
VALUES (?, ?)
RETURNING id, name;

-- name: DeleteOrg :exec
DELETE FROM organization
WHERE id = ?;

-- name: GetAllOrg :many
SELECT o.id,o.name
FROM organization o
JOIN user_organization uo ON o.id = uo.organization_id
WHERE uo.user_email = ?;

-- name: GetCurrentOrg :one
SELECT o.id, o.name
FROM user u
JOIN organization o ON o.id = u.current_org_id
WHERE u.email = ?;

-- name: CountUserOrgs :one
SELECT COUNT(*) FROM organization;

-- name: UnlinkUserOrg :exec
DELETE FROM user_organization WHERE user_email = ? AND organization_id = ?;

-- name: GetOrgById :one
SELECT id, name FROM organization WHERE id = ?;

-- name: LinkUserNOrg :exec
INSERT INTO user_organization (user_email, organization_id)
VALUES (?, ?);

-- name: CheckUserOrgExists :one
SELECT CAST(EXISTS(
    SELECT 1 FROM user_organization uo
    WHERE uo.user_email = ? AND uo.organization_id = ?
)AS BOOLEAN);

-- name: RenameOrg :one
UPDATE organization
SET name = ?
WHERE id = ?
RETURNING id, name;

-- name: GetProjectsByOrgId :many
SELECT id, name, created_at
FROM project
WHERE organization_id = ?;

-- name: CreateProject :one
INSERT INTO project (id, organization_id, name)
VALUES (?, ?, ?)
RETURNING id, name, created_at;

-- name: GetProjectIDByName :one
SELECT id
FROM project
WHERE organization_id = ? AND name = ?;

-- name: GetAllProjects :many
SELECT id, name, created_at
FROM project
WHERE organization_id = ?;

-- name: CheckProjectHasInstance :one
SELECT CAST(EXISTS(
    SELECT 1
    FROM instance
    WHERE project_id = ?
) AS BOOLEAN);

-- name: TransferProject :exec
UPDATE project
SET organization_id = ?
WHERE id = ?;

-- name: DeleteProject :exec
DELETE FROM project
WHERE id = ?;

-- name: CreateInstance :exec
INSERT INTO instance (id, project_id, is_production, name, network, status, git_source_type, git_source_value, created_by)
VALUES (?, ?, ?, ?, ?, ?, 'branch', '', 'manual');

-- name: GetAllInstance :many
SELECT i.id, i.name, i.is_production
FROM instance i
JOIN project p ON i.project_id = p.id
WHERE p.organization_id = ? AND p.name = @project;

-- name: GetInstanceStatus :one
SELECT status
FROM instance
WHERE id = @instance_id;

-- name: CheckInstanceHasServices :one
SELECT CAST(EXISTS(
    SELECT 1
    FROM app_service aps
    WHERE aps.instance_id = @instance_id
    UNION ALL
    SELECT 1
    FROM psql_service ps
    where ps.instance_id = @instance_id
    UNION ALL
    SELECT 1
    FROM redis_service rs
    where rs.instance_id = @instance_id
) AS BOOLEAN);

-- name: DeleteInstance :exec
DELETE FROM instance
WHERE id = ?;

-- name: GetInstanceNetwork :one
SELECT network
FROM instance
WHERE id = @instance_id;

-- name: GetAllNetworksByProjectId :many
SELECT network
FROM instance
WHERE project_id = ?;

-- name: RenameInstance :one
UPDATE instance
SET name = ?
WHERE id = ?
RETURNING id, name, is_production;

-- name: GetAllInstancesByOrgId :many
SELECT i.id, i.name, i.is_production, i.project_id, p.name AS project_name,
    CAST((
        SELECT COUNT(*)
        FROM (
            SELECT 1 FROM app_service aps WHERE aps.instance_id = i.id
            UNION ALL
            SELECT 1 FROM psql_service ps WHERE ps.instance_id = i.id
            UNION ALL
            SELECT 1 FROM redis_service rs WHERE rs.instance_id = i.id
        )
    ) AS INTEGER) AS service_count
FROM instance i
JOIN project p ON p.id = i.project_id
WHERE p.organization_id = ?;

-- name: RenameProject :one
UPDATE project
SET name = ?
WHERE id = ?
RETURNING id, name, created_at;
