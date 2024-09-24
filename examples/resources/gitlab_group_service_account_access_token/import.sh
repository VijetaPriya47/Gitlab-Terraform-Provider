# You can import a service account access token using `terraform import <resource> <id>`.  The
# `id` is in the form of <group_id>:<service_account_id>:<access_token_id>
# Importing an access token does not import the access token value.
terraform import gitlab_group_service_account_access_token.example 1:2:3
