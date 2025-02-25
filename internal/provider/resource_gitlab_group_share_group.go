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
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabGroupShareGroupResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupShareGroupResource{}
	_ resource.ResourceWithImportState = &gitlabGroupShareGroupResource{}
)

func init() {
	registerResource(NewGitLabGroupShareGroupResource)
}

func NewGitLabGroupShareGroupResource() resource.Resource {
	return &gitlabGroupShareGroupResource{}
}

type gitlabGroupShareGroupResource struct {
	client *gitlab.Client
}

type gitlabGroupShareGroupResourceModel struct {
	ID           types.String `tfsdk:"id"`
	GroupID      types.String `tfsdk:"group_id"`
	ShareGroupID types.Int64  `tfsdk:"share_group_id"`
	GroupAccess  types.String `tfsdk:"group_access"`
	ExpiresAt    types.String `tfsdk:"expires_at"`
}

func (r *gitlabGroupShareGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_share_group"
}

func (r *gitlabGroupShareGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_group_share_group` + "`" + ` resource allows managing the lifecycle of a group shared with another group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/groups/#share-groups-with-groups)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource. In the format of <group-id:share-group-id>.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The id of the main group to be shared.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"share_group_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the additional group with which the main group will be shared.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"group_access": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The access level to grant the group. Valid values are: %s", utils.RenderValueListForDocs(api.ValidGroupAccessLevelNames)),
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidGroupAccessLevelNames...)},
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "Share expiration date. Format: `YYYY-MM-DD`",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *gitlabGroupShareGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabGroupShareGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabGroupShareGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupShareGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := data.GroupID.ValueString()
	shareGroupID := int(data.ShareGroupID.ValueInt64())
	groupAccess := api.AccessLevelNameToValue[data.GroupAccess.ValueString()]

	options := &gitlab.ShareWithGroupOptions{
		GroupID:     &shareGroupID,
		GroupAccess: &groupAccess,
	}

	if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() {
		options.ExpiresAt = gitlab.Ptr(data.ExpiresAt.ValueString())
	}

	_, _, err := r.client.GroupMembers.ShareWithGroup(groupID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create group share: %s", err.Error()))
		return
	}

	shareGroupIDString := strconv.Itoa(shareGroupID)
	data.ID = types.StringValue(utils.BuildTwoPartID(&groupID, &shareGroupIDString))
	data.GroupID = types.StringValue(groupID)
	data.ShareGroupID = types.Int64Value(int64(shareGroupID))
	data.GroupAccess = types.StringValue(api.AccessLevelValueToName[groupAccess])
	if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() {
		data.ExpiresAt = types.String(data.ExpiresAt)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupShareGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupShareGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	groupID, sharedGroupID, err := groupIdsFromId(id)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing resource ID", fmt.Sprintf("Unable to parse resource ID %s: %s", id, err.Error()))
		return
	}

	group, _, err := r.client.Groups.GetGroup(groupID, nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("Group not found", fmt.Sprintf("[DEBUG] gitlab group %s not found so removing from state", groupID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read group %s: %s", groupID, err.Error()))
		return
	}

	for _, sharedGroup := range group.SharedWithGroups {
		if sharedGroupID == sharedGroup.GroupID {
			convertedAccessLevel := gitlab.AccessLevelValue(sharedGroup.GroupAccessLevel)
			data.GroupID = types.StringValue(groupID)
			data.ShareGroupID = types.Int64Value(int64(sharedGroupID))
			data.GroupAccess = types.StringValue(api.AccessLevelValueToName[convertedAccessLevel])

			if sharedGroup.ExpiresAt == nil {
				data.ExpiresAt = types.StringNull()
			} else {
				data.ExpiresAt = types.StringValue(sharedGroup.ExpiresAt.String())
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddWarning("Group share not found", fmt.Sprintf("Unable to find group share %s so removing from state", id))
	resp.State.RemoveResource(ctx)
}

func (r *gitlabGroupShareGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place update which is not possible.")
}

func (r *gitlabGroupShareGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupShareGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	groupID, sharedGroupID, err := groupIdsFromId(id)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing resource ID", fmt.Sprintf("Unable to parse resource ID %s: %s", id, err.Error()))
		return
	}

	_, err = r.client.GroupMembers.DeleteShareWithGroup(groupID, sharedGroupID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete group share: %s", err.Error()))
		return
	}

	resp.State.RemoveResource(ctx)
}

func groupIdsFromId(id string) (string, int, error) {
	groupId, sharedGroupIdString, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, fmt.Errorf("error parsing ID: %s", id)
	}

	sharedGroupId, err := strconv.Atoi(sharedGroupIdString)
	if err != nil {
		return "", 0, fmt.Errorf("can not determine shared group id: %s", sharedGroupIdString)
	}

	return groupId, sharedGroupId, nil
}
