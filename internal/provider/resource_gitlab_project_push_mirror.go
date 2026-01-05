package provider

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	_ resource.Resource                = &gitlabProjectMirrorResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectMirrorResource{}
	_ resource.ResourceWithImportState = &gitlabProjectMirrorResource{}
)

// Register the resource with the provider.
func init() {
	registerResource(NewGitLabProjectPushMirrorResource)

	// Register the old alias of the resource
	registerResource(NewGitLabProjectMirrorResource)
}

// NewGitLabProjectMirrorResource returns a new instance of the resource.
func NewGitLabProjectPushMirrorResource() resource.Resource {
	return &gitlabProjectMirrorResource{
		ResourceName: "_project_push_mirror",
	}
}

// NewGitLabProjectMirrorResource returns a new instance of the resource.
// deprecated: This function is deprecated and will be removed in a future version. Use NewGitLabProjectPushMirrorResource instead.
func NewGitLabProjectMirrorResource() resource.Resource {
	return &gitlabProjectMirrorResource{
		ResourceName:       "_project_mirror",
		DeprecationMessage: "This resource is deprecated in favor of `gitlab_project_push_mirror` which is a rename. The alias will be removed in version 19.0.",
	}
}

// gitlabProjectMirrorResourceModel maps the resource schema data.
type gitlabProjectMirrorResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Project               types.String `tfsdk:"project"`
	MirrorID              types.Int64  `tfsdk:"mirror_id"`
	URL                   types.String `tfsdk:"url"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	OnlyProtectedBranches types.Bool   `tfsdk:"only_protected_branches"`
	MirrorBranchRegex     types.String `tfsdk:"mirror_branch_regex"`
	KeepDivergentRefs     types.Bool   `tfsdk:"keep_divergent_refs"`
	AuthMethod            types.String `tfsdk:"auth_method"`
}

// gitlabProjectMirrorResource implements the resource.
type gitlabProjectMirrorResource struct {
	client *gitlab.Client

	ResourceName       string
	DeprecationMessage string
}

// Metadata returns the resource type name.
func (r *gitlabProjectMirrorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

// Schema defines the schema for the resource.
func (r *gitlabProjectMirrorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.getSchema()
}

// ModifyPlan handles the plan modification for the resource.
func (r *gitlabProjectMirrorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var planData, stateData *gitlabProjectMirrorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if planData == nil || stateData == nil || planData.URL.IsNull() || stateData.URL.IsNull() {
		return
	}

	// Compare URLs ignoring credentials
	oldURL, err := url.Parse(stateData.URL.ValueString())
	if err != nil {
		return
	}
	newURL, err := url.Parse(planData.URL.ValueString())
	if err != nil {
		return
	}

	if oldURL.User != nil {
		oldURL.User = url.UserPassword("redacted", "redacted")
	}
	if newURL.User != nil {
		newURL.User = url.UserPassword("redacted", "redacted")
	}

	if oldURL.String() == newURL.String() {
		// Keep the original URL from state to maintain consistency
		planData.URL = stateData.URL
		resp.Diagnostics.Append(resp.Plan.Set(ctx, planData)...)
	} else {
		// URLs are different, mark for replacement
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("url"))
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectMirrorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Create creates a new project mirror.
func (r *gitlabProjectMirrorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectMirrorResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set defaults if not specified
	if data.Enabled.IsNull() {
		data.Enabled = types.BoolValue(true)
	}
	if data.OnlyProtectedBranches.IsNull() && data.MirrorBranchRegex.IsNull() {
		// Set default value to true if mirror_branch_regex is null.
		data.OnlyProtectedBranches = types.BoolValue(true)
	}

	if data.KeepDivergentRefs.IsNull() {
		data.KeepDivergentRefs = types.BoolValue(true)
	}

	// Construct the full URL with credentials
	fullURL := data.URL.ValueString()
	options := &gitlab.AddProjectMirrorOptions{
		URL:               &fullURL,
		Enabled:           data.Enabled.ValueBoolPointer(),
		KeepDivergentRefs: data.KeepDivergentRefs.ValueBoolPointer(),
	}

	if !data.OnlyProtectedBranches.IsNull() && !data.OnlyProtectedBranches.IsUnknown() {
		options.OnlyProtectedBranches = data.OnlyProtectedBranches.ValueBoolPointer()
	}

	if !data.MirrorBranchRegex.IsNull() && !data.MirrorBranchRegex.IsUnknown() {
		options.MirrorBranchRegex = data.MirrorBranchRegex.ValueStringPointer()
	}

	if !data.AuthMethod.IsNull() && !data.AuthMethod.IsUnknown() {
		options.AuthMethod = data.AuthMethod.ValueStringPointer()
	}

	tflog.Debug(ctx, "creating gitlab project mirror for project", map[string]any{
		"project": data.Project,
	})

	mirror, _, err := r.client.ProjectMirrors.AddProjectMirror(data.Project.ValueString(), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitLab project mirror", err.Error())
		return
	}

	// Keep the URL from the configuration instead of using the one from the API
	configURL := data.URL
	data.modelToStateModel(mirror)
	data.URL = configURL

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabProjectMirrorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectMirrorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, mirrorId, err := data.ResourceGitlabProjectMirrorParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab project mirror", err.Error())
		return
	}

	tflog.Debug(ctx, "reading gitlab project mirror with details", map[string]any{
		"project": project,
		"id":      mirrorId,
	})

	mirror, _, err := r.client.ProjectMirrors.GetProjectMirror(project, mirrorId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "gitlab project mirror not found, removing from state", map[string]any{
				"project": project,
				"id":      mirrorId,
			})
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error reading GitLab project mirror", err.Error())
		}
		return
	}

	data.Project = types.StringValue(project)
	data.modelToStateModel(mirror)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *gitlabProjectMirrorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectMirrorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Also get the state to preserve the ID
	var state *gitlabProjectMirrorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the ID from the state
	data.ID = state.ID
	data.MirrorID = state.MirrorID

	project, mirrorId, err := data.ResourceGitlabProjectMirrorParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab project mirror", err.Error())
		return
	}

	options := &gitlab.EditProjectMirrorOptions{
		Enabled:           data.Enabled.ValueBoolPointer(),
		KeepDivergentRefs: data.KeepDivergentRefs.ValueBoolPointer(),
	}

	if !data.OnlyProtectedBranches.IsNull() && !data.OnlyProtectedBranches.IsUnknown() {
		options.OnlyProtectedBranches = data.OnlyProtectedBranches.ValueBoolPointer()
	}

	if !data.MirrorBranchRegex.IsNull() && !data.MirrorBranchRegex.IsUnknown() && data.MirrorBranchRegex.ValueString() != "" {
		options.MirrorBranchRegex = data.MirrorBranchRegex.ValueStringPointer()
	}

	if !data.AuthMethod.IsNull() && !data.AuthMethod.IsUnknown() {
		options.AuthMethod = data.AuthMethod.ValueStringPointer()
	}

	tflog.Debug(ctx, "updating gitlab project mirror", map[string]any{
		"project":  project,
		"mirrorId": mirrorId,
	})

	mirror, _, err := r.client.ProjectMirrors.EditProjectMirror(project, mirrorId, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error updating GitLab project mirror", err.Error())
		return
	}

	// Preserve the ID and Project when updating the state model
	projectID := data.Project
	id := data.ID
	data.modelToStateModel(mirror)
	data.ID = id
	data.Project = projectID

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *gitlabProjectMirrorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectMirrorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, mirrorId, err := data.ResourceGitlabProjectMirrorParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab project mirror", err.Error())
		return
	}

	_, err = r.client.ProjectMirrors.DeleteProjectMirror(project, mirrorId, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting GitLab project mirror", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

// getSchema returns the schema for this resource.
func (r *gitlabProjectMirrorResource) getSchema() schema.Schema {
	allowedAuthMethods := []string{"ssh_public_key", "password"}

	return schema.Schema{
		Version: 0,
		MarkdownDescription: fmt.Sprintf(`The `+"`gitlab"+r.ResourceName+"`"+` resource manages the lifecycle of a project mirror.

