# GitLab project external status checks can be imported using an id made up of `<project-id>:<external-check-id>`, e.g.
terraform import gitlab_project_external_status_check.foo "123:42"

# NOTE: the `shared_secret` resource attribute is not available for imported resources as this information cannot be read from the GitLab API.
