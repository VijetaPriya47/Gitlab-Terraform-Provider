package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dcarbone/terraform-plugin-framework-utils/v3/conv"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &gitlabProjectJobTokenScopesResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectJobTokenScopesResource{}
	_ resource.ResourceWithImportState = &gitlabProjectJobTokenScopesResource{}
)

func init() {
	registerResource(NewGitLabProjectJobTokenScopesResource)
}

// NewGitLabProjectJobTokensResource is a helper function to simplify the provider implementation.
func NewGitLabProjectJobTokenScopesResource() resource.Resource {
	return &gitlabProjectJobTokenScopesResource{}
}

// gitlabProjectJobTokenScopesResource defines the resource implementation.
type gitlabProjectJobTokenScopesResource struct {
	client *gitlab.Client
}

// gitlabProjectJobTokenScopesResourceModel describes the resource data model.
type gitlabProjectJobTokenScopesResourceModel struct {
	Id        types.String `tfsdk:"id"`
	Project   types.String `tfsdk:"project"`
	ProjectID types.Int64  `tfsdk:"project_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`

	// types.Set in the schema
	TargetProjectIDs types.Set `tfsdk:"target_project_ids"`
	TargetGroupIDs   types.Set `tfsdk:"target_group_ids"`
}

func (m *gitlabProjectJobTokenScopesResourceModel) getProject() string {
	if m.Project.ValueString() == "" {
		return strconv.Itoa(int(m.ProjectID.ValueInt64()))
	} else {
		return m.Project.ValueString()
	}
}

func (r *gitlabProjectJobTokenScopesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_job_token_scopes"
}

func (r *gitlabProjectJobTokenScopesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_job_token_scopes`" + ` resource allows to manage the CI/CD Job Token scopes in a project.
Any project or group not within the defined set of ` + "`target_project_ids` or `target_group_ids`" + `, respectively, will be removed,
which allows this resource to be used as an explicit deny.

~> Conflicts with the use of ` + "`gitlab_project_job_token_scope`" + ` when used on the same project. Use one or the other to ensure the desired state.

~> If the ` + "`enabled`" + ` property is false, any project or group will be allowed regardless of the given allowlist attributes.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_job_token_scopes/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project_id>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("project"), path.MatchRelative().AtParent().AtName("project_id")),
				},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1), stringvalidator.ConflictsWith(path.MatchRoot("project_id"))},
			},
			"project_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the project.",
				DeprecationMessage:  "`project_id` has been deprecated. Use `project` instead.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Validators:          []validator.Int64{int64validator.AtLeast(0), int64validator.ConflictsWith(path.MatchRoot("project"))},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable the given inbound allowlist. If false, will allow any project or group regardless of the values in `target_project_ids` or `target_group_ids`. Deleting the associated `gitlab_project_job_token_scopes` resource will reset `Enabled` on the group to `true`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"target_project_ids": schema.SetAttribute{
				MarkdownDescription: "A set of project IDs that are in the CI/CD job token inbound allowlist.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				ElementType: types.Int64Type,
			},
			"target_group_ids": schema.SetAttribute{
				MarkdownDescription: "A set of group IDs that are in the CI/CD job token inbound allowlist.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				ElementType: types.Int64Type,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectJobTokenScopesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Create a new upstream resources and adds it into the Terraform state.
func (r *gitlabProjectJobTokenScopesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectJobTokenScopesResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project := data.getProject()

	// Since a user may have added scopes to a project before this resource was added, we essentially need to do a
	// "diff" operation even in the "create" function
	resp.Diagnostics.Append(r.setProjectCIJobScopes(ctx, project, data))

	// Populate the state model object, since we don't get all projects back in one request, so we have to re-read them
	resp.Diagnostics.Append(r.readIntoState(ctx, project, data))

	data.Id = types.StringValue(project)
	data.Project = types.StringValue(project)
	projectID, err := strconv.Atoi(data.Id.ValueString())
	if err == nil {
		data.ProjectID = types.Int64Value(int64(projectID))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabProjectJobTokenScopesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectJobTokenScopesResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Populate the state model object
	resp.Diagnostics.Append(r.readIntoState(ctx, data.Id.ValueString(), data))

	// Save updated data into Terraform state
	data.Project = types.StringValue(data.Id.ValueString())
	projectID, err := strconv.Atoi(data.Id.ValueString())
	if err == nil {
		data.ProjectID = types.Int64Value(int64(projectID))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabProjectJobTokenScopesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectJobTokenScopesResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project := data.getProject()

	// Since a user may have added scoped to a project before this resource was added, we essentially need to do a
	// "diff" operation even in the "create" function
	resp.Diagnostics.Append(r.setProjectCIJobScopes(ctx, project, data))

	// Populate the state model object, since we don't get all projects back in one request, so we have to re-read them
	resp.Diagnostics.Append(r.readIntoState(ctx, project, data))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the resource.
func (r *gitlabProjectJobTokenScopesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectJobTokenScopesResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// default value for new projects, so restore this default state upon deletion
	data.Enabled = types.BoolValue(true)

	// Set the expected target project Ids to empty
	projectSet, diags := types.SetValueFrom(ctx, types.Int64Type, []types.Int64{})
	resp.Diagnostics.Append(diags...)
	data.TargetProjectIDs = projectSet

	groupSet, diags := types.SetValueFrom(ctx, types.Int64Type, []types.Int64{})
	resp.Diagnostics.Append(diags...)
	data.TargetGroupIDs = groupSet

	project := data.getProject()

	// Run the set to empty out project Ids
	r.setProjectCIJobScopes(ctx, project, data)

	if resp.Diagnostics.HasError() {
		return
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabProjectJobTokenScopesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// setProjectCIJobScopes takes in lists of int64 projects and groups IDs from the data model,
// and sets the project CI token scope to exactly match that list.
// That means it performs the following actions:
//   - Adds any scopes not already on the project
//   - Removes scopes on the project, but not in the list
//   - Leaves all other scopes alone.
//
// Additionally, it sets the enabled flag to the desired value and thus controls if the allowlists apply at all.
func (r *gitlabProjectJobTokenScopesResource) setProjectCIJobScopes(ctx context.Context, project string, data *gitlabProjectJobTokenScopesResourceModel) diag.Diagnostic {
	if diags := r.setAllowedTargetProjects(ctx, project, data); diags != nil {
		return diags
	}

	if diags := r.setAllowedTargetGroups(ctx, project, data); diags != nil {
		return diags
	}

	_, err := r.client.JobTokenScope.PatchProjectJobTokenAccessSettings(project, &gitlab.PatchProjectJobTokenAccessSettingsOptions{Enabled: data.Enabled.ValueBool()})
	if err != nil {
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("GitLab API error occured when setting the jobtoken allowlist enabled flag in project %s", project),
			err.Error(),
		)
	}

	return nil
}

func (r *gitlabProjectJobTokenScopesResource) setAllowedTargetProjects(ctx context.Context, project string, data *gitlabProjectJobTokenScopesResourceModel) diag.Diagnostic {
	// Get a list of existing CI project scopes for the project
	projects, err := r.getProjectCIJobScopes(ctx, project)
	if err != nil {
		return diag.NewErrorDiagnostic(
			"GitLab API error occured when retrieving existing project scopes to compare",
			err.Error(),
		)
	}

	currentProjectsIDs := make([]int, 0, len(projects))
	for _, p := range projects {
		currentProjectsIDs = append(currentProjectsIDs, p.ID)
	}

	createTargetProjects, deleteTargetProjects := r.compareAndGenerateActions(conv.Int64SetToInts(data.TargetProjectIDs), currentProjectsIDs)
	for _, currentProjectID := range deleteTargetProjects {
		_, err := r.client.JobTokenScope.RemoveProjectFromJobScopeAllowList(project, currentProjectID, gitlab.WithContext(ctx))
		if err != nil {
			return diag.NewErrorDiagnostic(
				fmt.Sprintf("GitLab API error occured when removing job scopes from project %s", project),
				err.Error(),
			)
		}
	}
	for _, targetProjectID := range createTargetProjects {
		options := &gitlab.JobTokenInboundAllowOptions{
			TargetProjectID: gitlab.Ptr(targetProjectID),
		}
		_, _, err := r.client.JobTokenScope.AddProjectToJobScopeAllowList(project, options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.NewErrorDiagnostic(
				fmt.Sprintf("GitLab API error occured when adding job scopes to project %s", project),
				err.Error(),
			)
		}
	}
	return nil
}

func (r *gitlabProjectJobTokenScopesResource) setAllowedTargetGroups(ctx context.Context, project string, data *gitlabProjectJobTokenScopesResourceModel) diag.Diagnostic {
	// Get a list of existing CI groups scopes for the project
	groups, err := r.getProjectCIJobScopesGroups(ctx, project)
	if err != nil {
		return diag.NewErrorDiagnostic(
			"GitLab API error occured when retrieving existing groups scopes to compare",
			err.Error(),
		)
	}

	groupsIDs := make([]int, 0, len(groups))
	for _, g := range groups {
		groupsIDs = append(groupsIDs, g.ID)
	}

	createTargetGroups, deleteTargetGroups := r.compareAndGenerateActions(conv.Int64SetToInts(data.TargetGroupIDs), groupsIDs)
	for _, groupID := range deleteTargetGroups {
		_, err := r.client.JobTokenScope.RemoveGroupFromJobTokenAllowlist(project, groupID, gitlab.WithContext(ctx))
		if err != nil {
			return diag.NewErrorDiagnostic(
				fmt.Sprintf("GitLab API error occured when removing group from job scopes for project %s", project),
				err.Error(),
			)
		}
	}
	for _, groupID := range createTargetGroups {
		options := &gitlab.AddGroupToJobTokenAllowlistOptions{
			TargetGroupID: gitlab.Ptr(groupID),
		}
		_, _, err := r.client.JobTokenScope.AddGroupToJobTokenAllowlist(project, options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.NewErrorDiagnostic(
				fmt.Sprintf("GitLab API error occured when adding group to job scopes for project %s", project),
				err.Error(),
			)
		}
	}
	return nil
}

// compareAndGenerateActions compares the difference between the desired slice and the current slice of IDs,
// and returns slices of IDs that need to be created and deleted.
func (r *gitlabProjectJobTokenScopesResource) compareAndGenerateActions(desiredIDs []int, currentIDs []int) (create []int, delete []int) {
	for _, cid := range currentIDs {
		shouldDelete := true
		for _, did := range desiredIDs {
			if did == cid {
				shouldDelete = false
				break
			}
		}
		if shouldDelete {
			delete = append(delete, cid)
		}
	}

	for _, did := range desiredIDs {
		shouldCreate := true
		for _, cid := range currentIDs {
			if cid == did {
				shouldCreate = false
				break
			}
		}

		if shouldCreate {
			create = append(create, did)
		}
	}
	return create, delete
}

// getProjectCIJobScopes retrieves a comprehensive list of CI project scope targets
func (r *gitlabProjectJobTokenScopesResource) getProjectCIJobScopes(ctx context.Context, project string) ([]*gitlab.Project, error) {
	var projectScopes []*gitlab.Project

	// Get a list of existing CI project scopes for the project, 20 pages at once
	options := gitlab.GetJobTokenInboundAllowListOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}

	for options.Page != 0 {
		paginatedProjects, resp, err := r.client.JobTokenScope.GetProjectJobTokenInboundAllowList(project, &options, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("unable to read CI/CD Job Token inbound allowlist. %s", err)
		}

		projectScopes = append(projectScopes, paginatedProjects...)
		options.Page = resp.NextPage

		tflog.Debug(ctx, "Read CI/CD Job Token inbound allowlist for project", map[string]interface{}{
			"project":                      project,
			"page":                         options.Page,
			"number_of_project_identified": len(projectScopes),
		})
	}

	// Remove itself from the list, which will cause issues during the "set" operation, and during post-apply calculations.
	for i, p := range projectScopes {
		if p.PathWithNamespace == project || strconv.Itoa(p.ID) == project {
			projectScopes = append(projectScopes[:i], projectScopes[i+1:]...)
			break
		}
	}

	return projectScopes, nil
}

// getProjectCIJobScopesGroups retrieves a comprehensive list of CI groups scope targets
func (r *gitlabProjectJobTokenScopesResource) getProjectCIJobScopesGroups(ctx context.Context, projectID string) ([]*gitlab.Group, error) {
	var groupsScopes []*gitlab.Group

	options := gitlab.GetJobTokenAllowlistGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}
	for options.Page != 0 {
		paginatedGroups, resp, err := r.client.JobTokenScope.GetJobTokenAllowlistGroups(projectID, &options, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("unable to read CI/CD Job Token inbound groups_allowlist. %s", err)
		}

		groupsScopes = append(groupsScopes, paginatedGroups...)
		options.Page = resp.NextPage

		tflog.Debug(ctx, "Read CI/CD Job Token inbound groups_allowlist for project", map[string]interface{}{
			"project":                     projectID,
			"page":                        options.Page,
			"number_of_groups_identified": len(paginatedGroups),
		})
	}

	return groupsScopes, nil
}

// Retrieves a comprehensive list of CI project scope targets
func (r *gitlabProjectJobTokenScopesResource) readIntoState(ctx context.Context, project string, data *gitlabProjectJobTokenScopesResourceModel) diag.Diagnostic {
	settings, _, err := r.client.JobTokenScope.GetProjectJobTokenAccessSettings(project)
	if err != nil {
		return diag.NewErrorDiagnostic("Error reading project job token access settings", err.Error())
	}
	data.Enabled = types.BoolValue(settings.InboundEnabled)

	// re-read the project IDs from the API to set to state since we don't get them back in one request
	projects, err := r.getProjectCIJobScopes(ctx, project)
	if err != nil {
		return diag.NewErrorDiagnostic("Error reading project scopes", err.Error())
	}

	// build the slice of projects
	projectIds := []types.Int64{}
	for _, p := range projects {
		projectIds = append(projectIds, types.Int64Value(int64(p.ID)))
	}
	// convert the slice to a set, and assign it
	projectIdSet, diags := types.SetValueFrom(ctx, types.Int64Type, projectIds)
	if diags.HasError() {
		return diags[0]
	}
	data.TargetProjectIDs = projectIdSet

	// re-read the group IDs from the API to set to state since we don't get them back in one request
	groups, err := r.getProjectCIJobScopesGroups(ctx, project)
	if err != nil {
		return diag.NewErrorDiagnostic("Error reading groups for project scopes", err.Error())
	}

	// build the slice of groups
	groupIds := []types.Int64{}
	for _, p := range groups {
		groupIds = append(groupIds, types.Int64Value(int64(p.ID)))
	}
	// convert the slice to a set, and assign it
	groupIdSet, diags := types.SetValueFrom(ctx, types.Int64Type, groupIds)
	if diags.HasError() {
		return diags[0]
	}
	data.TargetGroupIDs = groupIdSet
	return nil
}
