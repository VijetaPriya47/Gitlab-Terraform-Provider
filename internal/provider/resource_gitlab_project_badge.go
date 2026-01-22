package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectBadgeResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectBadgeResource{}
	_ resource.ResourceWithImportState = &gitlabProjectBadgeResource{}
)

func init() {
	registerResource(NewGitLabProjectBadgeResource)
}

// NewGitLabProjectBadgeResource is a helper function to simplify the provider implementation.
func NewGitLabProjectBadgeResource() resource.Resource {
	return &gitlabProjectBadgeResource{}
}

type gitlabProjectBadgeResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Project          types.String `tfsdk:"project"`
	LinkURL          types.String `tfsdk:"link_url"`
	ImageURL         types.String `tfsdk:"image_url"`
	Name             types.String `tfsdk:"name"`
	RenderedLinkURL  types.String `tfsdk:"rendered_link_url"`
	RenderedImageURL types.String `tfsdk:"rendered_image_url"`
}

type gitlabProjectBadgeResource struct {
	client *gitlab.Client
}

func (r *gitlabProjectBadgeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_badge"
}

func (r *gitlabProjectBadgeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_badge`" + ` resource allows to manage the lifecycle of project badges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/user/project/badges/#project-badges)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project-id>:<badge-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project to add the badge to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"link_url": schema.StringAttribute{
				MarkdownDescription: "The url linked with the badge.",
				Required:            true,
			},
			"image_url": schema.StringAttribute{
				MarkdownDescription: "The image url which will be presented on project overview.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the badge.",
				Optional:            true,
				Computed:            true,
			},
			"rendered_link_url": schema.StringAttribute{
				MarkdownDescription: "The link_url argument rendered (in case of use of placeholders).",
				Computed:            true,
			},
			"rendered_image_url": schema.StringAttribute{
				MarkdownDescription: "The image_url argument rendered (in case of use of placeholders).",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabProjectBadgeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabProjectBadgeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectBadgeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectBadgeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	options := &gitlab.AddProjectBadgeOptions{
		LinkURL:  data.LinkURL.ValueStringPointer(),
		ImageURL: data.ImageURL.ValueStringPointer(),
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		options.Name = data.Name.ValueStringPointer()
	}

	badge, _, err := r.client.ProjectBadges.AddProjectBadge(project, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create project badge: %s", err.Error()))
		return
	}

	badgeID := strconv.FormatInt(badge.ID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &badgeID))
	data.modelToStateModel(badge, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectBadgeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectBadgeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, badgeID, err := ResourceGitlabProjectBadgeParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	badge, _, err := r.client.ProjectBadges.GetProjectBadge(project, badgeID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "Project badge doesn't exist anymore, removing from state", map[string]any{"err": err.Error()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project badge: %s", err.Error()))
		return
	}
	data.modelToStateModel(badge, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectBadgeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectBadgeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID to get project and badge IDs
	project, badgeID, err := ResourceGitlabProjectBadgeParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	options := &gitlab.EditProjectBadgeOptions{
		LinkURL:  data.LinkURL.ValueStringPointer(),
		ImageURL: data.ImageURL.ValueStringPointer(),
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		options.Name = data.Name.ValueStringPointer()
	}

	badge, _, err := r.client.ProjectBadges.EditProjectBadge(project, badgeID, options, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "Project badge doesn't exist anymore, removing from state", map[string]any{"err": err.Error()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update project badge: %s", err.Error()))
		return
	}

	data.modelToStateModel(badge, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectBadgeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectBadgeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID to get project and badge IDs
	project, badgeID, err := ResourceGitlabProjectBadgeParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	_, err = r.client.ProjectBadges.DeleteProjectBadge(project, badgeID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Project badge doesn't exist anymore, removing from state", map[string]any{"err": err.Error()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete project badge: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectBadgeResourceModel) modelToStateModel(b *gitlab.ProjectBadge, project string) {
	r.Project = types.StringValue(project)
	r.LinkURL = types.StringValue(b.LinkURL)
	r.ImageURL = types.StringValue(b.ImageURL)
	r.Name = types.StringValue(b.Name)
	r.RenderedLinkURL = types.StringValue(b.RenderedLinkURL)
	r.RenderedImageURL = types.StringValue(b.RenderedImageURL)
}

func ResourceGitlabProjectBadgeParseID(id string) (string, int64, error) {
	project, rawBadgeId, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	badgeId, err := strconv.ParseInt(rawBadgeId, 10, 64)
	if err != nil {
		return "", 0, err
	}

	return project, badgeId, nil
}
