resource "gitlab_project" "new_project" {
  // include required attributes
}

// use project IDs to get additional information, such as the GraphQL ID
// for other resources
data "gitlab_project_ids" "foo" {
  project = "gitlab_project.new_project.id"
}

output "graphQL_id" {
  value = data.gitlab_project_ids.foo.project_graphql_id
}
