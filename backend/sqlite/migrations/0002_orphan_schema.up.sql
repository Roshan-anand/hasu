CREATE TABLE IF NOT EXISTS orphan_volume (
    id uuid PRIMARY KEY,
    project_id uuid REFERENCES project(id),
    volume TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);
