package api

// GraphQLComplianceRequirement represents a compliance requirement from the GitLab GraphQL API.
//
// GitLab API docs: https://docs.gitlab.com/api/graphql/reference/#compliancerequirement
type GraphQLComplianceRequirement struct {
	ID                    string                     `json:"id"`
	Name                  string                     `json:"name"`
	Description           string                     `json:"description"`
	ComplianceFrameworkID string                     `json:"complianceFrameworkId,omitempty"`
	Controls              []GraphQLComplianceControl `json:"complianceRequirementsControls,omitempty"`
}

// GraphQLComplianceControl represents a control within a compliance requirement.
//
// GitLab API docs: https://docs.gitlab.com/api/graphql/reference/#compliancerequirementscontrol
type GraphQLComplianceControl struct {
	Name        string                    `json:"name"`
	ControlType string                    `json:"controlType"`
	Expression  *GraphQLControlExpression `json:"expression,omitempty"`
	ExternalURL string                    `json:"externalUrl,omitempty"`
	SecretToken string                    `json:"secretToken,omitempty"`
}

// GraphQLControlExpression represents the expression for an internal control.
type GraphQLControlExpression struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// ValidComplianceControlTypes contains the valid control types for compliance controls.
var ValidComplianceControlTypes = []string{
	"internal",
	"external",
}

// ValidComplianceControlOperators contains the valid operators for internal control expressions.
var ValidComplianceControlOperators = []string{
	"equals",
	"not_equals",
	"greater_than",
	"less_than",
}
