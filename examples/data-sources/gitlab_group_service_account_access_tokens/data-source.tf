data "gitlab_group" "example" {
  id = "foo/bar/baz"
}

# List all access tokens for a group service account
data "gitlab_group_service_account_access_tokens" "example" {
  group              = data.gitlab_group.example.id
  service_account_id = 123
}

# Output example: Get the names of all tokens
output "token_names" {
  value = [for token in data.gitlab_group_service_account_access_tokens.example.access_tokens : token.name]
}

# Output example: Get only active tokens
output "active_tokens" {
  value = [for token in data.gitlab_group_service_account_access_tokens.example.access_tokens : token if token.active]
}

