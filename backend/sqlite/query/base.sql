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

-- name: CheckOrgExists :one
SELECT CAST(EXISTS(
    SELECT 1 FROM organization o
    JOIN user_organization uo ON o.id = uo.organization_id
    WHERE uo.user_email = ? AND o.name = @org_name
) AS BOOLEAN);

-- name: CreateProject :one
INSERT INTO project (id, organization_id, name)
VALUES (?, ?, ?)
RETURNING id, name, created_at;

-- name: GetAllProjects :many
SELECT id, name, created_at
FROM project
WHERE organization_id = ?;

-- name: CheckProjectExists :one
SELECT CAST(EXISTS(
    SELECT 1
    FROM project
    WHERE organization_id = ? AND name = @project_name
) AS BOOLEAN);

-- name: CheckProjectHasInstance :one
SELECT CAST(EXISTS(
    SELECT 1
    FROM instance
    WHERE project_id = ?
) AS BOOLEAN);

-- name: DeleteProject :exec
DELETE FROM project
WHERE id = ?;

-- name: CreateInstance :exec
INSERT INTO instance (id, project_id, is_production, name, network)
VALUES (?, ?, ?, ?, ?);

-- name: GetAllInstance :many
SELECT i.id, i.name, i.is_production
FROM instance i
JOIN project p ON i.project_id = p.id
WHERE p.organization_id = ? AND p.name = @project;

-- name: CheckInstanceExists :one
SELECT CAST(EXISTS(
    SELECT 1
    FROM instance
    WHERE project_id = ? AND name = @instance_name
) AS BOOLEAN);

-- name: CheckInstanceHasServices :one
SELECT CAST(EXISTS(
    SELECT 1
    FROM app_service aps
    WHERE aps.instance_id = @instance_id
    UNION ALL
    SELECT 1
    FROM psql_service ps
    where ps.instance_id = @instance_id
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
