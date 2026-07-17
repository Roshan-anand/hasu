-- name: CreateRedirectSession :exec
INSERT INTO redirect_session (state, user_id, org_id, expires_at)
VALUES  (?, ?, ?, ?);

-- name: GetRedirectSession :one
SELECT state, user_id, org_id, gh_app_id, expires_at, created_at
FROM redirect_session
WHERE state = ?;

-- name: GetRedirectSessionGhAppID :one
SELECT gh_app_id
FROM redirect_session
WHERE state = ?;

-- name: UpdateRedirectSession :exec
UPDATE redirect_session
SET gh_app_id = ?
WHERE state = ?;

-- name: DeleteRedirectSession :exec
DELETE FROM redirect_session
WHERE state = ?;

-- name: CreateGithubApp :one
INSERT INTO github_app (id, name, organization_id,app_id, pem_key, webhook_secret)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING app_id;

-- name: GetGhAppByAppId :one
SELECT * FROM github_app
WHERE app_id = ?;

-- name: GetAllGhAppsByEmail :many
SELECT gh.name, gh.app_id, gh.created_at
FROM user u
JOIN github_app gh ON u.current_org_id = gh.organization_id
WHERE u.email = ?;

-- name: InsertInstallationID :exec
UPDATE github_app
SET installation_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE app_id = ?;

-- name: DeleteGithubApp :exec
DELETE FROM github_app
WHERE app_id = ?;

-- name: UpsertPullRequest :exec
INSERT INTO github_pull_requests (id, repo_id, pr_number, title, head_branch, base_branch, state, html_url)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(repo_id, pr_number) DO UPDATE SET
    title = excluded.title,
    head_branch = excluded.head_branch,
    base_branch = excluded.base_branch,
    state = excluded.state,
    html_url = excluded.html_url,
    updated_at = CURRENT_TIMESTAMP;

-- name: GetPullRequestsByInstance :many
SELECT DISTINCT pr.id, pr.repo_id, pr.pr_number, pr.title, pr.head_branch, pr.base_branch, pr.state, pr.html_url, pr.created_at, pr.updated_at
FROM github_pull_requests pr
JOIN app_service a ON a.gh_repo_id = pr.repo_id
WHERE a.instance_id = ? AND pr.state = 'open'
ORDER BY pr.updated_at DESC;

-- name: GetPullRequestByRepoAndNumber :one
SELECT id, repo_id, pr_number, title, head_branch, base_branch, state, html_url, created_at, updated_at
FROM github_pull_requests
WHERE repo_id = ? AND pr_number = ?;

-- name: DeletePullRequest :exec
DELETE FROM github_pull_requests
WHERE repo_id = ? AND pr_number = ?;
