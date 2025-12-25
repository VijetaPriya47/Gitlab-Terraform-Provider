package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                 = &gitlabProjectJobTokenScopeResource{}
	_ resource.ResourceWithConfigure    = &gitlabProjectJobTokenScopeResource{}
	_ resource.ResourceWithImportState  = &gitlabProjectJobTokenScopeResource{}
	_ resource.ResourceWithUpgradeState = &gitlabProjectJobTokenScopeResource{}
)

const (
	targetTypeProject = "project"
	targetTypeGroup   = "group"
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
	TargetGroupID   types.Int64  `tfsdk:"target_group_id"`
}

// gitlabProjectJobTokenScopeResourceModelSchema0 represents the V0 version of the resource schema
type gitlabProjectJobTokenScopeResourceModelSchema0 struct {
	Id              types.String `tfsdk:"id"`
	Project         types.String `tfsdk:"project"`
	TargetProjectID types.Int64  `tfsdk:"target_project_id"`
}

func (r *gitlabProjectJobTokenScopeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_job_token_scope"
}

func (d *gitlabProjectJobTokenScopeResource) getV1Schema(resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_job_token_scope`" + ` resource allows to manage the CI/CD Job Token scope in a project.
Any projects added to the CI/CD Job Token scope outside of TF will be untouched by the resource.

~> Conflicts with the use of ` + "`gitlab_project_job_token_scopes`" + ` when used on the same project. Use one or the other to ensure the desired state.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_job_token_scopes/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<type>:<target-id>` where `type` is either `project` or `group`.",
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
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.ExactlyOneOf(path.MatchRoot("target_group_id")),
				},
			},
			"target_group_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the group that is in the CI/CD job token inbound allowlist.",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.ExactlyOneOf(path.MatchRoot("target_project_id")),
				},
			},
		},
		Version: 1,
	}
}

// Schema returns the schema for the resource.
func (r *gitlabProjectJobTokenScopeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	r.getV1Schema(resp)
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectJobTokenScopeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
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

	var targetIDStr string
	var targetType string

	// Add the target project or group to the CI/CD Job Token inbound allowlist
	switch {
	case !data.TargetProjectID.IsNull():
		targetID := data.TargetProjectID.ValueInt64()
		options := &gitlab.JobTokenInboundAllowOptions{TargetProjectID: gitlab.Ptr(targetID)}
		addTokenResponse, _, err := r.client.JobTokenScope.AddProjectToJobScopeAllowList(projectID, options, gitlab.WithContext(ctx))

		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to add the target project to CI/CD Job Token inbound allowlist: %s", err.Error()))
			return
		}
		targetIDStr = strconv.FormatInt(addTokenResponse.TargetProjectID, 10)
		targetType = targetTypeProject

		// Log the creation of the resource
		tflog.Debug(ctx, "Added the target project to CI/CD Job Token inbound allowlist", map[string]any{
			"project": projectID, "target_project": targetIDStr,
		})

	case !data.TargetGroupID.IsNull():
		targetID := data.TargetGroupID.ValueInt64()
		options := &gitlab.AddGroupToJobTokenAllowlistOptions{TargetGroupID: gitlab.Ptr(targetID)}
		addTokenResponse, _, err := r.client.JobTokenScope.AddGroupToJobTokenAllowlist(projectID, options, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to add the target group to CI/CD Job Token inbound allowlist: %s", err.Error()))
			return
		}
		targetIDStr = strconv.FormatInt(addTokenResponse.TargetGroupID, 10)
		targetType = targetTypeGroup
		// Log the creation of the resource
		tflog.Debug(ctx, "Added the target group to CI/CD Job Token inbound allowlist", map[string]any{
			"project": projectID, "target_group": targetIDStr,
		})
	}

	// Create resource ID and persist in state model
	data.Id = types.StringValue(fmt.Sprintf("%s:%s:%s", projectID, targetType, targetIDStr))

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

	projectID, targetType, _, err := utils.ParseThreePartID(data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<target-id>'. Error: %s", data.Id.ValueString(), err.Error()))
		return
	}

	var found bool

	// Check which type of target exists
	switch targetType {
	case targetTypeProject:
		targetID := data.TargetProjectID.ValueInt64()
		err, found = findProjectJobTokenInboundAllowlist(r.client, ctx, projectID, targetID)
	case targetTypeGroup:
		targetID := data.TargetGroupID.ValueInt64()
		err, found = findGroupJobTokenInboundAllowlist(r.client, ctx, projectID, targetID)
	}
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project allowlist: %s", err.Error()))
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

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

	projectID, targetType, _, err := utils.ParseThreePartID(data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format. Error: %s", data.Id.ValueString(), err.Error()))
		return
	}

	switch targetType {
	case targetTypeProject:
		targetID := data.TargetProjectID.ValueInt64()
		_, err = r.client.JobTokenScope.RemoveProjectFromJobScopeAllowList(projectID, targetID, gitlab.WithContext(ctx))
	case targetTypeGroup:
		targetID := data.TargetGroupID.ValueInt64()
		_, err = r.client.JobTokenScope.RemoveGroupFromJobTokenAllowlist(projectID, targetID, gitlab.WithContext(ctx))
	}

	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to remove target from CI/CD Job Token inbound allowlist: %s", err.Error()))
		return
	}
}

// GetV0Schema returns the schema for the V0 version of the resource
func (r *gitlabProjectJobTokenScopeResource) getV0Schema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_job_token_scope`" + ` resource allows to manage the CI/CD Job Token scope in a project.
Any projects added to the CI/CD Job Token scope outside of TF will be untouched by the resource.

~> Conflicts with the use of ` + "`gitlab_project_job_token_scopes`" + ` when used on the same project. Use one or the other to ensure the desired state.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_job_token_scopes/)`,

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
				Required:            true, // Changed from Optional to Required since it was the only option in V0
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
		Version: 0,
	}
}

// UpgradeState upgrades the resource state from V0 to V1 by adding the target type to the ID.
func resourceGitlabProjectJobTokenScopeStateUpgradeV0ToV1(ctx context.Context, data *gitlabProjectJobTokenScopeResourceModelSchema0) (*gitlabProjectJobTokenScopeResourceModel, error) {
	projectID, targetIDStr, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		return nil, fmt.Errorf("unable to parse two part ID: %s", err.Error())
	}

	// Target type will always be project because v0 only supported projects
	targetType := targetTypeProject

	tflog.Debug(ctx, "migrating state from V0 to V1", map[string]any{
		"old_id": data.Id.ValueString(),
		"type":   targetType,
	})

	newID := fmt.Sprintf("%s:%s:%s", projectID, targetType, targetIDStr)

	newData := &gitlabProjectJobTokenScopeResourceModel{
		Id:              types.StringValue(newID),
		Project:         data.Project,
		TargetProjectID: data.TargetProjectID,
	}
	return newData, nil
}

// UpgradeState upgrades the resource state from V0 to V1
func (r *gitlabProjectJobTokenScopeResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	schema := r.getV0Schema()

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data gitlabProjectJobTokenScopeResourceModelSchema0
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}
				newData, err := resourceGitlabProjectJobTokenScopeStateUpgradeV0ToV1(ctx, &data)
				if err != nil {
					resp.Diagnostics.AddError("Unable to upgrade state", err.Error())
					return
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &newData)...)
			},
		},
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabProjectJobTokenScopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Split the import ID into its components and try to parse it as a three part ID
	projectID, targetType, targetIDStr, err := utils.ParseThreePartID(req.ID)
	if err != nil {
		// If the three part ID fails, try to parse it as a two part ID
		projectID, targetIDStr, err = utils.ParseTwoPartID(req.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid import ID format",
				fmt.Sprintf("The resource ID must be in the format 'project:type:target_id' or the legacy format 'project:target_id'. Got: %s. Error: %s", req.ID, err.Error()),
			)
			return
		}
		// For legacy IDs, assume the target type is project (as we do not have the group target in V0)
		targetType = targetTypeProject
	}

	// Convert target ID to int for validation
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid target ID",
			fmt.Sprintf("Failed to parse target ID '%s' as integer: %s", targetIDStr, err.Error()),
		)
		return
	}

	// Create the new three part ID
	newID := fmt.Sprintf("%s:%s:%s", projectID, targetType, targetIDStr)

	// Set the basic fields
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), newID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), projectID)...)

	// Set the appropriate target ID based on type
	switch targetType {
	case targetTypeProject:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target_project_id"), targetID)...)
	case targetTypeGroup:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target_group_id"), targetID)...)
	default:
		resp.Diagnostics.AddError(
			"Invalid target type",
			fmt.Sprintf("Target type must be either 'project' or 'group'. Got: %s", targetType),
		)
	}
}

// findProjectJobTokenInboundAllowlist finds the target project in the CI/CD Job Token inbound allowlist
func findProjectJobTokenInboundAllowlist(client *gitlab.Client, ctx context.Context, projectID string, targetProjectID int64) (error, bool) {
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

// findGroupJobTokenInboundAllowlist finds the target group in the CI/CD Job Token inbound allowlist
func findGroupJobTokenInboundAllowlist(client *gitlab.Client, ctx context.Context, projectID string, targetGroupID int64) (error, bool) {
	options := gitlab.GetJobTokenAllowlistGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}

	for options.Page != 0 {
		paginatedGroups, resp, err := client.JobTokenScope.GetJobTokenAllowlistGroups(projectID, &options, gitlab.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("unable to read CI/CD Job Token inbound groups_allowlist. %s", err), false
		}

		for i := range paginatedGroups {
			if paginatedGroups[i].ID == targetGroupID {
				return nil, true
			}
		}

		options.Page = resp.NextPage
	}

	return nil, false
}
