# You can import a user identity to terraform state using `terraform import <resource> <id>`.
# The `id` must be a string for the id of the user and identity provider you want to import,
# for example:
terraform import gitlab_user_identity.example "42:google"
