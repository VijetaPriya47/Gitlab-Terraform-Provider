package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabProjectComplianceFrameworksResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectComplianceFrameworksResource{}
	_ resource.ResourceWithImportState = &gitlabProjectComplianceFrameworksResource{}
)

func init() {
	registerResource(NewGitLabProjectComplianceFrameworksResource)
}

func NewGitLabProjectComplianceFrameworksResource() resource.Resource {
	return &gitlabProjectComplianceFrameworksResource{}
}

type gitlabProjectComplianceFrameworksResource struct {
	client *gitlab.Client
}

type gitlabProjectComplianceFrameworksResourceModel struct {
	Id                     types.String   `tfsdk:"id"`
	ComplianceFrameworkIds []types.String `tfsdk:"compliance_framework_ids"`
	Project                types.String   `tfsdk:"project"`
}

// Metadata returns the resource name
func (d *gitlabProjectComplianceFrameworksResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_compliance_frameworks"
}

func (r *gitlabProjectComplianceFrameworksResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_compliance_frameworks`" + ` resource allows to manage the lifecycle of compliance frameworks on a project.

-> This resource requires a GitLab Enterprise instance with a Premium license to set the compliance frameworks on a project.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/ee/api/graphql/reference/#mutationprojectupdatecomplianceframeworks)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"compliance_framework_ids": schema.SetAttribute{
				MarkdownDescription: "Globally unique IDs of the compliance frameworks to assign to the project.",
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.RequiresReplace()},
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`^gid:\/\/gitlab\/ComplianceManagement::Framework\/\d+$`), "value must be a valid compliance framework global identifier")),
				},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project to change the compliance frameworks of.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectComplianceFrameworksResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectComplianceFrameworksResource) projectComplianceFrameworksToStateModel(response *graphQLProject, data *gitlabProjectComplianceFrameworksResourceModel) {
	data.Project = data.Id

	// set the compliance frameworks in the model
	frameworks := make([]types.String, 0, len(response.ComplianceFrameworks.Nodes))
	for _, v := range response.ComplianceFrameworks.Nodes {
		frameworks = append(frameworks, types.StringValue(v.ID))
	}
	data.ComplianceFrameworkIds = frameworks
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabProjectComplianceFrameworksResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectComplianceFrameworksResourceModel

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
	tflog.Debug(ctx, "executing GraphQL Query to retrieve current compliance frameworks on project", map[string]interface{}{
		"query": query.Query,
	})

	var response projectResponse
	if _, err = api.SendGraphQLRequest(ctx, r.client, query, &response); err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "compliance frameworks do not exist on project, removing from state", map[string]interface{}{
				"project_path_with_namespace": project.PathWithNamespace,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project compliance framework details: %s", err.Error()))
		return
	}

	// error if no project compliance frameworks were returned
	if len(response.Data.Project.ComplianceFrameworks.Nodes) == 0 {
		tflog.Debug(ctx, "compliance frameworks do not exist on project, removing from state", map[string]interface{}{
			"project_path_with_namespace": project.PathWithNamespace,
		})
		resp.State.RemoveResource(ctx)
		resp.Diagnostics.AddError("Project Compliance Framework not found", fmt.Sprintf("Unable to find Compliance Frameworks on project: %s", project.PathWithNamespace))
		return
	}

	// persist API response in state model
	r.projectComplianceFrameworksToStateModel(&response.Data.Project, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create creates a new upstream resource and adds it into the Terraform state.
func (r *gitlabProjectComplianceFrameworksResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectComplianceFrameworksResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// convert data.ComplianceFrameworkIds into string list for mutation
	frameworks := make([]string, len(data.ComplianceFrameworkIds))
	for i, v := range data.ComplianceFrameworkIds {
		frameworks[i] = v.ValueString()
	}

	frameworksStr := fmt.Sprintf(`["%s"]`, strings.Join(frameworks, `","`))

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
				projectUpdateComplianceFrameworks(
					input: {
						projectId: "gid://gitlab/Project/%d",
						complianceFrameworkIds: %s
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
			}`, project.ID, frameworksStr),
	}
	tflog.Debug(ctx, "executing GraphQL Query to update project compliance frameworks", map[string]interface{}{
		"query": query.Query,
	})

	var response projectUpdateComplianceFrameworksResponse
	if _, err = api.SendGraphQLRequest(ctx, r.client, query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update project compliance frameworks: %s", err.Error()))
		return
	}

	// error if 0 project compliance frameworks were returned
	if len(response.Data.ProjectUpdateComplianceFrameworks.Project.ComplianceFrameworks.Nodes) == 0 {
		resp.Diagnostics.AddError("Project Compliance Frameworks not found", fmt.Sprintf("Unable to find Compliance Frameworks: %s on project: %s", frameworksStr, project.PathWithNamespace))
		return
	}

	// Create resource ID and persist in state model
	// id will always match the project attribute
	data.Id = data.Project

	// persist API response in state model
	r.projectComplianceFrameworksToStateModel(&response.Data.ProjectUpdateComplianceFrameworks.Project, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "update project compliance frameworks", map[string]interface{}{
		"id": data.Id.ValueString(), "project": data.Project.ValueString(), "compliance_framework_ids": frameworksStr,
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabProjectComplianceFrameworksResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

// Delete removes the resource.
func (r *gitlabProjectComplianceFrameworksResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectComplianceFrameworksResourceModel

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
				projectUpdateComplianceFrameworks(
					input: {
						projectId: "gid://gitlab/Project/%d",
						complianceFrameworkIds: null
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

	tflog.Debug(ctx, "executing GraphQL Query to update project compliance frameworks", map[string]interface{}{
		"query": query.Query,
	})

	if _, err = api.SendGraphQLRequest(ctx, r.client, query, nil); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete project compliance frameworks: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectComplianceFrameworksResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type projectUpdateComplianceFrameworksResponse struct {
	Data struct {
		ProjectUpdateComplianceFrameworks struct {
			Project graphQLProject `json:"project"`
		} `json:"projectUpdateComplianceFrameworks"`
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
