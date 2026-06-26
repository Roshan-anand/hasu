CREATE TABLE IF NOT EXISTS redis_service (
    id uuid PRIMARY KEY,
    instance_id uuid NOT NULL REFERENCES instance(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK(status IN ('running','paused')),
    type TEXT NOT NULL CHECK(type IN ('redis')),
    name TEXT NOT NULL,
    swarm_service TEXT NOT NULL,
    password TEXT NOT NULL,
    image TEXT NOT NULL,
    volume TEXT NOT NULL,
    internal_url TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);
