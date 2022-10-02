CREATE TABLE rbac_project (
    id   INTEGER PRIMARY KEY,
    name TEXT
);

CREATE TABLE rbac_user (
    id   INTEGER PRIMARY KEY,
    name TEXT
);

CREATE TABLE rbac_role (
    id   INTEGER PRIMARY KEY,
    name TEXT
);

CREATE TABLE rbac_project_user_role (
    rbac_project_id INTEGER,
    rbac_user_id    INTEGER,
    rbac_role_id    INTEGER
);

CREATE TABLE rbac_permission (
    id   INTEGER PRIMARY KEY,
    name TEXT
);

CREATE TABLE rbac_role_permission (
    rbac_role_id       INTEGER,
    rbac_permission_id INTEGER
);

CREATE VIEW IF NOT EXISTS rbac_data AS
    WITH
     user_project_role AS (
        SELECT rbac_user.id AS user_id,
               rbac_project.name AS project_name,
               json_group_array(rbac_role.name) AS roles
        FROM rbac_project_user_role
                 JOIN rbac_user ON rbac_user.id = rbac_project_user_role.rbac_user_id,
             rbac_role ON rbac_role.id = rbac_project_user_role.rbac_role_id,
             rbac_project ON rbac_project.id = rbac_project_user_role.rbac_project_id
        GROUP BY rbac_user.name, rbac_project.name),

     user_project_role_agg AS (
         SELECT user_project_role.user_id AS user_id,
                json_group_object(
                        user_project_role.project_name,
                        json(user_project_role.roles)) AS project_roles
         FROM user_project_role GROUP BY user_project_role.user_id),

     role_permissions AS (
         SELECT rbac_role.name AS name,
                json_group_array(rbac_permission.name) AS permissions
         FROM rbac_role_permission
                  JOIN rbac_role ON rbac_role_permission.rbac_role_id = rbac_role.id,
              rbac_permission ON rbac_permission.id = rbac_role_permission.rbac_permission_id
         GROUP BY rbac_role.name),

     roles_object AS (
         SELECT json_group_object(user_id, json(project_roles)) as object from user_project_role_agg),

     roles_permissions_object AS (
         SELECT json_group_object(name, json(permissions)) as object from role_permissions)

    SELECT json_object(
        'roles', json((SELECT object FROM roles_object)),
        'permissions', json((SELECT object from roles_permissions_object))
    ) AS data;
