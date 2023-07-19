package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabProjectComplianceFrameworkResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectComplianceFrameworkResource{}
	_ resource.ResourceWithImportState = &gitlabProjectComplianceFrameworkResource{}
)

func init() {
	registerResource(NewGitLabProjectComplianceFrameworkResource)
}

func NewGitLabProjectComplianceFrameworkResource() resource.Resource {
	return &gitlabProjectComplianceFrameworkResource{}
}

type gitlabProjectComplianceFrameworkResource struct {
	client *gitlab.Client
}

type gitlabProjectComplianceFrameworkResourceModel struct {
	Id                    types.String `tfsdk:"id"`
	ComplianceFrameworkId types.String `tfsdk:"compliance_framework_id"`
	Project               types.String `tfsdk:"project"`
}

// Metadata returns the resource name
func (d *gitlabProjectComplianceFrameworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_compliance_framework"
}

func (r *gitlabProjectComplianceFrameworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_compliance_framework`" + ` resource allows to manage the lifecycle of a compliance framework on a project.

-> This resource requires a GitLab Enterprise instance with a Premium license to set the compliance framework on a project.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/ee/api/graphql/reference/#mutationprojectsetcomplianceframework)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"compliance_framework_id": schema.StringAttribute{
				MarkdownDescription: "Globally unique ID of the compliance framework to assign to the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project to change the compliance framework of.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectComplianceFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

func (r *gitlabProjectComplianceFrameworkResource) projectComplianceFrameworkToStateModel(response *graphQLProject, data *gitlabProjectComplianceFrameworkResourceModel) {
	data.ComplianceFrameworkId = types.StringValue(response.ComplianceFrameworks.Nodes[0].ID)
	data.Project = data.Id
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabProjectComplianceFrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectComplianceFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The project attribute may either be a numerical project id or the full path,
	// thus we always resolve it to a project to gather the full path for subsequent GraphQL API calls which ALWAYS
	// require the full path ...
	project, _, err := r.client.Projects.GetProject(data.Id.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "project does not exist, removing resource from state", map[string]interface{}{
				"project": data.Id.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project details: %s", err.Error()))
		return
	}

	// read all information for refresh from resource id
	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			query {
				project(fullPath: "%s") {
					id,
					fullPath,
					complianceFrameworks {
						nodes {
							id
						}
					}
				}
			}`, project.PathWithNamespace),
	}
	tflog.Debug(ctx, "executing GraphQL Query to retrieve current compliance framework on project", map[string]interface{}{
		"query": query.Query,
	})

	var response projectResponse
	if _, err = api.SendGraphQLRequest(ctx, r.client, query, &response); err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "compliance framework does not exist on project, removing from state", map[string]interface{}{
				"project_path_with_namespace": project.PathWithNamespace,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project compliance framework details: %s", err.Error()))
		return
	}

	// error if 0 or >1 project compliance frameworks were returned
	if len(response.Data.Project.ComplianceFrameworks.Nodes) != 1 {
		resp.Diagnostics.AddError("Project Compliance Framework not found", fmt.Sprintf("Unable to find Compliance Framework on project: %s", project.PathWithNamespace))
		return
	}

	// persist API response in state model
	r.projectComplianceFrameworkToStateModel(&response.Data.Project, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create creates a new upstream resource and adds it into the Terraform state.
func (r *gitlabProjectComplianceFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectComplianceFrameworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	frameworkId := data.ComplianceFrameworkId.ValueString()

	// The project attribute may either be a numerical project id or the full path,
	// thus we always resolve it to a project to gather the full path for subsequent GraphQL API calls which ALWAYS
	// require the full path ...
	project, _, err := r.client.Projects.GetProject(data.Project.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "project does not exist, removing resource from state", map[string]interface{}{
				"project": data.Project.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project details: %s", err.Error()))
		return
	}

	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				projectSetComplianceFramework(
					input: {
						projectId: "gid://gitlab/Project/%d",
						complianceFrameworkId: "%s"
					}
				) {
					project {
						id,
						fullPath,
						complianceFrameworks {
							nodes {
								id
							}
						}
					}
					errors
				}
			}`, project.ID, frameworkId),
	}
	tflog.Debug(ctx, "executing GraphQL Query to set project compliance framework", map[string]interface{}{
		"query": query.Query,
	})

	var response projectSetComplianceFrameworkResponse
	if _, err = api.SendGraphQLRequest(ctx, r.client, query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to set project compliance framework: %s", err.Error()))
		return
	}

	// error if 0 or >1 project compliance frameworks were returned
	if len(response.Data.ProjectSetComplianceFramework.Project.ComplianceFrameworks.Nodes) != 1 {
		resp.Diagnostics.AddError("Project Compliance Framework not found", fmt.Sprintf("Unable to find Compliance Framework: %s on project: %s", frameworkId, project.PathWithNamespace))
		return
	}

	// Create resource ID and persist in state model
	// id will always match the project attribute
	data.Id = data.Project

	// persist API response in state model
	r.projectComplianceFrameworkToStateModel(&response.Data.ProjectSetComplianceFramework.Project, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "set a project compliance framework", map[string]interface{}{
		"id": data.Id.ValueString(), "project": data.Project.ValueString(), "compliance_framework_id": data.ComplianceFrameworkId.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabProjectComplianceFrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

// Delete removes the resource.
func (r *gitlabProjectComplianceFrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectComplianceFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The project attribute may either be a numerical project id or the full path,
	// thus we always resolve it to a project to gather the full path for subsequent GraphQL API calls which ALWAYS
	// require the full path ...
	project, _, err := r.client.Projects.GetProject(data.Id.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "project does not exist, removing resource from state", map[string]interface{}{
				"project": data.Id.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project details: %s", err.Error()))
		return
	}

	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				projectSetComplianceFramework(
					input: {
						projectId: "gid://gitlab/Project/%d",
						complianceFrameworkId: null
					}
				) {
					project {
						id,
						complianceFrameworks {
							nodes {
								id
							}
						}
					}
					errors
				}
			}`, project.ID),
	}

	tflog.Debug(ctx, "executing GraphQL Query to update project compliance framework", map[string]interface{}{
		"query": query.Query,
	})

	if _, err = api.SendGraphQLRequest(ctx, r.client, query, nil); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete project compliance framework: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectComplianceFrameworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type projectSetComplianceFrameworkResponse struct {
	Data struct {
		ProjectSetComplianceFramework struct {
			Project graphQLProject `json:"project"`
		} `json:"projectSetComplianceFramework"`
	} `json:"data"`
}

type projectResponse struct {
	Data struct {
		Project graphQLProject `json:"project"`
	} `json:"data"`
}

type graphQLProject struct {
	ProjectId            string `json:"id"`
	FullPath             string `json:"fullPath"`
	ComplianceFrameworks struct {
		Nodes []struct {
			ID string `json:"id"` // This comes back as a globally unique ID
		} `json:"nodes"`
	} `json:"complianceFrameworks"`
}
