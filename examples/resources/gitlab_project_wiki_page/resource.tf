resource "gitlab_project_wiki_page" "example" {
  project = var.ProjectID    # Specify your project path or ID
  slug    = "test-wiki-page" # Unique slug for the wiki page
  title   = "Test Wiki Page"
  content = <<EOF
This is a test content for the wiki page.
And this is a second line of content.
EOF
}