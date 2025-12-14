package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

const complianceRequirementIDSeparator = "|"

// parseComplianceRequirementID parses the terraform resource ID into framework_id and requirement_id.
// Uses "|" as separator since GIDs contain colons.
func parseComplianceRequirementID(id string) (string, string, error) {
	parts := strings.SplitN(id, complianceRequirementIDSeparator, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected ID format (%q). Expected <framework_id>|<requirement_id>", id)
	}
	return parts[0], parts[1], nil
}

// buildComplianceRequirementID builds the terraform resource ID from framework_id and requirement_id.
func buildComplianceRequirementID(frameworkID, requirementID string) string {
	return fmt.Sprintf("%s%s%s", frameworkID, complianceRequirementIDSeparator, requirementID)
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &gitlabComplianceRequirementResource{}
	_ resource.ResourceWithConfigure      = &gitlabComplianceRequirementResource{}
	_ resource.ResourceWithImportState    = &gitlabComplianceRequirementResource{}
	_ resource.ResourceWithValidateConfig = &gitlabComplianceRequirementResource{}
)

func init() {
	registerResource(NewGitLabComplianceRequirementResource)
}

func NewGitLabComplianceRequirementResource() resource.Resource {
	return &gitlabComplianceRequirementResource{}
}

type gitlabComplianceRequirementResource struct {
	client *gitlab.Client
}

