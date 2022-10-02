# OPA-RBAC

A simple RBAC service based on OPA and SQLite.

## Getting Started

The project includes a Makefile and can be build by simply running

```shell
make
```

Users, roles and permissions are stored using a SQLite database. A new RBAC
database can be bootstrapped by applying the 
[`migration.sql`](https://github.com/progbits/opa-rbac/blob/main/database/migration.sql)
schema. For example:

```shell
sqlite3 /tmp/opa-rbac.sqlite < database/migration.sql 
```

Once built, the server can be run using the newly created RBAC database:

```shell
OPA_RBAC_DATABASE_FILE=/tmp/opa-rbac.sqlite ./bin/opa-rbac
```

## Checking Permissions

The server exposes a single `v1/check` endpoint that takes a JSON body of the form:

```json
{
  "user_id": "...",
  "project": "...",
  "object": "...",
  "permission": "..."
}
```

The `v1/check` endpoint loads RBAC data from the configured database and
evaluates the decision, returning `200` on success and `403` on failure. 

### Example

The following fictional RBAC database describes a configuration with 2
projects (Foo and Bar), 2 users (John and Sally), 2 roles (account-manager
and finance-manager) and 2 permissions per role:

```sql
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
VALUES (1, 1, 1), -- John is an Account Manager for project Foo.
       (2, 2, 2); -- Sally is a Finance Manager for project Bar.

INSERT INTO rbac_permission(id, name)
VALUES (1, 'create:account'),
       (2, 'close:account'),
       (3, 'create:payment'),
       (4, 'delete:payment');

INSERT INTO rbac_role_permission(rbac_role_id, rbac_permission_id)
VALUES (1, 1), -- Account Managers can create:account.
       (1, 2), -- Account Managers can close:account.
       (2, 3), -- Finance Managers can create:payment.
       (2, 4); -- Finance Managers can delete:payment.
```

This example database can be found at [`bootstrap-simple.sql`](https://github.com/progbits/opa-rbac/blob/main/database/bootstrap-simple.sql)
and can be applied to the database created in [Getting Started](#getting-started):

```shell
sqlite3 /tmp/opa-rbac.sqlite < database/bootstrap-simple.sql
```

Once the RBAC database has been populated with the example schema, permission
checks can be evaluated.

John has permission to `create` an `account` in project `Foo`. 

```shell
curl -w "%{http_code}\n" localhost:8080/v1/check -H 'content-type:application/json' -d '{
    "user_id": "1",
    "project": "Foo", 
    "object": "account", 
    "permission": "create"
}'
200
```

John has permission to `close` an `account` in project `Foo`.

```shell
curl -w "%{http_code}\n" localhost:8080/v1/check -H 'content-type:application/json' -d '{
    "user_id": "1",
    "project": "Foo", 
    "object": "account", 
    "permission": "close"
}'
200
```

John __does not__ have permission to `close` an `account` in project `Bar`.

```shell
curl -w "%{http_code}\n" localhost:8080/v1/check -H 'content-type:application/json' -d '{
    "user_id": "1",
    "project": "Bar", 
    "object": "account", 
    "permission": "close"
}'
403
```

Sally has permission to `create` a `payment` in project `Bar`.
```shell
curl -w "%{http_code}\n" localhost:8080/v1/check -H 'content-type:application/json' -d '{
    "user_id": "2",
    "project": "Bar", 
    "object": "payment", 
    "permission": "create"
}'
200
```
