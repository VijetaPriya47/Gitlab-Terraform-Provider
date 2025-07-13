resource "gitlab_group" "this" {
  name        = "example"
  path        = "example"
  description = "An example group"
}
resource "gitlab_project" "this" {
  name                   = "example"
  namespace_id           = gitlab_group.this.id
  initialize_with_readme = true
}
resource "gitlab_repository_file" "this" {
  project        = gitlab_project.this.id
  file_path      = "meow.txt"
  branch         = "main"
  encoding       = "base64"
  content        = base64encode("Meow goes the cat")
  author_email   = "terraform@example.com"
  author_name    = "Terraform"
  commit_message = "feature: add meow file"
}

resource "gitlab_repository_file" "readme" {
  project   = gitlab_project.this.id
  file_path = "readme.txt"
  branch    = "main"
  encoding  = "text"
  // content will be auto base64 encoded
  content        = "Meow goes the cat"
  author_email   = "terraform@example.com"
  author_name    = "Terraform"
  commit_message = "feature: add readme file"
}

resource "gitlab_repository_file" "readme_for_dogs" {
  project        = gitlab_project.this.id
  file_path      = "readme.txt"
  branch         = "main"
  encoding       = "text"
  content        = "Bark goes the dog"
  author_email   = "terraform@example.com"
  author_name    = "Terraform"
  commit_message = "feature: update readme file"
  // file will be overwritten if it already exists and added to state
  overwrite_on_create = true
}
