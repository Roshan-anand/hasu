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
INSERT INTO project (id, organization_id, name, network_name)
VALUES (?, ?, ?, ?)
RETURNING id, name;

-- name: GetAllProjects :many
SELECT *
FROM project
WHERE organization_id = ?;

-- name: CheckProjectExists :one
SELECT CAST(EXISTS(
    SELECT 1
    FROM project
    WHERE organization_id = ? AND name = @project_name
) AS BOOLEAN);

-- name: DeleteProject :exec
DELETE FROM project
WHERE id = ?;

-- name: CheckProjectHasServices :one
SELECT CAST(EXISTS(
    SELECT 1
    FROM app_service aps
    WHERE aps.project_id = @project_id
    UNION ALL
    SELECT 1
    FROM psql_service ps
    WHERE ps.project_id = @project_id
) AS BOOLEAN);
