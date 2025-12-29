# Configure the project package dependency proxy for Maven packages
resource "gitlab_project_package_dependency_proxy" "example" {
  project = gitlab_project.example.id

  enabled                     = true
  maven_external_registry_url = "https://repo.maven.apache.org/maven2/"
}

# With authentication credentials for the external registry
resource "gitlab_project_package_dependency_proxy" "authenticated" {
  project = gitlab_project.example.id

  enabled                          = true
  maven_external_registry_url      = "https://private-repo.example.com/maven/"
  maven_external_registry_username = "maven_user"
  maven_external_registry_password = var.maven_registry_password
}

