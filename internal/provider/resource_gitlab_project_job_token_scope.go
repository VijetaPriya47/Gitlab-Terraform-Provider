package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &gitlabProjectJobTokenScopeResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectJobTokenScopeResource{}
	_ resource.ResourceWithImportState = &gitlabProjectJobTokenScopeResource{}
)

func init() {
	registerResource(NewGitLabProjectJobTokenScopeResource)
}

// NewGitLabProjectJobTokenScopeResource is a helper function to simplify the provider implementation.
func NewGitLabProjectJobTokenScopeResource() resource.Resource {
	return &gitlabProjectJobTokenScopeResource{}
}

// gitlabProjectJobTokenScopeResource defines the resource implementation.
type gitlabProjectJobTokenScopeResource struct {
	client *gitlab.Client
}

// gitlabProjectJobTokenScopeResourceModel describes the resource data model.
type gitlabProjectJobTokenScopeResourceModel struct {
	Id              types.String `tfsdk:"id"`
	Project         types.String `tfsdk:"project"`
	TargetProjectID types.Int64  `tfsdk:"target_project_id"`
}

func (r *gitlabProjectJobTokenScopeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_job_token_scope"
}

func (r *gitlabProjectJobTokenScopeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_job_token_scope`" + ` resource allows to manage the CI/CD Job Token scope in a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/project_job_token_scopes.html)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<target-project-id>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"target_project_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the project that is in the CI/CD job token inbound allowlist.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectJobTokenScopeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

// Create a new upstream resources and adds it into the Terraform state.
func (r *gitlabProjectJobTokenScopeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectJobTokenScopeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// local copies of plan arguments
	projectID := data.Project.ValueString()
	targetProjectID := int(data.TargetProjectID.ValueInt64())

	options := &gitlab.JobTokenInboundAllowOptions{
		TargetProjectID: gitlab.Int(targetProjectID),
	}

	addTokenResponse, _, err := r.client.JobTokenScope.AddProjectToJobScopeAllowList(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to add the target project to CI/CD Job Token inbound allowlist: %s", err.Error()))
		return
	}

	// Create resource ID and persist in state model
	targetProjectIdStr := strconv.Itoa(addTokenResponse.TargetProjectID)
	data.Id = types.StringValue(utils.BuildTwoPartID(&projectID, &targetProjectIdStr))

	// Log the creation of the resource
	tflog.Debug(ctx, "Added the target project to CI/CD Job Token inbound allowlist", map[string]interface{}{
		"project": projectID, "target_project": targetProjectID,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabProjectJobTokenScopeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectJobTokenScopeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	projectID, targetProjectIdStr, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<target-project-id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	targetProjectID, err := strconv.Atoi(targetProjectIdStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("target-project-id %s cannot be converted to int", targetProjectIdStr),
		)
		return
	}

	// Read CI/CD Job Token inbound allowlist
	err, found := findProjectJobTokenInboundAllowlist(r.client, ctx, projectID, targetProjectID)

	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", err.Error())
		return
	}
	if !found {
		tflog.Debug(ctx, fmt.Sprintf("The target project (%d) not found in the CI/CD Job Token inbound allowlist, removing from state", targetProjectID))
		resp.State.RemoveResource(ctx)
		return
	}

	data.Project = types.StringValue(projectID)
	data.TargetProjectID = types.Int64Value(int64(targetProjectID))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Updates updates the resource in-place.
func (r *gitlabProjectJobTokenScopeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

// Deletes removes the resource.
func (r *gitlabProjectJobTokenScopeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectJobTokenScopeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	projectID, targetProjectIdStr, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<target-project-id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	targetProjectID, err := strconv.Atoi(targetProjectIdStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("target-project-id %s cannot be converted to int", targetProjectIdStr),
		)
		return
	}

	_, err = r.client.JobTokenScope.RemoveProjectFromJobScopeAllowList(projectID, targetProjectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "project does not exist, removing resource from state", map[string]interface{}{
				"project":        projectID,
				"target_project": targetProjectIdStr,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to remove the target project to CI/CD Job Token inbound allowlist: %s", err.Error()))
		return
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabProjectJobTokenScopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func findProjectJobTokenInboundAllowlist(client *gitlab.Client, ctx context.Context, projectID string, targetProjectID int) (error, bool) {
	options := gitlab.GetJobTokenInboundAllowListOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}

	for options.Page != 0 {
		paginatedProjects, resp, err := client.JobTokenScope.GetProjectJobTokenInboundAllowList(projectID, &options, gitlab.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("unable to read CI/CD Job Token inbound allowlist. %s", err), false
		}

		for i := range paginatedProjects {
			if paginatedProjects[i].ID == targetProjectID {
				return nil, true
			}
		}

		options.Page = resp.NextPage
	}

	return nil, false
}