type gitlabComplianceRequirementResourceModel struct {
	Id          types.String `tfsdk:"id"`
	FrameworkId types.String `tfsdk:"framework_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Controls    types.List   `tfsdk:"controls"`
}

type gitlabComplianceControlModel struct {
	Name        types.String `tfsdk:"name"`
	ControlType types.String `tfsdk:"control_type"`
	Expression  types.Object `tfsdk:"expression"`
	ExternalURL types.String `tfsdk:"external_url"`
	SecretToken types.String `tfsdk:"secret_token"`
}

type gitlabControlExpressionModel struct {
	Field    types.String `tfsdk:"field"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

// Metadata returns the resource name
func (r *gitlabComplianceRequirementResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_requirement"
}

func (r *gitlabComplianceRequirementResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_compliance_requirement`" + ` resource allows managing the lifecycle of a compliance requirement within a compliance framework.

Compliance requirements define specific compliance conditions that projects must meet, with associated controls that specify how compliance is verified.

-> This resource requires a GitLab Enterprise instance with an Ultimate license.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/#mutationcreatecompliancerequirement)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<framework_id>|<requirement_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"framework_id": schema.StringAttribute{
				MarkdownDescription: "The globally unique ID of the compliance framework to add the requirement to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name for the compliance requirement.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description for the compliance requirement.",
				Optional:            true,
			},
			"controls": schema.ListNestedAttribute{
				MarkdownDescription: "List of controls for this compliance requirement. Controls define how compliance is verified.",
				Optional:            true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the control.",
							Required:            true,
							Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"control_type": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf("Type of control. Valid values are %s.", utils.RenderValueListForDocs(api.ValidComplianceControlTypes)),
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf(api.ValidComplianceControlTypes...),
							},
						},
						"external_url": schema.StringAttribute{
							MarkdownDescription: "External URL for external controls. Required when `control_type` is `external`.",
							Optional:            true,
						},
						"secret_token": schema.StringAttribute{
							MarkdownDescription: "Secret token for external controls. Optional when `control_type` is `external`.",
							Optional:            true,
							Sensitive:           true,
						},
						"expression": schema.SingleNestedAttribute{
							MarkdownDescription: "Expression for internal controls. Required when `control_type` is `internal`.",
							Optional:            true,
							Attributes: map[string]schema.Attribute{
								"field": schema.StringAttribute{
									MarkdownDescription: "The field to evaluate (e.g., `scanner_dep_scanning_running`).",
									Required:            true,
								},
								"operator": schema.StringAttribute{
									MarkdownDescription: fmt.Sprintf("The operator for comparison. Valid values are %s.", utils.RenderValueListForDocs(api.ValidComplianceControlOperators)),
									Required:            true,
									Validators: []validator.String{
										stringvalidator.OneOf(api.ValidComplianceControlOperators...),
									},
								},
								"value": schema.StringAttribute{
									MarkdownDescription: "The value to compare against. Use `true` or `false` for boolean values.",
									Required:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *gitlabComplianceRequirementResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data gitlabComplianceRequirementResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Controls.IsNull() || data.Controls.IsUnknown() {
		return
	}

	var controls []gitlabComplianceControlModel
	resp.Diagnostics.Append(data.Controls.ElementsAs(ctx, &controls, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i, control := range controls {
		if control.ControlType.IsUnknown() {
			continue
		}

		controlType := control.ControlType.ValueString()

		if controlType == "external" {
			if control.ExternalURL.IsNull() {
				resp.Diagnostics.AddAttributeError(
					path.Root("controls").AtListIndex(i).AtName("external_url"),
					"Missing Attribute Configuration",
					"external_url is required when control_type is 'external'",
				)
			}
		} else if controlType == "internal" {
			if control.Expression.IsNull() {
				resp.Diagnostics.AddAttributeError(
					path.Root("controls").AtListIndex(i).AtName("expression"),
					"Missing Block Configuration",
					"expression block is required when control_type is 'internal'",
				)
			}
		}
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabComplianceRequirementResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Create creates a new upstream resource and adds it into the Terraform state.
func (r *gitlabComplianceRequirementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabComplianceRequirementResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	frameworkID := data.FrameworkId.ValueString()
	name := data.Name.ValueString()
	description := data.Description.ValueString()

	// Build controls input
	controlsInput, err := r.buildControlsInput(ctx, data.Controls)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build controls input", err.Error())
		return
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				createComplianceRequirement(
					input: {
						complianceFrameworkId: "%s",
						name: "%s",
						description: "%s",
						controls: %s
					}
				) {
					complianceRequirement {
						id,
						name,
						description
					}
					errors
				}
			}`, frameworkID, escapeGraphQLString(name), escapeGraphQLString(description), controlsInput),
	}
	tflog.Debug(ctx, "executing GraphQL Query to create compliance requirement")

	var response createComplianceRequirementResponse
	if _, err := r.client.GraphQL.Do(query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create compliance requirement: %s", err.Error()))
		return
	}

	// Check response for errors
	if len(response.Errors) > 0 || len(response.Data.CreateComplianceRequirement.Errors) > 0 {
		var allerr string
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
		}
		for i, err := range response.Data.CreateComplianceRequirement.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
		}
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", allerr)
		return
	}

	requirementID := response.Data.CreateComplianceRequirement.ComplianceRequirement.ID
	data.Id = types.StringValue(buildComplianceRequirementID(frameworkID, requirementID))

	tflog.Debug(ctx, "created a compliance requirement", map[string]any{
		"id":           data.Id.ValueString(),
		"framework_id": frameworkID,
		"name":         name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabComplianceRequirementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabComplianceRequirementResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	frameworkID, requirementID, err := parseComplianceRequirementID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<framework_id>|<requirement_id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			query {
				complianceFramework(id: "%s") {
					id
					complianceRequirements(id: "%s") {
						nodes {
							id
							name
							description
						}
					}
				}
			}`, frameworkID, requirementID),
	}
	tflog.Debug(ctx, "executing GraphQL Query to read compliance requirement", map[string]any{
		"query": query.Query,
	})

	var response readComplianceRequirementResponse
	if _, err := r.client.GraphQL.Do(query, &response); err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "compliance requirement does not exist, removing from state", map[string]any{
				"framework_id":   frameworkID,
				"requirement_id": requirementID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read compliance requirement: %s", err.Error()))
		return
	}

	// Check if requirement exists
	if len(response.Data.ComplianceFramework.ComplianceRequirements.Nodes) == 0 {
		tflog.Debug(ctx, "compliance requirement does not exist, removing from state", map[string]any{
			"framework_id":   frameworkID,
			"requirement_id": requirementID,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	requirement := response.Data.ComplianceFramework.ComplianceRequirements.Nodes[0]
	data.FrameworkId = types.StringValue(frameworkID)
	data.Name = types.StringValue(requirement.Name)
	if requirement.Description != "" {
		data.Description = types.StringValue(requirement.Description)
	} else {
		data.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabComplianceRequirementResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabComplianceRequirementResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, requirementID, err := parseComplianceRequirementID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<framework_id>|<requirement_id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	name := data.Name.ValueString()
	description := data.Description.ValueString()

	// Build controls input
	controlsInput, err := r.buildControlsInput(ctx, data.Controls)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build controls input", err.Error())
		return
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				updateComplianceRequirement(
					input: {
						id: "%s",
						name: "%s",
						description: "%s",
						controls: %s
					}
				) {
					complianceRequirement {
						id,
						name,
						description
					}
					errors
				}
			}`, requirementID, escapeGraphQLString(name), escapeGraphQLString(description), controlsInput),
	}
	tflog.Debug(ctx, "executing GraphQL Query to update compliance requirement")

	var response updateComplianceRequirementResponse
	if _, err := r.client.GraphQL.Do(query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update compliance requirement: %s", err.Error()))
		return
	}

	// Check response for errors
	if len(response.Errors) > 0 || len(response.Data.UpdateComplianceRequirement.Errors) > 0 {
		var allerr string
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
		}
		for i, err := range response.Data.UpdateComplianceRequirement.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
		}
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", allerr)
		return
	}

	tflog.Debug(ctx, "updated compliance requirement", map[string]any{
		"id":   data.Id.ValueString(),
		"name": name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the resource.
func (r *gitlabComplianceRequirementResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabComplianceRequirementResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, requirementID, err := parseComplianceRequirementID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<framework_id>|<requirement_id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				destroyComplianceRequirement(
					input: {
						id: "%s"
					}
				) {
					errors
				}
			}`, requirementID),
	}
	tflog.Debug(ctx, "executing GraphQL Query to delete compliance requirement", map[string]any{
		"query": query.Query,
	})

	var response deleteComplianceRequirementResponse
	if _, err := r.client.GraphQL.Do(query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete compliance requirement: %s", err.Error()))
		return
	}

	// Check response for errors
	if len(response.Errors) > 0 || len(response.Data.DestroyComplianceRequirement.Errors) > 0 {
		var allerr string
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
		}
		for i, err := range response.Data.DestroyComplianceRequirement.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
		}
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", allerr)
		return
	}
}

func (r *gitlabComplianceRequirementResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildControlsInput constructs the GraphQL input for controls
func (r *gitlabComplianceRequirementResource) buildControlsInput(ctx context.Context, controlsList types.List) (string, error) {
	if controlsList.IsNull() || controlsList.IsUnknown() || len(controlsList.Elements()) == 0 {
		return "[]", nil
	}

	var controls []gitlabComplianceControlModel
	diags := controlsList.ElementsAs(ctx, &controls, false)
	if diags.HasError() {
		return "", fmt.Errorf("failed to parse controls")
	}

	var controlStrings []string
	for _, control := range controls {
		var controlStr string
		controlType := control.ControlType.ValueString()

		if controlType == "external" {
			externalURL := control.ExternalURL.ValueString()
			secretToken := control.SecretToken.ValueString()
			controlStr = fmt.Sprintf(`{
				name: "%s",
				controlType: EXTERNAL,
				externalUrl: "%s"`,
				escapeGraphQLString(control.Name.ValueString()),
				escapeGraphQLString(externalURL))
			if secretToken != "" {
				controlStr += fmt.Sprintf(`,
				secretToken: "%s"`, escapeGraphQLString(secretToken))
			}
			controlStr += `}`
		} else {
			// Internal control with expression
			var expr gitlabControlExpressionModel
			if !control.Expression.IsNull() && !control.Expression.IsUnknown() {
				diags := control.Expression.As(ctx, &expr, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return "", fmt.Errorf("failed to parse expression")
				}

				// Determine if value is boolean or string
				value := expr.Value.ValueString()
				var valueStr string
				if value == "true" || value == "false" {
					valueStr = value
				} else {
					valueStr = fmt.Sprintf(`"%s"`, escapeGraphQLString(value))
				}

				controlStr = fmt.Sprintf(`{
					name: "%s",
					controlType: INTERNAL,
					expression: {
						field: "%s",
						operator: %s,
						value: %s
					}
				}`,
					escapeGraphQLString(control.Name.ValueString()),
					escapeGraphQLString(expr.Field.ValueString()),
					strings.ToUpper(expr.Operator.ValueString()),
					valueStr)
			} else {
				controlStr = fmt.Sprintf(`{
					name: "%s",
					controlType: INTERNAL
				}`, escapeGraphQLString(control.Name.ValueString()))
			}
		}
		controlStrings = append(controlStrings, controlStr)
	}

	return "[" + strings.Join(controlStrings, ", ") + "]", nil
}

// escapeGraphQLString escapes special characters in a string for use in GraphQL queries
func escapeGraphQLString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// GraphQL response types

type createComplianceRequirementResponse struct {
	Data struct {
		CreateComplianceRequirement struct {
			ComplianceRequirement api.GraphQLComplianceRequirement `json:"complianceRequirement"`
			Errors                []string                         `json:"errors"`
		} `json:"createComplianceRequirement"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type readComplianceRequirementResponse struct {
	Data struct {
		ComplianceFramework struct {
			ID                     string `json:"id"`
			ComplianceRequirements struct {
				Nodes []api.GraphQLComplianceRequirement `json:"nodes"`
			} `json:"complianceRequirements"`
		} `json:"complianceFramework"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type updateComplianceRequirementResponse struct {
	Data struct {
		UpdateComplianceRequirement struct {
			ComplianceRequirement api.GraphQLComplianceRequirement `json:"complianceRequirement"`
			Errors                []string                         `json:"errors"`
		} `json:"updateComplianceRequirement"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type deleteComplianceRequirementResponse struct {
	Data struct {
		DestroyComplianceRequirement struct {
			Errors []string `json:"errors"`
		} `json:"destroyComplianceRequirement"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}
