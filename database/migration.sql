CREATE TABLE rbac_user (
    id INTEGER
);

CREATE TABLE rbac_role (
    id        INTEGER,
    role_name TEXT
);

CREATE TABLE rbac_user_role (
    rbac_user_id INTEGER,
    rbac_role_id INTEGER
);

CREATE TABLE rbac_permission (
    id              INTEGER,
    permission_name TEXT
);

CREATE TABLE rbac_role_permission (
    rbac_role_id       INTEGER,
    rbac_permission_id INTEGER
);
