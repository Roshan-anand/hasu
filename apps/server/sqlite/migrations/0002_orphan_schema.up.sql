CREATE TABLE IF NOT EXISTS orphan_volume (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL DEFAULT '',
    volume TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('psql','redis','mongodb')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);
