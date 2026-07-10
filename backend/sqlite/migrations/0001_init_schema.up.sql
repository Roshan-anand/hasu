CREATE TABLE IF NOT EXISTS organization (
    id uuid PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS user (
    id uuid PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    hash_pass TEXT NOT NULL,
    role TEXT NOT NULL,
    avatar TEXT,
    current_org_id uuid NOT NULL REFERENCES organization(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS user_organization (
    user_email TEXT NOT NULL REFERENCES user(email) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_email, organization_id)
);

CREATE TABLE IF NOT EXISTS session (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS project(
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (organization_id, name)
);

CREATE TABLE IF NOT EXISTS instance(
    id uuid PRIMARY KEY,
    project_id uuid NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    is_production BOOLEAN NOT NULL,
    name TEXT NOT NULL,
    network TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    git_source_type TEXT CHECK(git_source_type IN ('pr','branch')),
    git_source_value TEXT,
    status TEXT NOT NULL CHECK(status IN ('creating','ready','deleting')),
    created_by TEXT DEFAULT 'manual' CHECK(created_by IN ('manual','webhook')),
    UNIQUE (project_id, name)
);

CREATE TABLE IF NOT EXISTS app_service (
    id uuid PRIMARY KEY,
    instance_id uuid NOT NULL REFERENCES instance(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    git_provider TEXT NOT NULL,
    gh_app_id INTEGER NOT NULL REFERENCES github_app(app_id) ON DELETE SET NULL,
    gh_repo_id INTEGER NOT NULL,
    gh_repo_name TEXT NOT NULL,
    gh_repo_url TEXT NOT NULL,
    build_path TEXT NOT NULL,
    watch_path TEXT NOT NULL,
    -- docker realted column
    docker_filepath TEXT NOT NULL DEFAULT 'Dockerfile',
    docker_contextpath TEXT NOT NULL DEFAULT '.',
    docker_buildstage TEXT NOT NULL DEFAULT '',
    -- environment related column
    env BLOB,
    build_secrets BLOB,
    -- swarm related column
    is_public BOOLEAN NOT NULL,
    branch TEXT NOT NULL,
    swarm_service TEXT NOT NULL,
    domain TEXT,
    internal_url TEXT NOT NULL,
    port INTEGER NOT NULL DEFAULT 80,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS deployments (
    id uuid PRIMARY KEY,
    is_current BOOLEAN NOT NULL,
    service_id uuid NOT NULL REFERENCES app_service(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('building','ready','error','queued','inactive','pruned','paused')),
    commit_hash TEXT NOT NULL,
    commit_msg TEXT NOT NULL,
    image TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS psql_service (
    id uuid PRIMARY KEY,
    instance_id uuid NOT NULL REFERENCES instance(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    swarm_service TEXT NOT NULL,
    db_name TEXT NOT NULL,
    db_user TEXT NOT NULL,
    db_password TEXT NOT NULL,
    image TEXT NOT NULL,
    volume TEXT NOT NULL,
    internal_url TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS redirect_session (
    state TEXT PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    org_id uuid NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    gh_app_id INTEGER REFERENCES github_app(app_id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS github_app (
    id uuid NOT NULL,
    name TEXT NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    app_id INTEGER NOT NULL PRIMARY KEY,
    installation_id INTEGER,
    pem_key TEXT NOT NULL,
    webhook_secret TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);
