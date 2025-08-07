# GitLab project deploy tokens can be imported using an id made up of `{project_id}:{deploy_token_id}`.
terraform import gitlab_project_deploy_token.project_token 1:4

# Note: the `token` resource attribute is not available for imported resources as this information cannot be read from the GitLab API.
