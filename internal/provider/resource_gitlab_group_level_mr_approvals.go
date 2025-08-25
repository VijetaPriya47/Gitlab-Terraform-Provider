package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ resource.Resource                = &gitlabGroupLevelMRApprovalsResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupLevelMRApprovalsResource{}
	_ resource.ResourceWithImportState = &gitlabGroupLevelMRApprovalsResource{}
)

func init() {
	registerResource(NewGitlabGroupLevelMRApprovalsResource)
}

func NewGitlabGroupLevelMRApprovalsResource() resource.Resource {
	return &gitlabGroupLevelMRApprovalsResource{}
}

type gitlabGroupLevelMRApprovalsResourceModel struct {
	ID                                          types.String `tfsdk:"id"`
	Group                                       types.String `tfsdk:"group"`
	KeepSettingsOnDestroy                       types.Bool   `tfsdk:"keep_settings_on_destroy"`
	AllowAuthorApproval                         types.Bool   `tfsdk:"allow_author_approval"`
	AllowCommitterApproval                      types.Bool   `tfsdk:"allow_committer_approval"`
	AllowOverridesToApproverListPerMergeRequest types.Bool   `tfsdk:"allow_overrides_to_approver_list_per_merge_request"`
	RetainApprovalsOnPush                       types.Bool   `tfsdk:"retain_approvals_on_push"`
	RequireReauthenticationToApprove            types.Bool   `tfsdk:"require_reauthentication_to_approve"`
}

type gitlabGroupLevelMRApprovalsPrivateStateModel struct {
	AllowAuthorApproval                         bool `json:"allow_author_approval"`
	AllowCommitterApproval                      bool `json:"allow_committer_approval"`
	AllowOverridesToApproverListPerMergeRequest bool `json:"allow_overrides_to_approver_list_per_merge_request"`
	RetainApprovalsOnPush                       bool `json:"retain_approvals_on_push"`
	RequireReauthenticationToApprove            bool `json:"require_reauthentication_to_approve"`
}

type gitlabGroupLevelMRApprovalsResource struct {
	client *gitlab.Client
}

func (r *gitlabGroupLevelMRApprovalsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_level_mr_approvals"
}

func (r *gitlabGroupLevelMRApprovalsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_level_mr_approvals`" + ` resource manages the lifecycle of group merge request approval settings. More than one resource per group will conflict with each other.

~> This is an **experimental resource**. By nature it doesn't properly fit into how Terraform resources are meant to work.

~> If ` + "`" + `keep_settings_on_destroy` + "`" + ` is set to false, destroying the resource will revert settings to the values that were present when the resource was first created.
You will need to apply the resource with the new setting before destroying the resource.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/merge_request_approval_settings/#group-mr-approval-settings)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the group.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"keep_settings_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Set to true if the group merge request approval settings should not be reset to their pre-terraform defaults on destroy. You will need to apply the resource with the new setting before destroying the resource.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"allow_author_approval": schema.BoolAttribute{
				MarkdownDescription: "Allow or prevent authors from self approving merge requests; `true` means authors can self approve.",
				Optional:            true,
				Computed:            true,
			},
			"allow_committer_approval": schema.BoolAttribute{
				MarkdownDescription: "Allow or prevent committers from self approving merge requests.",
				Optional:            true,
				Computed:            true,
			},
			"allow_overrides_to_approver_list_per_merge_request": schema.BoolAttribute{
				MarkdownDescription: "Allow or prevent overriding approvers per merge request.",
				Optional:            true,
				Computed:            true,
			},
			"retain_approvals_on_push": schema.BoolAttribute{
				MarkdownDescription: "Retain approval count on a new push.",
				Optional:            true,
				Computed:            true,
			},
			"require_reauthentication_to_approve": schema.BoolAttribute{
				MarkdownDescription: "Require approver to authenticate before adding the approval.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *gitlabGroupLevelMRApprovalsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabGroupLevelMRApprovalsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabGroupLevelMRApprovalsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupLevelMRApprovalsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := data.Group.ValueString()
	r.storeOriginalSettings(ctx, group, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.changeSettings(ctx, data, group)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update group merge request approval settings: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(group)
	data.modelToStateModel(settings, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupLevelMRApprovalsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupLevelMRApprovalsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := data.ID.ValueString()
	settings, _, err := r.client.MergeRequestApprovalSettings.GetGroupMergeRequestApprovalSettings(group, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("GitLab API error occurred", fmt.Sprintf("Group doesn't exist anymore, removing from state: %s", err.Error()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get group merge request approval settings: %s", err.Error()))
		return
	}
	data.modelToStateModel(settings, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupLevelMRApprovalsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabGroupLevelMRApprovalsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := data.ID.ValueString()
	settings, err := r.changeSettings(ctx, data, group)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update group merge request approval settings: %s", err.Error()))
		return
	}
	data.modelToStateModel(settings, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupLevelMRApprovalsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupLevelMRApprovalsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.KeepSettingsOnDestroy.ValueBool() {
		group := data.ID.ValueString()
		original, diags := req.Private.GetKey(ctx, fmt.Sprintf("gitlab_group_level_mr_approvals_original_%s", group))
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if original == nil {
			tflog.Debug(ctx, "Could not reset the group merge request approval settings", map[string]any{
				"group_id": group,
			})
			resp.Diagnostics.AddWarning("Could not reset the group merge request approval settings", "No original approval settings found to reset to")
			resp.State.RemoveResource(ctx)
			return
		}

		var settings *gitlabGroupLevelMRApprovalsPrivateStateModel
		err := json.Unmarshal(original, &settings)
		if err != nil {
			tflog.Debug(ctx, "Could not unmarshal the original settings", map[string]any{
				"group_id": group,
				"original": original,
			})
			resp.Diagnostics.AddError("Could not unmarshal the original settings", err.Error())
			return
		}

		options := &gitlab.UpdateMergeRequestApprovalSettingsOptions{
			AllowAuthorApproval:                         gitlab.Ptr(settings.AllowAuthorApproval),
			AllowCommitterApproval:                      gitlab.Ptr(settings.AllowCommitterApproval),
			AllowOverridesToApproverListPerMergeRequest: gitlab.Ptr(settings.AllowOverridesToApproverListPerMergeRequest),
			RetainApprovalsOnPush:                       gitlab.Ptr(settings.RetainApprovalsOnPush),
			RequireReauthenticationToApprove:            gitlab.Ptr(settings.RequireReauthenticationToApprove),
		}
		_, _, err = r.client.MergeRequestApprovalSettings.UpdateGroupMergeRequestApprovalSettings(group, options, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to reset group merge request approval settings: %s", err.Error()))
			return
		}
	}

	tflog.Debug(ctx, "destroying the group merge request approvals settings resource does not do anything.")
	resp.State.RemoveResource(ctx)
}

func (r *gitlabGroupLevelMRApprovalsResource) storeOriginalSettings(ctx context.Context, group string, resp *resource.CreateResponse) {
	settings, _, err := r.client.MergeRequestApprovalSettings.GetGroupMergeRequestApprovalSettings(group, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get current group merge request approval settings: %s", err.Error()))
		return
	}

	data := gitlabGroupLevelMRApprovalsPrivateStateModel{
		AllowAuthorApproval:                         settings.AllowAuthorApproval.Value,
		AllowCommitterApproval:                      settings.AllowCommitterApproval.Value,
		AllowOverridesToApproverListPerMergeRequest: settings.AllowOverridesToApproverListPerMergeRequest.Value,
		RetainApprovalsOnPush:                       settings.RetainApprovalsOnPush.Value,
		RequireReauthenticationToApprove:            settings.RequireReauthenticationToApprove.Value,
	}

	b, err := json.Marshal(data)
	if err != nil {
		resp.Diagnostics.AddError("Error marshalling settings into json", fmt.Sprintf("Unable to marshall resource model into json: %s", err.Error()))
		return
	}

	diags := resp.Private.SetKey(ctx, fmt.Sprintf("gitlab_group_level_mr_approvals_original_%s", group), b)
	resp.Diagnostics.Append(diags...)
}

func (r *gitlabGroupLevelMRApprovalsResource) changeSettings(ctx context.Context, data *gitlabGroupLevelMRApprovalsResourceModel, group string) (*gitlab.MergeRequestApprovalSettings, error) {
	options := &gitlab.UpdateMergeRequestApprovalSettingsOptions{}

	if !data.AllowAuthorApproval.IsNull() && !data.AllowAuthorApproval.IsUnknown() {
		options.AllowAuthorApproval = data.AllowAuthorApproval.ValueBoolPointer()
	}
	if !data.AllowCommitterApproval.IsNull() && !data.AllowCommitterApproval.IsUnknown() {
		options.AllowCommitterApproval = data.AllowCommitterApproval.ValueBoolPointer()
	}
	if !data.AllowOverridesToApproverListPerMergeRequest.IsNull() && !data.AllowOverridesToApproverListPerMergeRequest.IsUnknown() {
		options.AllowOverridesToApproverListPerMergeRequest = data.AllowOverridesToApproverListPerMergeRequest.ValueBoolPointer()
	}
	if !data.RetainApprovalsOnPush.IsNull() && !data.RetainApprovalsOnPush.IsUnknown() {
		options.RetainApprovalsOnPush = data.RetainApprovalsOnPush.ValueBoolPointer()
	}
	if !data.RequireReauthenticationToApprove.IsNull() && !data.RequireReauthenticationToApprove.IsUnknown() {
		options.RequireReauthenticationToApprove = data.RequireReauthenticationToApprove.ValueBoolPointer()
	}

	settings, _, err := r.client.MergeRequestApprovalSettings.UpdateGroupMergeRequestApprovalSettings(group, options, gitlab.WithContext(ctx))
	return settings, err
}

func (d *gitlabGroupLevelMRApprovalsResourceModel) modelToStateModel(settings *gitlab.MergeRequestApprovalSettings, group string) {
	d.Group = types.StringValue(group)
	d.AllowAuthorApproval = types.BoolValue(settings.AllowAuthorApproval.Value)
	d.AllowCommitterApproval = types.BoolValue(settings.AllowCommitterApproval.Value)
	d.AllowOverridesToApproverListPerMergeRequest = types.BoolValue(settings.AllowOverridesToApproverListPerMergeRequest.Value)
	d.RetainApprovalsOnPush = types.BoolValue(settings.RetainApprovalsOnPush.Value)
	d.RequireReauthenticationToApprove = types.BoolValue(settings.RequireReauthenticationToApprove.Value)
}
