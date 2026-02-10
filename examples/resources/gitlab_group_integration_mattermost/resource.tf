resource "gitlab_group" "example" {
  name        = "example-group"
  path        = "example-group"
  description = "An example group"
}

resource "gitlab_group_integration_mattermost" "mattermost" {
  group    = gitlab_group.example.id
  webhook  = "https://mattermost.example.com/hooks/..."
  username = "my-username"
}
