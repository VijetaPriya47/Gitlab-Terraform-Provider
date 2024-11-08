package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ resource.Resource                   = &gitlabProjectLevelMrApprovalsResource{}
	_ resource.ResourceWithConfigure      = &gitlabProjectLevelMrApprovalsResource{}
	_ resource.ResourceWithImportState    = &gitlabProjectLevelMrApprovalsResource{}
	_ resource.ResourceWithUpgradeState   = &gitlabProjectLevelMrApprovalsResource{}
	_ resource.ResourceWithValidateConfig = &gitlabProjectLevelMrApprovalsResource{}
)

func init() {
	registerResource(NewGitlabProjectLevelMrApprovalsResource)
}

func NewGitlabProjectLevelMrApprovalsResource() resource.Resource {
	return &gitlabProjectLevelMrApprovalsResource{}
}

type gitlabProjectLevelMrApprovalsResource struct {
	client *gitlab.Client
}

type gitlabProjectLevelMrApprovalsModel struct {
	ID                                        types.String `tfsdk:"id"`
	Project                                   types.String `tfsdk:"project"`
	ResetApprovalsOnPush                      types.Bool   `tfsdk:"reset_approvals_on_push"`
	DisableOverridingApproversPerMergeRequest types.Bool   `tfsdk:"disable_overriding_approvers_per_merge_request"`
	MergeRequestsAuthorApproval               types.Bool   `tfsdk:"merge_requests_author_approval"`
	MergeRequestsDisableCommittersApproval    types.Bool   `tfsdk:"merge_requests_disable_committers_approval"`
	RequirePasswordToApprove                  types.Bool   `tfsdk:"require_password_to_approve"`
	SelectiveCodeOwnerRemovals                types.Bool   `tfsdk:"selective_code_owner_removals"`
}

// Metadata returns the resource name
func (d *gitlabProjectLevelMrApprovalsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_level_mr_approvals"
}

func (d *gitlabProjectLevelMrApprovalsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = d.getV1Schema()
}

func (r *gitlabProjectLevelMrApprovalsResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// provides plan-time validation of the configuration across multiple attributes (as opposed to just attribute-level validation)
func (d *gitlabProjectLevelMrApprovalsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data gitlabProjectLevelMrApprovalsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that if selective_code_owner_removals is set to true, reset_approvals_on_push is set to false
	if !data.SelectiveCodeOwnerRemovals.IsNull() && !data.ResetApprovalsOnPush.IsNull() && data.SelectiveCodeOwnerRemovals.ValueBool() && data.ResetApprovalsOnPush.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("selective_code_owner_removals"),
			"selective_code_owner_removals can only be enabled when reset_approvals_on_push is disabled",
			"Selective code owner removals may only be configured to `true` while Reset approvals on push is configured to `false` or unconfigured",
		)
	}
}

func (d *gitlabProjectLevelMrApprovalsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabProjectLevelMrApprovalsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()

	// Retrieve all options (except selective code owner removals, since that's a separate call)
	options := d.getApprovalConfigurationOptions(data)

	tflog.Debug(ctx, fmt.Sprintf("Creating new MR approval configuration for project: %s", data.Project.ValueString()), nil)
	approvals, _, err := d.client.Projects.ChangeApprovalConfiguration(project, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error applying approval configuration", fmt.Sprintf("couldn't create approval configuration: %v", err)))
		return
	}

	// If Selective Code Owner Removals is set to `true`, it needs to be applied in a second step because the API errors if both
	// it and Reset Approvals On Push are passed in, even if they're set with proper values.
	if !data.SelectiveCodeOwnerRemovals.IsNull() && data.SelectiveCodeOwnerRemovals.ValueBool() {
		app, diag := d.applySelectiveCodeOwnerRemovals(data, ctx)
		if diag != nil {
			resp.Diagnostics.Append(diag)
			return
		}
		approvals = app
	}

	data.ID = types.StringValue(project)
	data.modelToStateModel(approvals, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabProjectLevelMrApprovalsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabProjectLevelMrApprovalsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Reading gitlab approval configuration for project %s", project))

	approvalConfig, _, err := d.client.Projects.GetApprovalConfiguration(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab project approval configuration not found for project %s", project))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"couldn't read approval configuration",
			fmt.Sprintf("couldn't read approval configuration due to error: %v", err),
		))
		return
	}

	data.modelToStateModel(approvalConfig, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabProjectLevelMrApprovalsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabProjectLevelMrApprovalsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()

	// Retrieve all options (except selective code owner removals, since that's a separate call)
	options := d.getApprovalConfigurationOptions(data)

	// Unfortunately, selective code owner removals needs to be applied twice here, once before and once after. This is because
	// if it's being set to "false", it needs to be set before `reset_approvals_on_push` can be set to true, and if it's being set to
	// "true", it needs to be set after `reset_approvals_on_push` can be set to false.
	var approvalConfig *gitlab.ProjectApprovals
	if !data.SelectiveCodeOwnerRemovals.IsNull() && !data.SelectiveCodeOwnerRemovals.ValueBool() {
		_, diag := d.applySelectiveCodeOwnerRemovals(data, ctx)
		if diag != nil {
			resp.Diagnostics.Append(diag)
			return
		}
	}

	approvalConfig, _, err := d.client.Projects.ChangeApprovalConfiguration(project, options, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab project approval configuration not found for project %s", project))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"couldn't read approval configuration",
			fmt.Sprintf("couldn't read approval configuration due to error: %v", err),
		))
		return
	}

	if !data.SelectiveCodeOwnerRemovals.IsNull() && data.SelectiveCodeOwnerRemovals.ValueBool() {
		app, diag := d.applySelectiveCodeOwnerRemovals(data, ctx)
		if diag != nil {
			resp.Diagnostics.Append(diag)
			return
		}
		approvalConfig = app
	}

	data.modelToStateModel(approvalConfig, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabProjectLevelMrApprovalsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectLevelMrApprovalsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()

	// Reset Selective Code Owner Removal (since it needs to be set to false before ResetApprovalsOnPush can be set to true)
	data.SelectiveCodeOwnerRemovals = types.BoolValue(false)
	_, errDiag := d.applySelectiveCodeOwnerRemovals(*data, ctx)
	if errDiag != nil {
		resp.Diagnostics.Append(errDiag)
		return
	}

	// Reset to default values (this is what we did on the SDK resource)
	options := &gitlab.ChangeApprovalConfigurationOptions{
		ResetApprovalsOnPush:                      gitlab.Ptr(true),
		DisableOverridingApproversPerMergeRequest: gitlab.Ptr(false),
		MergeRequestsAuthorApproval:               gitlab.Ptr(false),
		MergeRequestsDisableCommittersApproval:    gitlab.Ptr(false),
		RequirePasswordToApprove:                  gitlab.Ptr(false),
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Resetting approval configuration for project %s:", project))

	if _, _, err := d.client.Projects.ChangeApprovalConfiguration(project, options, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"couldn't reset approval configuration",
			fmt.Sprintf("couldn't reset approval configuration due to error: %v", err),
		))
	}
}

func (d *gitlabProjectLevelMrApprovalsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Note: In the framework, every state upgrade function must perform all steps necessary to upgrade the state
// in a single step. That means if we ever add another v2 schema, the v0 upgrader will also need to be updated.
func (d *gitlabProjectLevelMrApprovalsResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	// The v0 schema. Needed so the pointer can be used.
	schema := d.getV0Schema()

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data *gitlabProjectLevelMrApprovalsModelSchema0
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Move all data to the new struct values. Note: Only values that were present at v0 time
				// are included here, because they couldn't exist as known in the old struct.
				newData := &gitlabProjectLevelMrApprovalsModel{
					ID:                   data.ID,
					Project:              data.ProjectID,
					ResetApprovalsOnPush: data.ResetApprovalsOnPush,
					DisableOverridingApproversPerMergeRequest: data.DisableOverridingApproversPerMergeRequest,
					MergeRequestsAuthorApproval:               data.MergeRequestsAuthorApproval,
					MergeRequestsDisableCommittersApproval:    data.MergeRequestsDisableCommittersApproval,
					RequirePasswordToApprove:                  data.RequirePasswordToApprove,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &newData)...)
			},
		},
	}
}

