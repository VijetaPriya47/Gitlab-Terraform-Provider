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
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabGroupLabelResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupLabelResource{}
	_ resource.ResourceWithImportState = &gitlabGroupLabelResource{}
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
	Description types.String `tfsdk:"description"`
}

type gitlabGroupLabelResource struct {
	client *gitlab.Client
}

func (r *gitlabGroupLabelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_label"
}

func (r *gitlabGroupLabelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_label`" + ` resource allows to manage the lifecycle of labels within a group.

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
				MarkdownDescription: "The color of the label given in 6-digit hex notation with leading '#' sign (e.g. #FFAABB) or one of the [CSS color names](https://developer.mozilla.org/en-US/docs/Web/CSS/color_value#Color_keywords).",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the label.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
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

	options := &gitlab.CreateGroupLabelOptions{
		Name:  gitlab.Ptr(data.Name.ValueString()),
		Color: gitlab.Ptr(data.Color.ValueString()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	label, _, err := r.client.GroupLabels.CreateGroupLabel(group, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to create group label: %s", err.Error()))
		return
	}

	labelID := strconv.Itoa(label.ID)
	data.ID = types.StringValue(utils.BuildTwoPartID(&group, &labelID))
	data.modelToStateModel(label, group)
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

	data.modelToStateModel(label, group)
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

	options := &gitlab.UpdateGroupLabelOptions{
		NewName: gitlab.Ptr(data.Name.ValueString()),
		Color:   gitlab.Ptr(data.Color.ValueString()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	label, _, err := r.client.GroupLabels.UpdateGroupLabel(group, labelID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to update group label: %s", err.Error()))
		return
	}

	data.modelToStateModel(label, group)
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

func (r *gitlabGroupLabelResourceModel) modelToStateModel(l *gitlab.GroupLabel, group string) {
	r.LabelID = types.Int64Value(int64(l.ID))
	r.Group = types.StringValue(group)
	r.Name = types.StringValue(l.Name)
	r.Color = types.StringValue(l.Color)
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
