-- name: CreateUser :one
INSERT INTO user (id, name, email, hash_pass, role, current_org_id)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: GetUserByEmail :one
SELECT
   u.id,
   u.name,
   u.email,
   u.hash_pass,
   u.role,
   o.id AS org_id,
   o.name AS org_name
FROM user u
JOIN organization o ON o.id = u.current_org_id
WHERE u.email = ?;

-- name: GetUserCurrentOrg :one
SELECT u.current_org_id
FROM user u
WHERE u.email = ?;

-- name: IsUserAdmin :one
SELECT CAST(EXISTS (
    SELECT 1 FROM user
    WHERE email = ? AND role = 'admin'
) AS BOOLEAN);

-- name: AdminExists :one
SELECT CAST(EXISTS (
    SELECT 1 FROM user
    WHERE role = 'admin'
) AS BOOLEAN);

-- name: UpdateCurrentOrg :exec
UPDATE user
SET current_org_id = ?
WHERE id = ?;

-- name: GetUserProfile :one
SELECT id, name, email, role, avatar, created_at
FROM user
WHERE id = ?;

-- name: UpdateUserProfile :exec
UPDATE user
SET name = ?, email = ?, avatar = ?
WHERE id = ?;

-- name: UpdateUserPassword :exec
UPDATE user
SET hash_pass = ?
WHERE id = ?;
