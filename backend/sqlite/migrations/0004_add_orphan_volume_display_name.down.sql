-- SQLite doesn't support DROP COLUMN directly; recreate the table without the column
CREATE TABLE IF NOT EXISTS orphan_volume_new (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    volume TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO orphan_volume_new (id, organization_id, volume, type, created_at)
SELECT id, organization_id, volume, type, created_at FROM orphan_volume;

DROP TABLE orphan_volume;

ALTER TABLE orphan_volume_new RENAME TO orphan_volume;
