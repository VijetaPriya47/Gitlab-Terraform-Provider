package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                 = &gitlabGroupLabelResource{}
	_ resource.ResourceWithConfigure    = &gitlabGroupLabelResource{}
	_ resource.ResourceWithImportState  = &gitlabGroupLabelResource{}
	_ resource.ResourceWithUpgradeState = &gitlabGroupLabelResource{}
)

func init() {
	registerResource(NewGitLabGroupLabelResource)
}

func NewGitLabGroupLabelResource() resource.Resource {
	return &gitlabGroupLabelResource{}
}

type gitlabGroupLabelResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LabelID     types.Int64  `tfsdk:"label_id"`
	Group       types.String `tfsdk:"group"`
	Name        types.String `tfsdk:"name"`
	Color       types.String `tfsdk:"color"`
	ColorHex    types.String `tfsdk:"color_hex"`
	Description types.String `tfsdk:"description"`
}

type gitlabGroupLabelResource struct {
	client *gitlab.Client
}

func (r *gitlabGroupLabelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_label"
}

func (r *gitlabGroupLabelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.getV0Schema()
}

func (r *gitlabGroupLabelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabGroupLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabGroupLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupLabelResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := data.Group.ValueString()
	color := data.Color.ValueString()

	options := &gitlab.CreateGroupLabelOptions{
		Name:  gitlab.Ptr(data.Name.ValueString()),
		Color: gitlab.Ptr(color),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	label, _, err := r.client.GroupLabels.CreateGroupLabel(group, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to create group label: %s", err.Error()))
		return
	}

	labelID := strconv.FormatInt(label.ID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&group, &labelID))
	data.modelToStateModel(label, color, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, labelID, err := data.ResourceGitlabGroupLabelParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	label, _, err := r.client.GroupLabels.GetGroupLabel(group, labelID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("GitLab API error occured", fmt.Sprintf("Group label doesn't exist anymore, removing from state: %s", err.Error()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to get group label: %s", err.Error()))
		return
	}

	data.modelToStateModel(label, data.Color.ValueString(), group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupLabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabGroupLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	group, labelID, err := data.ResourceGitlabGroupLabelParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}
	color := data.Color.ValueString()

	options := &gitlab.UpdateGroupLabelOptions{
		NewName: gitlab.Ptr(data.Name.ValueString()),
		Color:   gitlab.Ptr(color),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	label, _, err := r.client.GroupLabels.UpdateGroupLabel(group, labelID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to update group label: %s", err.Error()))
		return
	}

	data.modelToStateModel(label, color, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, labelID, err := data.ResourceGitlabGroupLabelParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	options := &gitlab.DeleteGroupLabelOptions{
		Name: gitlab.Ptr(data.Name.ValueString()),
	}

	_, err = r.client.GroupLabels.DeleteGroupLabel(group, labelID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to delete group label: %s", err.Error()))
		return
	}

	resp.State.RemoveResource(ctx)
}

// All UpgradeState upgraders in the provider should be registered here, and must upgrade to the current state.
// That means currently all upgraders much upgrade fully to V2.
func (r *gitlabGroupLabelResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	schema := r.getV0Schema()

	// Both v0 and v1 were only ID changes, so they use the same upgrader function. Create that function here.
	upgraderFunction := func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
		var data gitlabGroupLabelResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}

		err := r.upgradeIdToV2Id(ctx, &data)
		if err != nil {
			tflog.Error(ctx, "Failed to upgrade resource ID", map[string]any{
				"oldId": data.ID.ValueString(),
				"group": data.Group.ValueString(),
			})
			resp.Diagnostics.AddError("Failed to upgrade resource ID", fmt.Sprintf("Unable to upgrade resource ID: %s, %s", data.ID.ValueString(), err.Error()))
			return
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema,
			// The only difference between V0 and V1 is the "id" attribute, which sets the ID to "groupId:labelId"
			StateUpgrader: upgraderFunction,
		},
		1: {
			PriorSchema: &schema,
			// The only difference between V0 and V2 is the "id" attribute, which sets the ID to "groupId:labelId"
			StateUpgrader: upgraderFunction,
		},
	}
}

// This function accepts an input state and will upgrde the "id" attribute to the new V2 ID that is <group_id>:<label_id>
func (r *gitlabGroupLabelResource) upgradeIdToV2Id(ctx context.Context, input *gitlabGroupLabelResourceModel) error {
	// Check if LabelId is in state, and retrieve the value from the API if it isn't.
	if input.LabelID.IsNull() || input.LabelID.IsUnknown() {
		tflog.Debug(ctx, "Retrieving label ID from API")
		label, _, err := r.client.GroupLabels.GetGroupLabel(input.Group.ValueString(), input.ID.ValueString(), gitlab.WithContext(ctx))
		if err != nil {
			return err
		}

		// set the value into state
		input.LabelID = types.Int64Value(int64(label.ID))
	}

	// Saving the string to a variable first for readability reasons.
	stringLabelId := strconv.Itoa(int(input.LabelID.ValueInt64()))
	newId := utils.BuildTwoPartID(input.Group.ValueStringPointer(), gitlab.Ptr(stringLabelId))

	tflog.Debug(ctx, "Upgrading state to the V2 ID", map[string]any{
		"oldId": input.ID.ValueString(),
		"group": input.Group.ValueString(),
		"newId": newId,
	})

	input.ID = types.StringValue(newId)
	return nil
}

// The V0 (and also V1) schema. This is used in both the Schema function and also the
// UpgradeState function.
func (r *gitlabGroupLabelResource) getV0Schema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_label`" + ` resource manages the lifecycle of labels within a group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_labels/)`,
		Version: 2,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group-id>:<label-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"label_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the group label.",
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The name or id of the group to add the label to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the label.",
				Required:            true,
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "The color of the label given in 6-digit hex notation with leading '#' sign (e.g. #FFAABB) or one of the [CSS color names](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Values/color_value#Color_keywords).",
				Required:            true,
			},
			"color_hex": schema.StringAttribute{
				MarkdownDescription: "Read-only, used by the provider to store the API response color. This is always in the 6-digit hex notation with leading '#' sign (e.g. #FFAABB). If `color` contains a color name, this attribute contains the hex notation equivalent. Otherwise, the value of this attribute is the same as `color`.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the label.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *gitlabGroupLabelResourceModel) modelToStateModel(l *gitlab.GroupLabel, color string, group string) {
	r.LabelID = types.Int64Value(int64(l.ID))
	r.Group = types.StringValue(group)
	r.Name = types.StringValue(l.Name)
	if color == "" {
		r.Color = types.StringValue(l.Color)
	} else {
		r.Color = types.StringValue(color)
	}
	r.ColorHex = types.StringValue(l.Color)
	r.Description = types.StringValue(l.Description)
}

func (d *gitlabGroupLabelResourceModel) ResourceGitlabGroupLabelParseID(id string) (string, int, error) {
	group, rawLabelId, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	labelId, err := strconv.Atoi(rawLabelId)
	if err != nil {
		return "", 0, err
	}

	return group, labelId, nil
}
