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
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabGroupBadgeResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupBadgeResource{}
	_ resource.ResourceWithImportState = &gitlabGroupBadgeResource{}
)

func init() {
	registerResource(NewGitLabGroupBadgeResource)
}

// NewGitLabGroupBadgeResource is a helper function to simplify the provider implementation.
func NewGitLabGroupBadgeResource() resource.Resource {
	return &gitlabGroupBadgeResource{}
}

type gitlabGroupBadgeResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Group            types.String `tfsdk:"group"`
	LinkURL          types.String `tfsdk:"link_url"`
	ImageURL         types.String `tfsdk:"image_url"`
	Name             types.String `tfsdk:"name"`
	RenderedLinkURL  types.String `tfsdk:"rendered_link_url"`
	RenderedImageURL types.String `tfsdk:"rendered_image_url"`
}

type gitlabGroupBadgeResource struct {
	client *gitlab.Client
}

func (r *gitlabGroupBadgeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_badge"
}

func (r *gitlabGroupBadgeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_badge`" + ` resource allows to manage the lifecycle of group badges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/user/project/badges/#group-badges)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group-id>:<badge-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the group to add the badge to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"link_url": schema.StringAttribute{
				MarkdownDescription: "The url linked with the badge.",
				Required:            true,
			},
			"image_url": schema.StringAttribute{
				MarkdownDescription: "The image url which will be presented on group overview.",
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

func (r *gitlabGroupBadgeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabGroupBadgeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabGroupBadgeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupBadgeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := data.Group.ValueString()
	options := &gitlab.AddGroupBadgeOptions{
		LinkURL:  gitlab.Ptr(data.LinkURL.ValueString()),
		ImageURL: gitlab.Ptr(data.ImageURL.ValueString()),
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		options.Name = gitlab.Ptr(data.Name.ValueString())
	}

	badge, _, err := r.client.GroupBadges.AddGroupBadge(group, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to create group badge: %s", err.Error()))
		return
	}

	badgeID := strconv.FormatInt(badge.ID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&group, &badgeID))
	data.modelToStateModel(badge, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupBadgeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupBadgeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, badgeID, err := data.ResourceGitlabGroupBadgeParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	badge, _, err := r.client.GroupBadges.GetGroupBadge(group, badgeID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("GitLab API error occured", fmt.Sprintf("Group badge doesn't exist anymore, removing from state: %s", err.Error()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to get group badge: %s", err.Error()))
		return
	}
	data.modelToStateModel(badge, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupBadgeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabGroupBadgeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID to get group and badge IDs
	group, badgeID, err := data.ResourceGitlabGroupBadgeParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	options := &gitlab.EditGroupBadgeOptions{
		LinkURL:  gitlab.Ptr(data.LinkURL.ValueString()),
		ImageURL: gitlab.Ptr(data.ImageURL.ValueString()),
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		options.Name = gitlab.Ptr(data.Name.ValueString())
	}

	badge, _, err := r.client.GroupBadges.EditGroupBadge(group, badgeID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to update group badge: %s", err.Error()))
		return
	}

	data.modelToStateModel(badge, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupBadgeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupBadgeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID to get group and badge IDs
	group, badgeID, err := data.ResourceGitlabGroupBadgeParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	_, err = r.client.GroupBadges.DeleteGroupBadge(group, badgeID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to delete group badge: %s", err.Error()))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *gitlabGroupBadgeResourceModel) modelToStateModel(b *gitlab.GroupBadge, group string) {
	r.Group = types.StringValue(group)
	r.LinkURL = types.StringValue(b.LinkURL)
	r.ImageURL = types.StringValue(b.ImageURL)
	r.Name = types.StringValue(b.Name)
	r.RenderedLinkURL = types.StringValue(b.RenderedLinkURL)
	r.RenderedImageURL = types.StringValue(b.RenderedImageURL)
}

func (d *gitlabGroupBadgeResourceModel) ResourceGitlabGroupBadgeParseID(id string) (string, int64, error) {
	group, rawBadgeId, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	badgeId, err := strconv.ParseInt(rawBadgeId, 10, 64)
	if err != nil {
		return "", 0, err
	}

	return group, badgeId, nil
}