This is for *pushing* changes to a remote repository. *Pull Mirroring* can be configured using a combination of the
import_url, mirror, and mirror_trigger_builds properties on the gitlab_project resource.

-> **Warning** By default, the provider sets the `+"`"+`keep_divergent_refs`+"`"+` argument to `+"`"+`True`+"`"+`.
   If you manually set `+"`"+`keep_divergent_refs`+"`"+` to `+"`"+`False`+"`"+`, GitLab mirroring removes branches in the target that aren't in the source.
   This action can result in unexpected branch deletions.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/remote_mirrors/)

%s`, r.DeprecationMessage),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The id of the project mirror. In the format of " + "`" + "project:mirror_id" + "`" + "",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The id of the project.",
				Required:            true,
			},
			"mirror_id": schema.Int64Attribute{
				MarkdownDescription: "Mirror ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "The URL of the remote repository to be mirrored.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Determines if the mirror is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"only_protected_branches": schema.BoolAttribute{
				MarkdownDescription: "Determines if only protected branches are mirrored.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRoot("mirror_branch_regex")),
				},
			},
			"mirror_branch_regex": schema.StringAttribute{
				MarkdownDescription: "Contains a regular expression. Only branches with names matching the regex are mirrored. Requires only_protected_branches to be disabled. Premium and Ultimate only.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("only_protected_branches")),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"keep_divergent_refs": schema.BoolAttribute{
				MarkdownDescription: "Determines if divergent refs are skipped.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"auth_method": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Determines the mirror authentication method. Valid values are: %s.", utils.RenderValueListForDocs(allowedAuthMethods)),
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(allowedAuthMethods...)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ImportState handles importing an existing GitLab project mirror.
func (d *gitlabProjectMirrorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// modelToStateModel maps the API response to the Terraform state.
func (d *gitlabProjectMirrorResourceModel) modelToStateModel(a *gitlab.ProjectMirror) {
	d.ID = types.StringValue(utils.BuildTwoPartID(d.Project.ValueStringPointer(), gitlab.Ptr(strconv.FormatInt(a.ID, 10))))
	d.MirrorID = types.Int64Value(int64(a.ID))

	d.URL = types.StringValue(a.URL)

	d.Enabled = types.BoolValue(a.Enabled)
	d.OnlyProtectedBranches = types.BoolValue(a.OnlyProtectedBranches)
	d.MirrorBranchRegex = types.StringValue(a.MirrorBranchRegex)
	d.KeepDivergentRefs = types.BoolValue(a.KeepDivergentRefs)
	d.AuthMethod = types.StringValue(a.AuthMethod)
}

// ResourceGitlabProjectMirrorParseId parses the resource ID into project and mirror ID components.
func (d *gitlabProjectMirrorResourceModel) ResourceGitlabProjectMirrorParseId(id string) (string, int64, error) {
	if id == "" {
		return "", 0, fmt.Errorf("Invalid ID format (\"\"). Expected <project>:<mirror_id>")
	}

	project, rawMirrorId, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, fmt.Errorf("Invalid ID format (%s). Expected <project>:<mirror_id>", id)
	}

	mirrorId, err := strconv.ParseInt(rawMirrorId, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("Invalid mirror ID (%s): %s", rawMirrorId, err)
	}

	return project, mirrorId, nil
}
