resource "gitlab_group" "example" {
  name        = "test_group"
  path        = "test_group"
  description = "An example group"
}

resource "gitlab_group_label" "label_1" {
  group = gitlab_group.example.id
  color = "#FF0000"
  name  = "red-label"
}

resource "gitlab_group_label" "label_3" {
  group = gitlab_group.example.id
  name  = "label-3"
  color = "#003000"
}

resource "gitlab_group_epic_board" "epic_board" {
  name  = "epic board 6"
  group = gitlab_group.example.path
  lists {
    label_id = gitlab_group_label.label_1.label_id
  }
}
