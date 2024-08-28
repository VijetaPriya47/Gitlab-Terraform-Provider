variable "admin_token" {
  description = "Owner/Maintainer PAT token with the api scope applied."
  type        = string
}

variable "base_uri" {
  description = "The HTTP base-URI for your GitLab instance."
  type        = string
  default     = "https://gitlab.com"
}

variable "root_group_id" {
  description = "The GitLab ID number for your root group namespace."
  type        = string
}

variable "team_lead_user" {
  description = "The team lead's GitLab user name."
  type        = string
}

variable "team_member_users" {
  description = "The GitLab user names for your team members."
  type        = list(string)
}

variable "license_type" {
  description = "The type of license (free, premium or ultimate) associated with your top-level group/admin token."
  type        = string
  default     = "free"
  validation {
    error_message = "License has to be set to ce, free, premium or ultimate."
    condition     = contains(["free", "premium", "ultimate"], var.license_type)
  }
}
