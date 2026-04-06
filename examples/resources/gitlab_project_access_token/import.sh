# A GitLab Project Access Token can be imported using a key composed of `<project>:<token-id>`, for example:
terraform import gitlab_project_access_token.example "12345:1"
# Where `project` may be the product ID or path with namespace depending on what you have in your config.
# NOTE: the `token` resource attribute is not available for imported resources as this information cannot be read from the GitLab API.
