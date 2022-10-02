INSERT INTO rbac_project(id, name)
VALUES (1, 'Foo'),
       (2, 'Bar');

INSERT INTO rbac_user(id, name)
VALUES (1, 'John'),
       (2, 'Sally');

INSERT INTO rbac_role(id, name)
VALUES (1, 'account-manager'),
       (2, 'finance-manager');

INSERT INTO rbac_project_user_role(rbac_project_id, rbac_user_id, rbac_role_id)
VALUES (1, 1, 1), -- John is an Account Manager for project Foo
       (2, 2, 2); -- Sally is a Finance Manager for project Bar

INSERT INTO rbac_permission(id, name)
VALUES (1, 'create:account'),
       (2, 'close:account'),
       (3, 'create:payment'),
       (4, 'delete:payment');

INSERT INTO rbac_role_permission(rbac_role_id, rbac_permission_id)
VALUES (1, 1), -- account-manager - create:account
       (1, 2), -- account-manager - close:account
       (2, 3), -- finance-staff - create:payment
       (2, 4); -- finance-manager - delete:payment
