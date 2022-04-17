package example.rbac

default allow = false

allow {
    # Look up the list of projects the user has access too.
    project_roles := data.roles[input.user_id]

    # For each of the roles held by the user for the named project.
    project_role := project_roles[input.project]
    pr := project_role[_]
    
    # Lookup the permissions for the roles.
    permissions := data.permissions[pr]

    # For each role permission, check if there is a match.
    p := permissions[_]
    p == concat("", [input.permission, ":", input.object])
}
