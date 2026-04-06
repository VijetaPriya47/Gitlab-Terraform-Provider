# GitLab project job token scopes can be imported using an id made up of `projectId:type:targetId`, for example:

# For target_project_id:
terraform import gitlab_project_job_token_scope.bar 123:project:321

# For target_group_id:
terraform import gitlab_project_job_token_scope.bar 123:group:321
