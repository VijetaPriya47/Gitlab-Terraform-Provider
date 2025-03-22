package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabComplianceFrameworkResource{}
	_ resource.ResourceWithConfigure   = &gitlabComplianceFrameworkResource{}
	_ resource.ResourceWithImportState = &gitlabComplianceFrameworkResource{}
)

func init() {
	registerResource(NewGitLabComplianceFrameworkResource)
}

func NewGitLabComplianceFrameworkResource() resource.Resource {
	return &gitlabComplianceFrameworkResource{}
}

type gitlabComplianceFrameworkResource struct {
	client *gitlab.Client
}

type gitlabComplianceFrameworkResourceModel struct {
	Id                            types.String `tfsdk:"id"`
	FrameworkId                   types.String `tfsdk:"framework_id"`
	NamespacePath                 types.String `tfsdk:"namespace_path"`
	Name                          types.String `tfsdk:"name"`
	Description                   types.String `tfsdk:"description"`
	Color                         types.String `tfsdk:"color"`
	DefaultFramework              types.Bool   `tfsdk:"default"`
	PipelineConfigurationFullPath types.String `tfsdk:"pipeline_configuration_full_path"`
}

// Metadata returns the resource name
func (d *gitlabComplianceFrameworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_framework"
}

func (r *gitlabComplianceFrameworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_compliance_framework`" + ` resource allows to manage the lifecycle of a compliance framework on top-level groups.

There can be only one ` + "`default`" + ` compliance framework. Of all the configured compliance frameworks marked as default, the last one applied will be the default compliance framework.

-> This resource requires a GitLab Enterprise instance with a Premium license to create the compliance framework.

-> This resource requires a GitLab Enterprise instance with an Ultimate license to specify a compliance pipeline configuration in the compliance framework.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/#mutationcreatecomplianceframework)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<namespace_path>:<framework_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"framework_id": schema.StringAttribute{
				MarkdownDescription: "Globally unique ID of the compliance framework.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"namespace_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the namespace to add the compliance framework to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name for the compliance framework.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description for the compliance framework.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "New color representation of the compliance framework in hex format. e.g. #FCA121.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`), "value must be a valid color code"),
				},
			},
			"default": schema.BoolAttribute{
				MarkdownDescription: "Set this compliance framework as the default framework for the group. Default: `false`",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"pipeline_configuration_full_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the compliance pipeline configuration stored in a project repository, such as `.gitlab/.compliance-gitlab-ci.yml@compliance/hipaa`. Required format: `path/file.y[a]ml@group-name/project-name` **Note**: Ultimate license required.",
				Optional:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabComplianceFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabComplianceFrameworkResource) complianceFrameworkToStateModel(response *api.GraphQLComplianceFramework, namespacePath string, data *gitlabComplianceFrameworkResourceModel) {
	data.FrameworkId = types.StringValue(response.ID)
	data.NamespacePath = types.StringValue(namespacePath)
	data.Name = types.StringValue(response.Name)
	data.Description = types.StringValue(response.Description)
	data.Color = types.StringValue(response.Color)
	data.DefaultFramework = types.BoolValue(response.DefaultFramework)
	if response.PipelineConfigurationFullPath != "" {
		data.PipelineConfigurationFullPath = types.StringValue(response.PipelineConfigurationFullPath)
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabComplianceFrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabComplianceFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	namespacePath, frameworkID, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<namespace_path>:<framework_id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			query {
				namespace(fullPath: "%s") {
					fullPath,
					complianceFrameworks(id: "%s") {
						nodes {
							id,
							name,
							description,
							color,
							default,
							pipelineConfigurationFullPath
						}
					}
				}
			}`, namespacePath, frameworkID),
	}
	tflog.Debug(ctx, "executing GraphQL Query to retrieve current compliance framework", map[string]interface{}{
		"query": query.Query,
	})

	var response ComplianceFrameworkResponse
	if _, err := api.SendGraphQLRequest(ctx, r.client, query, &response); err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "compliance framework does not exist, removing from state", map[string]interface{}{
				"namespace_path": namespacePath, "framework_id": frameworkID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read compliance framework details: %s", err.Error()))
		return
	}

	// error if 0 or >1 compliance framework was returned
	if len(response.Data.Namespace.ComplianceFrameworks.Nodes) != 1 {
		resp.Diagnostics.AddError("Compliance Framework not found", fmt.Sprintf("Unable to find Compliance Framework: %s in namespace: %s", frameworkID, namespacePath))
		return
	}

	// persist API response in state model
	r.complianceFrameworkToStateModel(&response.Data.Namespace.ComplianceFrameworks.Nodes[0], response.Data.Namespace.NamespacePath, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create creates a new upstream resource and adds it into the Terraform state.
func (r *gitlabComplianceFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabComplianceFrameworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespacePath := data.NamespacePath.ValueString()
	name := data.Name.ValueString()
	description := data.Description.ValueString()
	color := data.Color.ValueString()
	defaultFramework := data.DefaultFramework.ValueBool()

	var pipelineConfigurationFullPath string
	if !data.PipelineConfigurationFullPath.IsNull() && !data.PipelineConfigurationFullPath.IsUnknown() {
		pipelineConfigurationFullPath = data.PipelineConfigurationFullPath.ValueString()
	}

	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				createComplianceFramework(
					input: {
						params: {
							name: "%s",
							description: "%s",
							color: "%s",
							default: %t,
							pipelineConfigurationFullPath: "%s"
						},
						namespacePath: "%s"
					}
				) {
					framework {
						id,
						name,
						description,
						color,
						default,
						pipelineConfigurationFullPath
					}
					errors
				}
			}`, name, description, color, defaultFramework, pipelineConfigurationFullPath, namespacePath),
	}
	tflog.Debug(ctx, "executing GraphQL Query to create compliance framework", map[string]interface{}{
		"query": query.Query,
	})

	var response createComplianceFrameworkResponse
	if _, err := api.SendGraphQLRequest(ctx, r.client, query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create compliance framework: %s", err.Error()))
		return
	}

	// check response for errors
	var allerr string
	if len(response.Errors) > 0 {
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
		}
	}
	if len(response.Data.CreateComplianceFramework.Errors) > 0 {
		for i, err := range response.Data.CreateComplianceFramework.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
		}
	}
	if len(allerr) > 0 {
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", allerr)
		return
	}

	// Create resource ID and persist in state model
	data.Id = types.StringValue(utils.BuildTwoPartID(&namespacePath, &response.Data.CreateComplianceFramework.Framework.ID))

	// persist API response in state model
	r.complianceFrameworkToStateModel(&response.Data.CreateComplianceFramework.Framework, namespacePath, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "created a compliance framework", map[string]interface{}{
		"id": data.Id.ValueString(), "namespace_path": data.NamespacePath.ValueString(), "framework_id": data.FrameworkId.ValueString(), "name": data.Name.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabComplianceFrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabComplianceFrameworkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.update(ctx, data, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update compliance framework", err.Error())
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the resource.
func (r *gitlabComplianceFrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabComplianceFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, frameworkID, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<namespace_path>:<framework_id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	// check if compliance framework is the default framework
	// since the default framework can't be deleted, remove the default setting from the framework first
	defaultFramework := data.DefaultFramework.ValueBool()
	if defaultFramework {
		data.DefaultFramework = types.BoolValue(false)
		err := r.update(ctx, data, &resp.Diagnostics)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update compliance framework during delete", err.Error())
			return
		}
	}

	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				destroyComplianceFramework(
					input: {
						id: "%s"
					}
				) {
					errors
				}
			}`, frameworkID),
	}
	tflog.Debug(ctx, "executing GraphQL Query to delete compliance framework", map[string]interface{}{
		"query": query.Query,
	})

	if _, err := api.SendGraphQLRequest(ctx, r.client, query, nil); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete compliance framework: %s", err.Error()))
		return
	}
}

func (r *gitlabComplianceFrameworkResource) update(ctx context.Context, data *gitlabComplianceFrameworkResourceModel, diags *diag.Diagnostics) error {
	namespacePath, frameworkID, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		diags.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<namespace_path>:<framework_id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return err
	}

	name := data.Name.ValueString()
	description := data.Description.ValueString()
	color := data.Color.ValueString()
	defaultFramework := data.DefaultFramework.ValueBool()

	var pipelineConfigurationFullPath string
	if !data.PipelineConfigurationFullPath.IsNull() && !data.PipelineConfigurationFullPath.IsUnknown() {
		pipelineConfigurationFullPath = data.PipelineConfigurationFullPath.ValueString()
	}

	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				updateComplianceFramework(
					input: {
						params: {
							name: "%s",
							description: "%s",
							color: "%s",
							default: %t,
							pipelineConfigurationFullPath: "%s"
						},
						id: "%s"
					}
				) {
					complianceFramework {
						id,
						name,
						description,
						color,
						default,
						pipelineConfigurationFullPath
					}
					errors
				}
			}`, name, description, color, defaultFramework, pipelineConfigurationFullPath, frameworkID),
	}
	tflog.Debug(ctx, "executing GraphQL Query to update compliance framework", map[string]interface{}{
		"query": query.Query,
	})

	var response updateComplianceFrameworkResponse
	if _, err := api.SendGraphQLRequest(ctx, r.client, query, &response); err != nil {
		diags.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update compliance framework: %s", err.Error()))
		return err
	}

	// check response for errors
	var allerr string
	if len(response.Errors) > 0 {
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
		}
	}
	if len(response.Data.UpdateComplianceFramework.Errors) > 0 {
		for i, err := range response.Data.UpdateComplianceFramework.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
		}
	}
	if len(allerr) > 0 {
		diags.AddError("GitLab GraphQL error occurred", allerr)
		return fmt.Errorf("GitLab GraphQL error occurred: %s", allerr)
	}

	// persist API response in state model
	r.complianceFrameworkToStateModel(&response.Data.UpdateComplianceFramework.ComplianceFramework, namespacePath, data)

	// Log the update of the resource
	tflog.Debug(ctx, "updated a compliance framework", map[string]interface{}{
		"id": data.Id.ValueString(), "namespace_path": data.NamespacePath.ValueString(), "framework_id": data.FrameworkId.ValueString(), "name": data.Name.ValueString(),
	})

	return nil
}

