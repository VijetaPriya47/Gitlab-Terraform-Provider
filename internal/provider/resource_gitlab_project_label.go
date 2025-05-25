package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                 = &gitlabProjectLabelResource{}
	_ resource.ResourceWithConfigure    = &gitlabProjectLabelResource{}
	_ resource.ResourceWithImportState  = &gitlabProjectLabelResource{}
	_ resource.ResourceWithUpgradeState = &gitlabProjectLabelResource{}
)

func init() {
	registerResource(NewGitLabProjectLabelResource)

	// Remove in 19.0
	registerResource(NewGitLabLabelResource)
}

func NewGitLabProjectLabelResource() resource.Resource {
	return &gitlabProjectLabelResource{
		ResourceName: "_project_label",
	}
}

// Remove in 19.0
func NewGitLabLabelResource() resource.Resource {
	return &gitlabProjectLabelResource{
		ResourceName:       "_label",
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_label` instead.",
	}
}

type gitlabProjectLabelResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LabelID     types.Int64  `tfsdk:"label_id"`
	Project     types.String `tfsdk:"project"`
	Name        types.String `tfsdk:"name"`
	Color       types.String `tfsdk:"color"`
	Description types.String `tfsdk:"description"`
}

type gitlabProjectLabelResource struct {
	client *gitlab.Client

	// Represents the name of the resource, since this resource uses both `gitlab_project_label` and `gitlab_label` for
	// backwards compatibility reasons. Should be removed in %19.0
	ResourceName       string
	DeprecationMessage string
}

func (r *gitlabProjectLabelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectLabelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectLabelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.getV1Schema()
}

func (r *gitlabProjectLabelResource) getV1Schema() schema.Schema {
	toReturn := schema.Schema{
		MarkdownDescription: `The ` + "`" + fmt.Sprintf(`gitlab%s`, r.ResourceName) + "`" + ` resource manages the lifecycle of a project label.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/labels/#get-a-single-project-label)`,
		DeprecationMessage: r.DeprecationMessage,
		Version:            1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project-id>:<label-name>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"label_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the project label.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project to add the label to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the label.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "The color of the label given in 6-digit hex notation with leading '#' sign (e.g. #FFAABB) or one of the [CSS color names](https://developer.mozilla.org/en-US/docs/Web/CSS/color_value#Color_keywords).",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the label.",
				Optional:            true,
			},
		},
	}

	return toReturn
}

func (r *gitlabProjectLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectLabelResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	options := &gitlab.CreateLabelOptions{
		Name:  gitlab.Ptr(data.Name.ValueString()),
		Color: gitlab.Ptr(data.Color.ValueString()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab label %s", *options.Name))

	label, _, err := r.client.Labels.CreateLabel(project, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to create project label: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(utils.BuildTwoPartID(&project, data.Name.ValueStringPointer()))
	data.modelToStateModel(label, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, labelName, err := data.ResourceGitlabProjectLabelParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	label, _, err := r.client.Labels.GetLabel(project, labelName, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("GitLab API error occured", fmt.Sprintf("Project label doesn't exist anymore, removing from state: %s", err.Error()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to get project label: %s", err.Error()))
		return
	}

	data.modelToStateModel(label, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectLabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project, labelName, err := data.ResourceGitlabProjectLabelParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	options := &gitlab.UpdateLabelOptions{
		Name:  gitlab.Ptr(data.Name.ValueString()),
		Color: gitlab.Ptr(data.Color.ValueString()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab label %s", data.ID.ValueString()))
	label, _, err := r.client.Labels.UpdateLabel(project, labelName, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to update project label: %s", err.Error()))
		return
	}

	data.modelToStateModel(label, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, labelName, err := data.ResourceGitlabProjectLabelParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	_, err = r.client.Labels.DeleteLabel(project, labelName, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to delete project label: %s", err.Error()))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *gitlabProjectLabelResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	schema := r.getV1Schema()

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data gitlabProjectLabelResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}

				newData := resourceGitlabProjectLabelStateUpgradeV0(ctx, &data)
				resp.Diagnostics.Append(resp.State.Set(ctx, &newData)...)
			},
		},
	}
}

// resourceGitlabProjectLabelStateUpgradeV0 performs the state migration from V0 to V1.
func resourceGitlabProjectLabelStateUpgradeV0(ctx context.Context, data *gitlabProjectLabelResourceModel) *gitlabProjectLabelResourceModel {
	oldID := data.ID.ValueString()
	project := data.Project.ValueString()
	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]any{"project": project, "v0-id": oldID})
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &oldID))
	tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]any{"v0-id": oldID, "v1-id": data.ID.ValueString()})
	return data
}

func (data *gitlabProjectLabelResourceModel) modelToStateModel(label *gitlab.Label, project string) {
	data.LabelID = types.Int64Value(int64(label.ID))
	data.Project = types.StringValue(project)
	data.Description = types.StringValue(label.Description)
	data.Color = types.StringValue(label.Color)
	data.Name = types.StringValue(label.Name)
}

func (d *gitlabProjectLabelResourceModel) ResourceGitlabProjectLabelParseID(id string) (string, string, error) {
	project, labelName, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", "", err
	}

	return project, labelName, nil
}
