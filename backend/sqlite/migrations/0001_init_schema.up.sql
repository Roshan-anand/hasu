CREATE TABLE organization (
    id uuid PRIMARY KEY,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE user (
    id uuid PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    hash_pass TEXT NOT NULL,
    role TEXT NOT NULL,
    current_org_id uuid NOT NULL REFERENCES organization(id) ON DELETE RESTRICT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE user_organization (
    user_email TEXT NOT NULL REFERENCES user(email) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_email, organization_id)
);

CREATE TABLE session (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at DATETIME NOT NULL
);

CREATE TABLE project (
    id uuid PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE psql_service (
    id uuid PRIMARY KEY,
    project_id uuid NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    service_id TEXT NOT NULL REFERENCES service(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    app_name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    db_name TEXT NOT NULL,
    db_user TEXT NOT NULL,
    db_password TEXT NOT NULL,
    image TEXT NOT NULL,
    internal_url TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE app_service (
    id uuid PRIMARY KEY,
    project_id uuid NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    app_name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    git_provider TEXT NOT NULL,
    git_repo_id TEXT NOT NULL,
    git_repo_name TEXT NOT NULL,
    git_branch TEXT NOT NULL,
    build_path TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE deployments (
    id uuid PRIMARY KEY,
    service_id uuid NOT NULL REFERENCES app_service(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE redirect_session (
    state TEXT PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    org_id uuid NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    gh_app_id INTEGER REFERENCES github_app(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE github_app (
    id uuid PRIMARY KEY,
    name TEXT NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    app_id INTEGER NOT NULL,
    installation_id INTEGER,
    pem_key TEXT NOT NULL,
    webhook_secret TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);
