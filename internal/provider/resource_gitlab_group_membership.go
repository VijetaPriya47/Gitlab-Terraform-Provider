package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &gitlabGroupMembershipResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupMembershipResource{}
	_ resource.ResourceWithImportState = &gitlabGroupMembershipResource{}
)

func init() {
	registerResource(NewGitlabGroupMembershipResource)
}

func NewGitlabGroupMembershipResource() resource.Resource {
	return &gitlabGroupMembershipResource{}
}

type gitlabGroupMembershipResource struct {
	client *gitlab.Client
}

type gitlabGroupMembershipResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	GroupID                    types.Int64  `tfsdk:"group_id"`
	UserID                     types.Int64  `tfsdk:"user_id"`
	AccessLevel                types.String `tfsdk:"access_level"`
	MemberRoleID               types.Int64  `tfsdk:"member_role_id"`
	ExpiresAt                  types.String `tfsdk:"expires_at"`
	SkipSubresourcesOnDestroy  types.Bool   `tfsdk:"skip_subresources_on_destroy"`
	UnassignIssuablesOnDestroy types.Bool   `tfsdk:"unassign_issuables_on_destroy"`
}

func (r *gitlabGroupMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *gitlabGroupMembershipResource) Schema(ctc context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_membership`" + ` resource allows to manage the lifecycle of a users group membership.

-> If a group should grant membership to another group use the ` + "`gitlab_group_share_group`" + ` resource instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/members/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the group membership. In the format of `<group-id:user-id>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the group.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Access level for the member. Valid values are: %s.", utils.RenderValueListForDocs(api.ValidGroupAccessLevelNames)),
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidGroupAccessLevelNames...)},
			},
			"member_role_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of a custom member role. Only available for Ultimate instances.",
				Optional:            true,
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "Expiration date for the group membership. Format: `YYYY-MM-DD`",
				Optional:            true,
			},
			"skip_subresources_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Whether the deletion of direct memberships of the removed member in subgroups and projects should be skipped. Only used during a destroy.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"unassign_issuables_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Whether the removed member should be unassigned from any issues or merge requests inside a given group or project. Only used during a destroy.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *gitlabGroupMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (d *gitlabGroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabGroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupMembershipResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userId := int(data.UserID.ValueInt64())
	groupId := int(data.GroupID.ValueInt64())
	expiresAt := data.ExpiresAt.ValueString()
	accessLevelId := api.AccessLevelNameToValue[strings.ToLower(data.AccessLevel.ValueString())]

	options := &gitlab.AddGroupMemberOptions{
		UserID:      &userId,
		AccessLevel: &accessLevelId,
		ExpiresAt:   &expiresAt,
	}

	if !data.MemberRoleID.IsNull() {
		options.MemberRoleID = gitlab.Ptr(int(data.MemberRoleID.ValueInt64()))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab group groupMember for %d in %d", options.UserID, groupId))

	groupMember, _, err := d.client.GroupMembers.AddGroupMember(groupId, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error creating new GitLab group membership", fmt.Sprintf("couldn't create new GitLab group membership: %v", err)))
		return
	}
	resp.Diagnostics.Append(data.groupMembershipToStateModel(groupId, groupMember)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabGroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabGroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab group groupMember %s", id))
	groupIdString, userIdString, err := utils.ParseTwoPartID(id)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error getting group and user ID from resource ID", fmt.Sprintf("Error getting group and user ID from resource ID: %v", err)))
		return
	}
	groupId, err := strconv.Atoi(groupIdString)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error converting group ID to int", fmt.Sprintf("Error converting group ID to int: %v", err)))
		return
	}
	userId, err := strconv.Atoi(userIdString)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error converting user ID to int", fmt.Sprintf("Error converting user ID to int: %v", err)))
		return
	}

	groupMember, _, err := d.client.GroupMembers.GetGroupMember(groupId, userId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab group membership for %s not found so removing from state", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error reading GitLab group membership", fmt.Sprintf("Error reading GitLab group membership: %v", err)))
		return
	}
	resp.Diagnostics.Append(data.groupMembershipToStateModel(groupId, groupMember)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabGroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabGroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userId := int(data.UserID.ValueInt64())
	groupId := int(data.GroupID.ValueInt64())
	expiresAt := data.ExpiresAt.ValueString()
	accessLevelId := api.AccessLevelNameToValue[strings.ToLower(data.AccessLevel.ValueString())]

	options := gitlab.EditGroupMemberOptions{
		AccessLevel: &accessLevelId,
		ExpiresAt:   &expiresAt,
	}

	if !data.MemberRoleID.IsNull() {
		options.MemberRoleID = gitlab.Ptr(int(data.MemberRoleID.ValueInt64()))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab group membership %v for %v", userId, groupId))

	groupMember, _, err := d.client.GroupMembers.EditGroupMember(groupId, userId, &options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error updating GitLab group membership", fmt.Sprintf("Error updating GitLab group membership: %v", err)))
		return
	}
	resp.Diagnostics.Append(data.groupMembershipToStateModel(groupId, groupMember)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabGroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabGroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	groupIdString, userIdString, err := utils.ParseTwoPartID(id)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error getting group and user ID from resource ID", fmt.Sprintf("Error getting group and user ID from resource ID: %v", err)))
		return
	}
	groupId, err := strconv.Atoi(groupIdString)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error converting group ID to int", fmt.Sprintf("Error converting group ID to int: %v", err)))
		return
	}
	userId, err := strconv.Atoi(userIdString)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error converting user ID to int", fmt.Sprintf("Error converting user ID to int: %v", err)))
		return
	}

	options := gitlab.RemoveGroupMemberOptions{
		SkipSubresources:  gitlab.Ptr(data.SkipSubresourcesOnDestroy.ValueBool()),
		UnassignIssuables: gitlab.Ptr(data.UnassignIssuablesOnDestroy.ValueBool()),
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab group membership %v for %v with options: %+v", userId, groupId, options))

	_, err = d.client.GroupMembers.RemoveGroupMember(groupId, userId, &options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error deleting GitLab group membership", fmt.Sprintf("Error deleting GitLab group membership: %v", err)))
		return
	}
}

func (data *gitlabGroupMembershipResourceModel) groupMembershipToStateModel(groupId int, groupMember *gitlab.GroupMember) diag.Diagnostics {
	groupIDString := strconv.Itoa(groupId)
	userIDString := strconv.Itoa(groupMember.ID)
	data.ID = types.StringValue(utils.BuildTwoPartID(&groupIDString, &userIDString))
	data.GroupID = types.Int64Value(int64(groupId))
	data.UserID = types.Int64Value(int64(groupMember.ID))
	data.AccessLevel = types.StringValue(api.AccessLevelValueToName[groupMember.AccessLevel])

	if groupMember.MemberRole != nil {
		data.MemberRoleID = types.Int64Value(int64(groupMember.MemberRole.ID))
	}

	if groupMember.ExpiresAt != nil {
		data.ExpiresAt = types.StringValue(groupMember.ExpiresAt.String())
	}

	if data.SkipSubresourcesOnDestroy.IsNull() {
		data.SkipSubresourcesOnDestroy = types.BoolValue(false)
	}

	if data.UnassignIssuablesOnDestroy.IsNull() {
		data.UnassignIssuablesOnDestroy = types.BoolValue(false)
	}

	return nil
}
