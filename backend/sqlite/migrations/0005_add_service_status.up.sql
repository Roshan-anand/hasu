ALTER TABLE psql_service ADD COLUMN status TEXT NOT NULL DEFAULT 'running';
ALTER TABLE redis_service ADD COLUMN status TEXT NOT NULL DEFAULT 'running';
