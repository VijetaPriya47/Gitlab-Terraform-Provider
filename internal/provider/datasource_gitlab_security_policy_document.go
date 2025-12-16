package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

var (
	_ datasource.DataSource              = &gitlabSecurityPolicyDocumentDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabSecurityPolicyDocumentDataSource{}
)

var (
	scanExecutionRuleTypes = []string{
		"pipeline",
		"schedule",
		"agent",
	}

	scanExecutionBranchTypes = []string{
		"all",
		"protected",
		"default",
	}

	scanExecutionScanTypes = []string{
		"sast",
		"secret_detection",
		"container_scanning",
		"dependency_scanning",
		"dast",
		"sast_iac",
		"cluster_image_scanning",
		"api_fuzzing",
		"coverage_fuzzing",
	}
)

func init() {
	registerDataSource(NewGitLabSecurityPolicyDocumentDataSource)
}

// NewGitLabSecurityPolicyDocumentDataSource is a helper function to simplify the provider implementation.
func NewGitLabSecurityPolicyDocumentDataSource() datasource.DataSource {
	return &gitlabSecurityPolicyDocumentDataSource{}
}

type gitlabSecurityPolicyDocumentDataSource struct {
	// no client needed - pure transformation data source
}

// gitlabSecurityPolicyDocumentDataSourceModel describes the data source data model
type gitlabSecurityPolicyDocumentDataSourceModel struct {
	// Computed outputs
	Id   types.String `tfsdk:"id"`
	Yaml types.String `tfsdk:"yaml"`

	// Input: Scan Execution, MR Approval, Pipeline Execution & Vulnerability Management (stored as slices)
	ScanExecutionPolicies []ScanExecutionPolicyModel `tfsdk:"scan_execution_policy"`
}

type ScanExecutionPolicyModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`

	Rules       []ScanExecutionRuleModel   `tfsdk:"rules"`
	Actions     []ScanExecutionActionModel `tfsdk:"actions"`
	PolicyScope *PolicyScopeModel          `tfsdk:"policy_scope"`
	SkipCi      *SkipCiModel               `tfsdk:"skip_ci"`
}

type ScanExecutionRuleModel struct {
	Type             types.String   `tfsdk:"type"`
	Branches         []types.String `tfsdk:"branches"`
	BranchType       types.String   `tfsdk:"branch_type"`
	BranchExceptions []types.String `tfsdk:"branch_exceptions"`

	// schedule-specific
	Cadence types.String `tfsdk:"cadence"`

	// agent-specific
	Agents types.Map `tfsdk:"agents"`
}

type ScanExecutionActionModel struct {
	Scan           types.String   `tfsdk:"scan"`
	Template       types.String   `tfsdk:"template"`
	SiteProfile    types.String   `tfsdk:"site_profile"`
	ScannerProfile types.String   `tfsdk:"scanner_profile"`
	Variables      types.Map      `tfsdk:"variables"`
	TagsToExclude  []types.String `tfsdk:"tags_to_exclude"`
}

type PolicyScopeModel struct {
	Projects             *ProjectsScopeModel `tfsdk:"projects"`
	ComplianceFrameworks []types.String      `tfsdk:"compliance_frameworks"`
}

type ProjectsScopeModel struct {
	Including []types.Int64 `tfsdk:"including"`
	Excluding []types.Int64 `tfsdk:"excluding"`
}

type SkipCiModel struct {
	Allowed types.Bool `tfsdk:"allowed"`
}

// buildMarkdownList creates a new comma-separated list of backtick-wrapped items
func buildMarkdownList(items []string) string {
	result := ""
	for i, item := range items {
		result += "`" + item + "`"
		if i < len(items)-1 {
			result += ", "
		}
	}
	return result
}

// Metadata sets the data source type name
func (d *gitlabSecurityPolicyDocumentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_policy_document"
}

// Schema defines the data source schema
func (d *gitlabSecurityPolicyDocumentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Generates a GitLab security policy YAML document from structured configuration.
This data source performs pure transformation without any API calls.

**Upstream API**: [GitLab Security Policies Documentation](https://docs.gitlab.com/ee/user/application_security/policies/scan_execution_policies/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for this policy document (hash of generated YAML).",
			},
			"yaml": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The generated policy document in YAML format, ready to write to `.gitlab/security-policies/policy.yml`.",
			},
			"scan_execution_policy": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Scan execution policy configuration. Multiple policies can be specified.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Name of the scan execution policy.",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Description of the scan execution policy.",
						},
						"enabled": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Whether the policy is enabled.",
						},
						"rules": schema.ListNestedAttribute{
							Required:            true,
							MarkdownDescription: "Rules that trigger the policy. At least one rule is required.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: fmt.Sprintf("Type of rule. Valid values: %s.", buildMarkdownList(scanExecutionRuleTypes)),
									},
									"branches": schema.ListAttribute{
										Optional:            true,
										ElementType:         types.StringType,
										MarkdownDescription: "Branch names or patterns to match.",
									},
									"branch_type": schema.StringAttribute{
										Optional:            true,
										MarkdownDescription: fmt.Sprintf("Type of branches to match. Valid values: %s.", buildMarkdownList(scanExecutionBranchTypes)),
									},
									"branch_exceptions": schema.ListAttribute{
										Optional:            true,
										ElementType:         types.StringType,
										MarkdownDescription: "Branches to exclude from the policy.",
									},
									"cadence": schema.StringAttribute{
										Optional:            true,
										MarkdownDescription: "Cron expression for schedule type rules (e.g., `*/15 * * * *` for every 15 minutes).",
									},
									"agents": schema.MapAttribute{
										Optional:            true,
										ElementType:         types.StringType,
										MarkdownDescription: "Kubernetes agents configuration for agent-based policies.",
									},
								},
							},
						},
						"actions": schema.ListNestedAttribute{
							Required:            true,
							MarkdownDescription: "Actions to execute when rules match. At least one action is required.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"scan": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: fmt.Sprintf("Type of scan to run. Valid values: %s.", buildMarkdownList(scanExecutionScanTypes)),
									},
									"template": schema.StringAttribute{
										Optional:            true,
										MarkdownDescription: "The template to use for the scan. Valid values: `default`, `latest`.",
									},
									"site_profile": schema.StringAttribute{
										Optional:            true,
										MarkdownDescription: "Site profile to use for DAST scans.",
									},
									"scanner_profile": schema.StringAttribute{
										Optional:            true,
										MarkdownDescription: "Scanner profile to use for DAST scans.",
									},
									"variables": schema.MapAttribute{
										Optional:            true,
										ElementType:         types.StringType,
										MarkdownDescription: "Environment variables to pass to the scan job.",
									},
									"tags_to_exclude": schema.ListAttribute{
										Optional:            true,
										ElementType:         types.StringType,
										MarkdownDescription: "Tags to exclude from the scan.",
									},
								},
							},
						},
						"policy_scope": schema.SingleNestedAttribute{
							Optional:            true,
							MarkdownDescription: "Scope configuration to limit which projects the policy applies to.",
							Attributes: map[string]schema.Attribute{
								"compliance_frameworks": schema.ListAttribute{
									Optional:            true,
									ElementType:         types.StringType,
									MarkdownDescription: "Compliance framework names to scope the policy to.",
								},
								"projects": schema.SingleNestedAttribute{
									Optional:            true,
									MarkdownDescription: "Project scope configuration.",
									Attributes: map[string]schema.Attribute{
										"including": schema.ListAttribute{
											Optional:            true,
											ElementType:         types.Int64Type,
											MarkdownDescription: "List of project IDs to explicitly include in this policy.",
										},
										"excluding": schema.ListAttribute{
											Optional:            true,
											ElementType:         types.Int64Type,
											MarkdownDescription: "List of project IDs to exclude from this policy.",
										},
									},
								},
							},
						},
						"skip_ci": schema.SingleNestedAttribute{
							Optional:            true,
							MarkdownDescription: "Control whether users can use the skip-ci directive.",
							Attributes: map[string]schema.Attribute{
								"allowed": schema.BoolAttribute{
									Required:            true,
									MarkdownDescription: "Allow (true) or prevent (false) the use of skip-ci directive.",
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure - no-op since we don't need API client
func (d *gitlabSecurityPolicyDocumentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// No config needed - pure transformation data source
	// that doesn't interact with the GitLab API
}

// Read generates the YAML output from config
func (d *gitlabSecurityPolicyDocumentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabSecurityPolicyDocumentDataSourceModel

	// Read Terraform config
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the policy doc structure
	policyDoc := make(map[string]any)

	if len(data.ScanExecutionPolicies) > 0 {
		policies := make([]map[string]any, 0, len(data.ScanExecutionPolicies))
		for _, policy := range data.ScanExecutionPolicies {
			p := map[string]any{
				"name":    policy.Name.ValueString(),
				"enabled": policy.Enabled.ValueBool(),
			}

			// Add optional description if provided
			if !policy.Description.IsNull() && !policy.Description.IsUnknown() {
				p["description"] = policy.Description.ValueString()
			}

			// Add rules
			if len(policy.Rules) > 0 {
				rules := make([]map[string]any, 0, len(policy.Rules))
				for _, rule := range policy.Rules {
					r := map[string]any{
						"type": rule.Type.ValueString(),
					}

					// Add optional branches
					if len(rule.Branches) > 0 {
						branches := make([]string, 0, len(rule.Branches))
						for _, branch := range rule.Branches {
							branches = append(branches, branch.ValueString())
						}
						r["branches"] = branches
					}

					if !rule.BranchType.IsNull() && !rule.BranchType.IsUnknown() {
						r["branch_type"] = rule.BranchType.ValueString()
					}

					if len(rule.BranchExceptions) > 0 {
						exceptions := make([]string, 0, len(rule.BranchExceptions))
						for _, exc := range rule.BranchExceptions {
							exceptions = append(exceptions, exc.ValueString())
						}
						r["branch_exceptions"] = exceptions
					}

					if !rule.Cadence.IsNull() && !rule.Cadence.IsUnknown() {
						r["cadence"] = rule.Cadence.ValueString()
					}

					if !rule.Agents.IsNull() && !rule.Agents.IsUnknown() {
						// Converts agents map
						agents := make(map[string]any)
						rule.Agents.ElementsAs(ctx, &agents, false)
						r["agents"] = agents
					}

					rules = append(rules, r)
				}
				p["rules"] = rules
			}

			// Add actions
			if len(policy.Actions) > 0 {
				actions := make([]map[string]any, 0, len(policy.Actions))
				for _, action := range policy.Actions {
					a := map[string]any{
						"scan": action.Scan.ValueString(),
					}

					// add template if provided
					if !action.Template.IsNull() && !action.Template.IsUnknown() {
						a["template"] = action.Template.ValueString()
					}

					if !action.SiteProfile.IsNull() && !action.SiteProfile.IsUnknown() {
						a["site_profile"] = action.SiteProfile.ValueString()
					}

					if !action.ScannerProfile.IsNull() && !action.ScannerProfile.IsUnknown() {
						a["scanner_profile"] = action.ScannerProfile.ValueString()
					}

					if !action.Variables.IsNull() && !action.Variables.IsUnknown() {
						variables := make(map[string]string)
						action.Variables.ElementsAs(ctx, &variables, false)
						if len(variables) > 0 {
							a["variables"] = variables
						}
					}

					if len(action.TagsToExclude) > 0 {
						tags := make([]string, 0, len(action.TagsToExclude))
						for _, tag := range action.TagsToExclude {
							tags = append(tags, tag.ValueString())
						}
						a["tags_to_exclude"] = tags
					}

					actions = append(actions, a)
				}
				p["actions"] = actions
			}

			// Add policy scope
			if policy.PolicyScope != nil {
				scope := make(map[string]any)

				// Handle projects scope
				if policy.PolicyScope.Projects != nil {
					projects := make(map[string]any)

					// Add excluding list if present
					if len(policy.PolicyScope.Projects.Excluding) > 0 {
						excluding := []int64{}
						for _, id := range policy.PolicyScope.Projects.Excluding {
							excluding = append(excluding, id.ValueInt64())
						}
						projects["excluding"] = excluding
					}
					// add including list if present (optional)
					if len(policy.PolicyScope.Projects.Including) > 0 {
						including := []int64{}
						for _, id := range policy.PolicyScope.Projects.Including {
							including = append(including, id.ValueInt64())
						}
						projects["including"] = including
					}

					scope["projects"] = projects
				}

				if len(policy.PolicyScope.ComplianceFrameworks) > 0 {
					frameworks := []string{}
					for _, fw := range policy.PolicyScope.ComplianceFrameworks {
						frameworks = append(frameworks, fw.ValueString())
					}
					scope["compliance_frameworks"] = frameworks
				}

				if len(scope) > 0 {
					p["policy_scope"] = scope
				}
			}

			// Add skip_ci config
			if policy.SkipCi != nil {
				skipCi := map[string]any{
					"allowed": policy.SkipCi.Allowed.ValueBool(),
				}
				p["skip_ci"] = skipCi
			}

			policies = append(policies, p)
		}
		policyDoc["scan_execution_policy"] = policies
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(policyDoc)
	if err != nil {
		resp.Diagnostics.AddError(
			"YAML Generation Failed",
			fmt.Sprintf("Could not generate YAML from policy configuration: %s", err.Error()),
		)
		return
	}

	// Generate stable ID from YAML content (SHA256 hash)
	hash := sha256.Sum256(yamlBytes)
	hashString := hex.EncodeToString(hash[:])

	// Set the outputs
	data.Yaml = types.StringValue(string(yamlBytes))
	data.Id = types.StringValue(hashString)

	// Save to Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
