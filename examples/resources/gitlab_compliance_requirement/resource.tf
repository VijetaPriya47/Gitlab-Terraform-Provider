# Example: Create a compliance requirement with an internal control
resource "gitlab_compliance_requirement" "internal_control" {
  framework_id = gitlab_compliance_framework.example.framework_id
  name         = "Dependency Scanning Required"
  description  = "Ensures dependency scanning is enabled and running"

  controls {
    name         = "scanner_dep_scanning_running"
    control_type = "internal"

    expression {
      field    = "scanner_dep_scanning_running"
      operator = "equals"
      value    = "true"
    }
  }
}

# Example: Create a compliance requirement with an external control
resource "gitlab_compliance_requirement" "external_control" {
  framework_id = gitlab_compliance_framework.example.framework_id
  name         = "External Audit Verification"
  description  = "Verification via external audit service"

  controls {
    name         = "External Audit Report"
    control_type = "external"
    external_url = "https://example.com/audit-report"
    secret_token = var.audit_secret_token # Use a variable for sensitive values
  }
}

# Example: Create a compliance requirement with multiple controls
resource "gitlab_compliance_requirement" "multiple_controls" {
  framework_id = gitlab_compliance_framework.example.framework_id
  name         = "Comprehensive Security Check"
  description  = "Multiple security controls for compliance"

  controls {
    name         = "scanner_dep_scanning_running"
    control_type = "internal"

    expression {
      field    = "scanner_dep_scanning_running"
      operator = "equals"
      value    = "true"
    }
  }

  controls {
    name         = "External Security Audit"
    control_type = "external"
    external_url = "https://example.com/security-audit"
  }
}

