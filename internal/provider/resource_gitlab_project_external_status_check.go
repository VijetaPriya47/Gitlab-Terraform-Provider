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
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabProjectExternalStatusCheckResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectExternalStatusCheckResource{}
	_ resource.ResourceWithImportState = &gitlabProjectExternalStatusCheckResource{}
)

func init() {
	registerResource(NewGitLabProjectExternalStatusCheckResource)
}

func NewGitLabProjectExternalStatusCheckResource() resource.Resource {
	return &gitlabProjectExternalStatusCheckResource{}
}

type gitlabProjectExternalStatusCheckResource struct {
	client *gitlab.Client
}

type gitlabProjectExternalStatusCheckResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.Int64  `tfsdk:"project_id"`
	Name               types.String `tfsdk:"name"`
	ExternalURL        types.String `tfsdk:"external_url"`
	SharedSecret       types.String `tfsdk:"shared_secret"`
	HMAC               types.Bool   `tfsdk:"hmac"`
	ProtectedBranchIDs types.Set    `tfsdk:"protected_branch_ids"`
}

// Metadata returns the resource name
func (d *gitlabProjectExternalStatusCheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_external_status_check"
}

func (r *gitlabProjectExternalStatusCheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_external_status_check`" + ` resource allows you to manage the lifecycle of an external status check service on a project.

-> This resource requires a GitLab Enterprise instance with an Ultimate license.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/status_checks/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project-id>:<external-check-id>`",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The display name of the external status check service.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"external_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the external status check service.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"shared_secret": schema.StringAttribute{
				MarkdownDescription: "The HMAC secret for the external status check.  If this is set, then removed from the config, the value will get set to empty in the state.",
				Optional:            true,
				Sensitive:           true,
			},
			"hmac": schema.BoolAttribute{
				MarkdownDescription: "True if the external status check uses an HMAC secret.",
				Computed:            true,
			},
			"protected_branch_ids": schema.SetAttribute{
				MarkdownDescription: "The list of IDs of protected branches to scope the rule by.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.Int64Type,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectExternalStatusCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectExternalStatusCheckResource) projectExternalStatusCheckToStateModel(ctx context.Context, projectID int, externalStatusCheck *gitlab.ProjectStatusCheck, data *gitlabProjectExternalStatusCheckResourceModel) {
	data.ID = types.StringValue(utils.BuildTwoPartID(gitlab.Ptr(strconv.Itoa(projectID)), gitlab.Ptr(strconv.Itoa(externalStatusCheck.ID))))
	data.ProjectID = types.Int64Value(int64(projectID))
	data.Name = types.StringValue(externalStatusCheck.Name)
	data.ExternalURL = types.StringValue(externalStatusCheck.ExternalURL)
	data.HMAC = types.BoolValue(externalStatusCheck.HMAC)

	protectedBranchIDs := []types.Int64{}
	for _, branch := range externalStatusCheck.ProtectedBranches {
		protectedBranchIDs = append(protectedBranchIDs, types.Int64Value(int64(branch.ID)))
	}
	protectedBranchesSetType, _ := types.SetValueFrom(ctx, types.Int64Type, protectedBranchIDs)
	data.ProtectedBranchIDs = protectedBranchesSetType
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabProjectExternalStatusCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectExternalStatusCheckResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID, externalCheckID, err := parseProjectExternalStatusCheckID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	externalStatusCheck, err := findProjectExternalStatusCheck(r.client, projectID, externalCheckID)
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "project external status check does not exist, removing from state", map[string]any{
				"project": projectID, "external_status_check": externalCheckID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project external status check: %s", err.Error()))
		return
	}

	// persist API response in state model
	r.projectExternalStatusCheckToStateModel(ctx, projectID, externalStatusCheck, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create creates a new upstream resource and adds it into the Terraform state.
func (r *gitlabProjectExternalStatusCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectExternalStatusCheckResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := int(data.ProjectID.ValueInt64())

	options := gitlab.CreateProjectExternalStatusCheckOptions{
		Name:        data.Name.ValueStringPointer(),
		ExternalURL: data.ExternalURL.ValueStringPointer(),
	}

	if !data.SharedSecret.IsNull() && !data.SharedSecret.IsUnknown() {
		options.SharedSecret = data.SharedSecret.ValueStringPointer()
	}

	if !data.ProtectedBranchIDs.IsNull() && !data.ProtectedBranchIDs.IsUnknown() {
		// convert the Set to a []int and pass it in
		var protectedBranchIDs []int
		data.ProtectedBranchIDs.ElementsAs(ctx, &protectedBranchIDs, true)
		options.ProtectedBranchIDs = &protectedBranchIDs

	}

	externalStatusCheck, _, err := r.client.ExternalStatusChecks.CreateProjectExternalStatusCheck(projectID, &options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create project external status check: %s", err.Error()))
		return
	}

	// persist API response in state model
	r.projectExternalStatusCheckToStateModel(ctx, projectID, externalStatusCheck, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabProjectExternalStatusCheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectExternalStatusCheckResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID, externalCheckID, err := parseProjectExternalStatusCheckID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	options := &gitlab.UpdateProjectExternalStatusCheckOptions{}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		options.Name = data.Name.ValueStringPointer()
	}

	if !data.ExternalURL.IsNull() && !data.ExternalURL.IsUnknown() {
		options.ExternalURL = data.ExternalURL.ValueStringPointer()
	}

	// If the config does not have the secret defined or it's unknown, set it to empty so HMAC will get set appropriately
	if data.SharedSecret.IsNull() || data.SharedSecret.IsUnknown() {
		options.SharedSecret = new(string)
	} else {
		options.SharedSecret = data.SharedSecret.ValueStringPointer()
	}

	if !data.ProtectedBranchIDs.IsNull() && !data.ProtectedBranchIDs.IsUnknown() {
		// convert the Set to a []int and pass it in
		var protectedBranchIDs []int
		data.ProtectedBranchIDs.ElementsAs(ctx, &protectedBranchIDs, true)
		options.ProtectedBranchIDs = &protectedBranchIDs
	}

	externalStatusCheck, _, err := r.client.ExternalStatusChecks.UpdateProjectExternalStatusCheck(projectID, externalCheckID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update project external status check: %s", err.Error()))
		return
	}

	// persist API response in state model
	r.projectExternalStatusCheckToStateModel(ctx, projectID, externalStatusCheck, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the resource.
func (r *gitlabProjectExternalStatusCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectExternalStatusCheckResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID, externalCheckID, err := parseProjectExternalStatusCheckID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	_, err = r.client.ExternalStatusChecks.DeleteProjectExternalStatusCheck(projectID, externalCheckID, &gitlab.DeleteProjectExternalStatusCheckOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete project external status check: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectExternalStatusCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func parseProjectExternalStatusCheckID(id string) (int, int, error) {
	projectID, externalCheckID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return 0, 0, err
	}

	// Make sure the project ID is an int
	projectIDInt, err := strconv.Atoi(projectID)
	if err != nil {
		return 0, 0, err
	}

	// Make sure the external check ID is an int
	externalCheckIDInt, err := strconv.Atoi(externalCheckID)
	if err != nil {
		return 0, 0, err
	}

	return projectIDInt, externalCheckIDInt, nil
}

func findProjectExternalStatusCheck(client *gitlab.Client, projectID int, externalStatusCheckID int) (*gitlab.ProjectStatusCheck, error) {
	options := gitlab.ListProjectExternalStatusChecksOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}

	for options.Page != 0 {
		statusChecks, resp, err := client.ExternalStatusChecks.ListProjectExternalStatusChecks(projectID, &options)
		if err != nil {
			return nil, fmt.Errorf("unable to list project external status checks. %s", err)
		}

		for i := range statusChecks {
			if statusChecks[i].ID == externalStatusCheckID {
				return statusChecks[i], nil
			}
		}

		options.Page = resp.NextPage
	}

	// if we did not find the project external status check, we should error
	return nil, fmt.Errorf("404 Not Found")
}
