# Example: Create a compliance requirement with an internal control
resource "gitlab_compliance_requirement" "internal_control" {
  framework_id = gitlab_compliance_framework.example.framework_id
  name         = "Dependency Scanning Required"
  description  = "Ensures dependency scanning is enabled and running"

  controls = [{
    name         = "scanner_dep_scanning_running"
    control_type = "internal"

    expression = {
      field    = "scanner_dep_scanning_running"
      operator = "="
      value    = "true"
    }
  }]
}