func (r *gitlabComplianceFrameworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type ComplianceFrameworkResponse struct {
	Data struct {
		Namespace struct {
			NamespacePath        string `json:"fullPath"`
			ComplianceFrameworks struct {
				Nodes []api.GraphQLComplianceFramework `json:"nodes"`
			} `json:"complianceFrameworks"`
		} `json:"namespace"`
	} `json:"data"`
}

type createComplianceFrameworkResponse struct {
	Data struct {
		CreateComplianceFramework struct {
			Framework api.GraphQLComplianceFramework `json:"framework"`
			Errors    []string                       `json:"errors"`
		} `json:"createComplianceFramework"`
	} `json:"data"`
	Errors []struct {
		Message   string `json:"message"`
		Locations []struct {
			Line   int `json:"line"`
			Column int `json:"column"`
		} `json:"locations"`
		Path []string `json:"path"`
	} `json:"errors"`
}

type updateComplianceFrameworkResponse struct {
	Data struct {
		UpdateComplianceFramework struct {
			ComplianceFramework api.GraphQLComplianceFramework `json:"complianceFramework"`
			Errors              []string                       `json:"errors"`
		} `json:"updateComplianceFramework"`
	} `json:"data"`
	Errors []struct {
		Message   string `json:"message"`
		Locations []struct {
			Line   int `json:"line"`
			Column int `json:"column"`
		} `json:"locations"`
		Path []string `json:"path"`
	} `json:"errors"`
}
