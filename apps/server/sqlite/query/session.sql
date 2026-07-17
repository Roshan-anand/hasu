-- name: CreateSession :exec
INSERT INTO session (id, user_id, token, expires_at)
VALUES (?, ?, ?, ?);

-- name: GetSessionByToken :one
SELECT u.id,u.email,u.name,u.role,o.id AS org_id,o.name AS org_name,s.expires_at,s.created_at
FROM session s
JOIN user u ON s.user_id = u.id
JOIN organization o ON o.id = u.current_org_id
WHERE s.token = ?;

-- name: RemoveSessionByUID :exec
DELETE FROM session
WHERE user_id = ?;