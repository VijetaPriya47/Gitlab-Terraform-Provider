# Import using project ID
terraform import gitlab_project_pull_mirror.example 123

# Import using project path
# Note: Import is not supported for disabled mirrors because the GitLab API returns 
# http 400 for disabled mirrors.
terraform import gitlab_project_pull_mirror.example "group/project"
