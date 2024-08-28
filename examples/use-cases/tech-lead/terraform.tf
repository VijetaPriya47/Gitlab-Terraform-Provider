terraform {
  required_providers {
    gitlab = {
      source = "gitlabhq/gitlab"
    }
  }
}

provider "gitlab" {
  token    = var.admin_token
  base_url = var.base_uri
}
