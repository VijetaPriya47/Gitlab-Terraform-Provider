# Basic example: Retrieve all project access tokens
data "gitlab_project_access_tokens" "all_tokens" {
  project = "my/example/project"
}

# Example with state filtering: Only retrieve active tokens
data "gitlab_project_access_tokens" "active_tokens" {
  project = "my/example/project"
  state   = "active"
}

# Example with state filtering: Only retrieve inactive tokens
data "gitlab_project_access_tokens" "inactive_tokens" {
  project = "my/example/project"
  state   = "inactive"
}

# Output examples showing different ways to work with the data

# Get the total count of tokens
output "token_count" {
  description = "Total number of project access tokens"
  value       = length(data.gitlab_project_access_tokens.all_tokens.access_tokens)
}

# Get only the names of active tokens
output "active_token_names" {
  description = "Names of active project access tokens"
  value       = [for token in data.gitlab_project_access_tokens.active_tokens.access_tokens : token.name]
}

# Get detailed information about each token
output "token_details" {
  description = "Detailed information about each access token"
  value = [
    for token in data.gitlab_project_access_tokens.all_tokens.access_tokens : {
      id           = token.id
      name         = token.name
      description  = token.description
      scopes       = token.scopes
      active       = token.active
      revoked      = token.revoked
      created_at   = token.created_at
      expires_at   = token.expires_at
      access_level = token.access_level
      user_id      = token.user_id
      last_used_at = token.last_used_at
    }
  ]
}

# Filter tokens programmatically to get only active and non-revoked tokens
output "active_non_revoked_tokens" {
  description = "Only active and non-revoked tokens"
  value = [
    for token in data.gitlab_project_access_tokens.all_tokens.access_tokens : token
    if token.active && !token.revoked
  ]
}

# Get tokens that expire within the next 30 days
output "expiring_soon_tokens" {
  description = "Tokens that expire within the next 30 days"
  value = [
    for token in data.gitlab_project_access_tokens.all_tokens.access_tokens : token
    if token.expires_at != "" && can(parseint(split("T", token.expires_at)[0], 10)) &&
    timeadd(timestamp(), "30d") > timeparse("2006-01-02", split("T", token.expires_at)[0])
  ]
}

# Get tokens with specific scopes
output "api_tokens" {
  description = "Tokens that have API scope"
  value = [
    for token in data.gitlab_project_access_tokens.all_tokens.access_tokens : token
    if contains(token.scopes, "api")
  ]
}

# Get tokens by access level
output "maintainer_tokens" {
  description = "Tokens with maintainer access level"
  value = [
    for token in data.gitlab_project_access_tokens.all_tokens.access_tokens : token
    if token.access_level == "maintainer"
  ]
}

# Get a summary of token statistics
output "token_statistics" {
  description = "Summary statistics of project access tokens"
  value = {
    total_tokens   = length(data.gitlab_project_access_tokens.all_tokens.access_tokens)
    active_tokens  = length([for token in data.gitlab_project_access_tokens.all_tokens.access_tokens : token if token.active])
    revoked_tokens = length([for token in data.gitlab_project_access_tokens.all_tokens.access_tokens : token if token.revoked])
    expired_tokens = length([for token in data.gitlab_project_access_tokens.all_tokens.access_tokens : token if token.expires_at != "" && can(parseint(split("T", token.expires_at)[0], 10)) && timeadd(timestamp(), "0d") > timeparse("2006-01-02", split("T", token.expires_at)[0])])
    tokens_by_level = {
      for level in distinct([for token in data.gitlab_project_access_tokens.all_tokens.access_tokens : token.access_level]) : level => length([for token in data.gitlab_project_access_tokens.all_tokens.access_tokens : token if token.access_level == level])
    }
  }
}
