data "gitlab_project_secure_file" "by_id" {
  project        = "123"
  secure_file_id = 123
}

data "gitlab_project_secure_file" "by_name" {
  project = "123"
  name    = "secret.pem"
}
