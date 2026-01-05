package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ resource.Resource                   = &gitlabProjectPullMirrorResource{}
	_ resource.ResourceWithConfigure      = &gitlabProjectPullMirrorResource{}
	_ resource.ResourceWithImportState    = &gitlabProjectPullMirrorResource{}
	_ resource.ResourceWithValidateConfig = &gitlabProjectPullMirrorResource{}
)

func init() {
	registerResource(NewGitLabProjectPullMirrorResource)
}

func NewGitLabProjectPullMirrorResource() resource.Resource {
	return &gitlabProjectPullMirrorResource{}
}

type gitlabProjectPullMirrorResourceModel struct {
	ID                               types.String `tfsdk:"id"`
	Project                          types.String `tfsdk:"project"`
	MirrorID                         types.Int64  `tfsdk:"mirror_id"`
	URL                              types.String `tfsdk:"url"`
	Enabled                          types.Bool   `tfsdk:"enabled"`
	AuthUser                         types.String `tfsdk:"auth_user"`
	AuthPassword                     types.String `tfsdk:"auth_password"`
	MirrorTriggerBuilds              types.Bool   `tfsdk:"mirror_trigger_builds"`
	OnlyMirrorProtectedBranches      types.Bool   `tfsdk:"only_mirror_protected_branches"`
	MirrorOverwritesDivergedBranches types.Bool   `tfsdk:"mirror_overwrites_diverged_branches"`
	MirrorBranchRegex                types.String `tfsdk:"mirror_branch_regex"`
	LastError                        types.String `tfsdk:"last_error"`
	LastSuccessfulUpdateAt           types.String `tfsdk:"last_successful_update_at"`
	LastUpdateAt                     types.String `tfsdk:"last_update_at"`
	LastUpdateStartedAt              types.String `tfsdk:"last_update_started_at"`
	UpdateStatus                     types.String `tfsdk:"update_status"`
}

type gitlabProjectPullMirrorResource struct {
	client *gitlab.Client
}

func (r *gitlabProjectPullMirrorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_pull_mirror"
}

func (r *gitlabProjectPullMirrorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_pull_mirror`" + ` resource allows managing pull mirroring for GitLab projects.

This resource uses the dedicated pull mirror API endpoint which provides reliable configuration of pull mirroring after project creation.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_pull_mirroring/#configure-pull-mirroring-for-a-project)`,

		Attributes: map[string]schema.Attribute{
			// Note - it may seem weird to use the project ID here instead of the mirror ID, but that's because
			// when performing the "Read" operation (used for import), we use the project, not the mirror ID.
			// As a result, this _must_ be project.
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project owned by the authenticated user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL of the remote repository to mirror from. " +
					"While the API call allows including username and password as basic authentication in the URL, this resource" +
					"does not for security and idempotency reasons. Use `auth_user` and `auth_password` instead.",
				Required:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				// Additional validation for this attribute is done in the ValidateConfig function.
				// See there for additional details.
			},
			"mirror_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the pull mirror. This ID is set by GitLab.",
				Computed:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable or disable the pull mirror.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"auth_user": schema.StringAttribute{
				MarkdownDescription: "Authentication username for the remote repository.",
				Optional:            true,
			},
			"auth_password": schema.StringAttribute{
				MarkdownDescription: "Authentication password or token for the remote repository.",
				Optional:            true,
				Sensitive:           true,
			},
			"mirror_trigger_builds": schema.BoolAttribute{
				MarkdownDescription: "Trigger CI/CD pipelines when the mirror updates.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"only_mirror_protected_branches": schema.BoolAttribute{
				MarkdownDescription: "Mirror only protected branches. Cannot be used with `mirror_branch_regex`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRoot("mirror_branch_regex")),
				},
			},
			"mirror_branch_regex": schema.StringAttribute{
				MarkdownDescription: "Regular expression for branches to mirror. Requires GitLab Premium or Ultimate. Cannot be used with `only_mirror_protected_branches`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("only_mirror_protected_branches")),
				},
			},
			"mirror_overwrites_diverged_branches": schema.BoolAttribute{
				MarkdownDescription: "Overwrite diverged branches on the target project.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"last_error": schema.StringAttribute{
				MarkdownDescription: "Last error message from the mirror operation.",
				Computed:            true,
			},
			"last_successful_update_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last successful mirror update.",
				Computed:            true,
			},
			"last_update_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last mirror update attempt.",
				Computed:            true,
			},
			"last_update_started_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the last mirror update started.",
				Computed:            true,
			},
			"update_status": schema.StringAttribute{
				MarkdownDescription: "Current status of the mirror update.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabProjectPullMirrorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data gitlabProjectPullMirrorResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the URL present in the data URL and ensure it doesn't have a username or password included in it
	url, err := url.Parse(data.URL.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse URL", err.Error())
		return
	}

	if url.User != nil {
		resp.Diagnostics.AddError("URL contains username or password", "The URL should not contain a username or password. Use `auth_user` and `auth_password` instead.")
		return
	}
}

func (r *gitlabProjectPullMirrorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectPullMirrorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabProjectPullMirrorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()

	diags := r.updateProjectPullMirrorConfig(ctx, project, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	// Set the ID - everything else is set by `updateProjectPullMirrorConfig`
	data.ID = types.StringValue(project)

	// Trigger the mirror to start
	_, err := r.client.Projects.StartMirroringProject(project, gitlab.WithContext(ctx))
	if err != nil {
		tflog.Error(ctx, "failed to start mirroring after configuration", map[string]interface{}{
			"project": project,
			"error":   err.Error(),
		})
		resp.Diagnostics.AddError("Failed to start mirroring after configuration. The mirroring is configured, but failed to start.", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectPullMirrorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabProjectPullMirrorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use ID if project is not set (import case)
	project := data.Project.ValueString()
	if project == "" {
		project = data.ID.ValueString()
	}

	tflog.Debug(ctx, "reading gitlab project pull mirror", map[string]interface{}{
		"project": project,
	})

	mirror, _, err := r.client.Projects.GetProjectPullMirrorDetails(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "pull mirror not found, removing from state", map[string]interface{}{
				"project": project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		// Check if the error is "project is not mirrored" - this happens when mirror is disabled
		if errResp, ok := err.(*gitlab.ErrorResponse); ok && errResp.Response != nil && errResp.Response.StatusCode == 400 {
			// If the mirror is disabled in state, this is expected - keep the state as-is
			if !data.Enabled.IsNull() && !data.Enabled.ValueBool() {
				tflog.Debug(ctx, "pull mirror is disabled, keeping state", map[string]interface{}{
					"project": project,
				})
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
			// Otherwise, remove from state
			tflog.Warn(ctx, "pull mirror not configured, removing from state", map[string]interface{}{
				"project": project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to read project pull mirror",
			fmt.Sprintf("Error reading pull mirror for project %s: %s", project, err.Error()),
		)
		return
	}

	// Set project if it wasn't set (import case)
	if data.Project.IsNull() {
		data.Project = types.StringValue(project)
	}

	r.mapMirrorToModel(mirror, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectPullMirrorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabProjectPullMirrorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()

	diags := r.updateProjectPullMirrorConfig(ctx, project, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Note - we don't need to re-trigger the start since this is an update.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectPullMirrorResource) updateProjectPullMirrorConfig(ctx context.Context, project string, data *gitlabProjectPullMirrorResourceModel) diag.Diagnostics {
	options := &gitlab.ConfigureProjectPullMirrorOptions{
		URL: gitlab.Ptr(data.URL.ValueString()),
	}

	if !data.Enabled.IsNull() && !data.Enabled.IsUnknown() {
		options.Enabled = data.Enabled.ValueBoolPointer()
	}

	if !data.AuthUser.IsNull() && !data.AuthUser.IsUnknown() {
		options.AuthUser = gitlab.Ptr(data.AuthUser.ValueString())
	}

	if !data.AuthPassword.IsNull() && !data.AuthPassword.IsUnknown() {
		options.AuthPassword = gitlab.Ptr(data.AuthPassword.ValueString())
	}

	if !data.MirrorTriggerBuilds.IsNull() && !data.MirrorTriggerBuilds.IsUnknown() {
		options.MirrorTriggerBuilds = data.MirrorTriggerBuilds.ValueBoolPointer()
	}

	if !data.OnlyMirrorProtectedBranches.IsNull() && !data.OnlyMirrorProtectedBranches.IsUnknown() {
		options.OnlyMirrorProtectedBranches = data.OnlyMirrorProtectedBranches.ValueBoolPointer()
	}

	if !data.MirrorOverwritesDivergedBranches.IsNull() && !data.MirrorOverwritesDivergedBranches.IsUnknown() {
		options.MirrorOverwritesDivergedBranches = data.MirrorOverwritesDivergedBranches.ValueBoolPointer()
	}

	if !data.MirrorBranchRegex.IsNull() && !data.MirrorBranchRegex.IsUnknown() {
		options.MirrorBranchRegex = gitlab.Ptr(data.MirrorBranchRegex.ValueString())
	}

	tflog.Debug(ctx, "updating gitlab project pull mirror", map[string]interface{}{
		"project": project,
	})

	mirror, _, err := r.client.Projects.ConfigureProjectPullMirror(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("Failed to configure project pull mirror", fmt.Sprintf("Error encountered: %v", err)),
		}
	}
	// Set the response in the data payload
	r.mapMirrorToModel(mirror, data)

	return nil
}

func (r *gitlabProjectPullMirrorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabProjectPullMirrorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()

	options := &gitlab.ConfigureProjectPullMirrorOptions{
		URL:     gitlab.Ptr(data.URL.ValueString()),
		Enabled: gitlab.Ptr(false),
	}

	tflog.Debug(ctx, "disabling gitlab project pull mirror", map[string]interface{}{
		"project": project,
	})

	_, _, err := r.client.Projects.ConfigureProjectPullMirror(project, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to disable project pull mirror",
			fmt.Sprintf("Error disabling pull mirror for project %s: %s", project, err.Error()),
		)
		return
	}
}

func (r *gitlabProjectPullMirrorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to map the mirror to the model
//
// Note - does not map the ID, since that's only set on creation.
func (r *gitlabProjectPullMirrorResource) mapMirrorToModel(mirror *gitlab.ProjectPullMirrorDetails, data *gitlabProjectPullMirrorResourceModel) {
	data.MirrorID = types.Int64Value(int64(mirror.ID))

	if mirror.URL != "" {
		data.URL = types.StringValue(mirror.URL)
	} else {
		data.URL = types.StringNull()
	}

	if mirror.LastError != "" {
		data.LastError = types.StringValue(mirror.LastError)
	} else {
		data.LastError = types.StringNull()
	}

	if mirror.LastSuccessfulUpdateAt != nil {
		data.LastSuccessfulUpdateAt = types.StringValue(mirror.LastSuccessfulUpdateAt.String())
	} else {
		data.LastSuccessfulUpdateAt = types.StringNull()
	}

	if mirror.LastUpdateAt != nil {
		data.LastUpdateAt = types.StringValue(mirror.LastUpdateAt.String())
	} else {
		data.LastUpdateAt = types.StringNull()
	}

	if mirror.LastUpdateStartedAt != nil {
		data.LastUpdateStartedAt = types.StringValue(mirror.LastUpdateStartedAt.String())
	} else {
		data.LastUpdateStartedAt = types.StringNull()
	}

	if mirror.UpdateStatus != "" {
		data.UpdateStatus = types.StringValue(mirror.UpdateStatus)
	} else {
		data.UpdateStatus = types.StringNull()
	}

	// TODO - update client-go to support additional attributes.
}
