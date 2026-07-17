CREATE TABLE IF NOT EXISTS service_dependencies (
    id uuid PRIMARY KEY,
    source_service_id uuid NOT NULL REFERENCES app_service(id) ON DELETE CASCADE,
    target_service_id uuid NOT NULL,
    target_col TEXT NOT NULL CHECK (
        target_col IN (
            'internal_url',
            'domain',
            'db_name',
            'db_user',
            'db_password',
            'password',
            'name'
        )
    ),
    env_key TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE(source_service_id, env_key)
);
