resource "gitlab_compliance_framework" "alpha" {
  namespace_path = "top-level-group"
  name           = "HIPAA"
  description    = "A HIPAA Compliance Framework"
  color          = "#87BEEF"
  default        = false
}

resource "gitlab_compliance_framework" "beta" {
  namespace_path = "top-level-group"
  name           = "SOC"
  description    = "A SOC Compliance Framework"
  color          = "#223344"
  default        = false
}

resource "gitlab_project_compliance_frameworks" "sample" {
  compliance_framework_ids = [gitlab_compliance_framework.alpha.framework_id, gitlab_compliance_framework.beta.framework_id]
  project                  = "12345678"
}