func (d *gitlabProjectLevelMrApprovalsResource) getV1Schema() schema.Schema {
	return schema.Schema{
		Version: 1,
		MarkdownDescription: `The ` + "`" + `gitlab_project_level_mr_approval_rule` + "`" + ` resource allows to manage the lifecycle of a Merge Request-level approval rule.

-> This resource requires a GitLab Enterprise instance.
		
**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/merge_request_approvals.html#merge-request-level-mr-approvals)`,

		// Schema is external because we'll need to re-implement the state migration function.
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the resource. Matches the `project` value.",
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of a project to change MR approval configuration.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"reset_approvals_on_push": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` to remove all approvals in a merge request when new commits are pushed to its source branch. Default is `true`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"disable_overriding_approvers_per_merge_request": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` to disable overriding approvers per merge request.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"merge_requests_author_approval": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` to allow merge requests authors to approve their own merge requests.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"merge_requests_disable_committers_approval": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` to disable merge request committers from approving their own merge requests.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"require_password_to_approve": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` to require authentication to approve merge requests.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"selective_code_owner_removals": schema.BoolAttribute{
				MarkdownDescription: "Reset approvals from Code Owners if their files changed. Can be enabled only if reset_approvals_on_push is disabled.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Struct used for schema 0, which contains the "project_id" attribute instead of the "project" attribute.
// This is necessary for schema migration in the framework.
type gitlabProjectLevelMrApprovalsModelSchema0 struct {
	ID                                        types.String `tfsdk:"id"`
	ProjectID                                 types.String `tfsdk:"project_id"`
	ResetApprovalsOnPush                      types.Bool   `tfsdk:"reset_approvals_on_push"`
	DisableOverridingApproversPerMergeRequest types.Bool   `tfsdk:"disable_overriding_approvers_per_merge_request"`
	MergeRequestsAuthorApproval               types.Bool   `tfsdk:"merge_requests_author_approval"`
	MergeRequestsDisableCommittersApproval    types.Bool   `tfsdk:"merge_requests_disable_committers_approval"`
	RequirePasswordToApprove                  types.Bool   `tfsdk:"require_password_to_approve"`
}

func (d *gitlabProjectLevelMrApprovalsResource) getV0Schema() schema.Schema {
	return schema.Schema{
		Version: 0,
		MarkdownDescription: `The ` + "`" + `gitlab_project_level_mr_approval_rule` + "`" + ` resource allows to manage the lifecycle of a Merge Request-level approval rule.

-> This resource requires a GitLab Enterprise instance.
				
**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/merge_request_approvals.html#merge-request-level-mr-approvals)`,

		// Schema is external because we'll need to re-implement the state migration function.
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the resource. Matches the `project` value.",
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of a project to change MR approval configuration.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"reset_approvals_on_push": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` if you want to remove all approvals in a merge request when new commits are pushed to its source branch. Default is `true`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"disable_overriding_approvers_per_merge_request": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` if you want to disable overriding approvers per merge request.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"merge_requests_author_approval": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` if you want to allow merge request authors to self-approve merge requests.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"merge_requests_disable_committers_approval": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` if you want to allow merge request committers to self-approve merge requests.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"require_password_to_approve": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` if you want to require authentication to approve merge requests.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (d *gitlabProjectLevelMrApprovalsModel) modelToStateModel(a *gitlab.ProjectApprovals, project string) {
	d.ID = types.StringValue(project)
	d.Project = types.StringValue(project)
	d.DisableOverridingApproversPerMergeRequest = types.BoolValue(a.DisableOverridingApproversPerMergeRequest)
	d.MergeRequestsAuthorApproval = types.BoolValue(a.MergeRequestsAuthorApproval)
	d.MergeRequestsDisableCommittersApproval = types.BoolValue(a.MergeRequestsDisableCommittersApproval)
	d.ResetApprovalsOnPush = types.BoolValue(a.ResetApprovalsOnPush)
	d.RequirePasswordToApprove = types.BoolValue(a.RequirePasswordToApprove)
	d.SelectiveCodeOwnerRemovals = types.BoolValue(a.SelectiveCodeOwnerRemovals)
}

func (d *gitlabProjectLevelMrApprovalsResource) applySelectiveCodeOwnerRemovals(data gitlabProjectLevelMrApprovalsModel, ctx context.Context) (*gitlab.ProjectApprovals, diag.Diagnostic) {
	option := &gitlab.ChangeApprovalConfigurationOptions{
		SelectiveCodeOwnerRemovals: gitlab.Ptr(data.SelectiveCodeOwnerRemovals.ValueBool()),
	}
	approval, _, err := d.client.Projects.ChangeApprovalConfiguration(data.Project.ValueString(), option, gitlab.WithContext(ctx))
	if err != nil {
		return nil, diag.NewErrorDiagnostic(
			"couldn't update approval selective_code_owner_removals",
			fmt.Sprintf("couldn't update selective_code_owner_removals configuration due to error: %v", err),
		)
	}

	return approval, nil
}

func (d *gitlabProjectLevelMrApprovalsResource) getApprovalConfigurationOptions(data gitlabProjectLevelMrApprovalsModel) *gitlab.ChangeApprovalConfigurationOptions {
	// There are no required options here
	options := &gitlab.ChangeApprovalConfigurationOptions{}

	if !data.ResetApprovalsOnPush.IsNull() && !data.ResetApprovalsOnPush.ValueBool() {
		options.ResetApprovalsOnPush = data.ResetApprovalsOnPush.ValueBoolPointer()
	}
	if !data.DisableOverridingApproversPerMergeRequest.IsNull() && !data.DisableOverridingApproversPerMergeRequest.IsUnknown() {
		options.DisableOverridingApproversPerMergeRequest = data.DisableOverridingApproversPerMergeRequest.ValueBoolPointer()
	}
	if !data.ResetApprovalsOnPush.IsNull() && !data.ResetApprovalsOnPush.IsUnknown() {
		options.ResetApprovalsOnPush = data.ResetApprovalsOnPush.ValueBoolPointer()
	}
	if !data.DisableOverridingApproversPerMergeRequest.IsNull() && !data.DisableOverridingApproversPerMergeRequest.IsUnknown() {
		options.DisableOverridingApproversPerMergeRequest = data.DisableOverridingApproversPerMergeRequest.ValueBoolPointer()
	}
	if !data.MergeRequestsAuthorApproval.IsNull() && !data.MergeRequestsAuthorApproval.IsUnknown() {
		options.MergeRequestsAuthorApproval = data.MergeRequestsAuthorApproval.ValueBoolPointer()
	}
	if !data.MergeRequestsDisableCommittersApproval.IsNull() && !data.MergeRequestsDisableCommittersApproval.IsUnknown() {
		options.MergeRequestsDisableCommittersApproval = data.MergeRequestsDisableCommittersApproval.ValueBoolPointer()
	}
	if !data.RequirePasswordToApprove.IsNull() && !data.RequirePasswordToApprove.IsUnknown() {
		options.RequirePasswordToApprove = data.RequirePasswordToApprove.ValueBoolPointer()
	}

	return options
}
