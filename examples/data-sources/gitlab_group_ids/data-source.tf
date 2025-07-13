resource "gitlab_group" "new_group" {
  name        = "example-group"
  path        = "example-path"
  description = "This is an example group"
}

// use group IDs to get additional information, such as the GraphQL ID
// for other resources
data "gitlab_group_ids" "foo" {
  group = "gitlab_group.new_group.id"
}

output "graphQL_id" {
  value = data.gitlab_group_ids.foo.group_graphql_id
}
