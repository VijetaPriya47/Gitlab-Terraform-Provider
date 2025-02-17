resource "gitlab_cluster_agent" "example" {
  project = "12345"
  name    = "agent-1"
}

// Optionally, configure the agent as described in
// https://docs.gitlab.com/user/clusters/agent/install/index/#create-an-agent-configuration-file
resource "gitlab_repository_file" "example_agent_config" {
  project   = gitlab_cluster_agent.example.project
  branch    = "main" // or use the `default_branch` attribute from a project data source / resource
  file_path = ".gitlab/agents/${gitlab_cluster_agent.example.name}/config.yaml"
  content = base64encode(<<CONTENT
# the GitLab Agent for Kubernetes configuration goes here ...
  CONTENT
  )
  author_email   = "terraform@example.com"
  author_name    = "Terraform"
  commit_message = "feature: add agent config for ${gitlab_cluster_agent.example.name} [skip ci]"
}
