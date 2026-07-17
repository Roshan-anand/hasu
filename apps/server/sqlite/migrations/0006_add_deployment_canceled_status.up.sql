-- Add 'canceled' to the deployments status check constraint.
-- SQLite does not support ALTER TABLE to modify a CHECK constraint, so we
-- recreate the table and copy the existing rows.

PRAGMA foreign_keys = OFF;

CREATE TABLE deployments_new (
    id uuid PRIMARY KEY,
    is_current BOOLEAN NOT NULL,
    service_id uuid NOT NULL REFERENCES app_service(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('building','ready','error','queued','inactive','pruned','paused','canceled')),
    commit_hash TEXT NOT NULL,
    commit_msg TEXT NOT NULL,
    image TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO deployments_new
SELECT id, is_current, service_id, status, commit_hash, commit_msg, image, created_at
FROM deployments;

DROP TABLE deployments;

ALTER TABLE deployments_new RENAME TO deployments;

PRAGMA foreign_keys = ON;
