# GitLab compliance requirements can be imported using an id made up of `<framework_id>|<requirement_id>`
# Both IDs are GraphQL global IDs (GIDs) from GitLab. The pipe separator is used because GIDs contain colons.
terraform import gitlab_compliance_requirement.example "gid://gitlab/ComplianceManagement::Framework/123|gid://gitlab/ComplianceManagement::ComplianceRequirement/456"
