CREATE TABLE rbac_project (
    id   INTEGER PRIMARY KEY,
    name TEXT
);

CREATE TABLE rbac_user (
    id INTEGER PRIMARY KEY
);

CREATE TABLE rbac_role (
    id        INTEGER PRIMARY KEY,
    role_name TEXT
);

CREATE TABLE rbac_project_user_role (
    rbac_project_id INTEGER,
    rbac_user_id    INTEGER,
    rbac_role_id    INTEGER
);

CREATE TABLE rbac_permission (
    id              INTEGER PRIMARY KEY,
    permission_name TEXT
);

CREATE TABLE rbac_role_permission (
    rbac_role_id       INTEGER,
    rbac_permission_id INTEGER
);
