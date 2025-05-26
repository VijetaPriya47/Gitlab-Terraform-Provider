resource "gitlab_group" "example" {
  name        = "test_group"
  path        = "test_group"
  description = "An example group"
}

// Basic example
resource "gitlab_group_issue_board" "issue_board" {
  group = gitlab_group.example.id
  name  = "issue board 6"
}

resource "gitlab_group_label" "label" {
  group = gitlab_group.example.id
  name  = "label-1"
  color = "#FF0000"
}

// Example with all optional EE attributes
resource "gitlab_group_issue_board" "issue_board" {
  group        = gitlab_group.example.id
  name         = "issue board 6"
  milestone_id = 4
  labels       = [gitlab_group_label.label.label_id]
}
