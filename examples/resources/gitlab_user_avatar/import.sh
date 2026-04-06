# A GitLab User Avatar can be imported using the user id, for example:
terraform import gitlab_user_avatar.example "12345"

# NOTE: the `token` and `avatar` resource attributes are not available for imported resources as this information cannot be read from the GitLab API.
