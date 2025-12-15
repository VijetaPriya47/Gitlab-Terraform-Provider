package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabReleaseLinkResource{}
	_ resource.ResourceWithConfigure   = &gitlabReleaseLinkResource{}
	_ resource.ResourceWithImportState = &gitlabReleaseLinkResource{}
)

func init() {
	registerResource(NewGitlabReleaseLinkResource)
}

func NewGitlabReleaseLinkResource() resource.Resource {
	return &gitlabReleaseLinkResource{}
}

type gitlabReleaseLinkResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Project        types.String `tfsdk:"project"`
	TagName        types.String `tfsdk:"tag_name"`
	Name           types.String `tfsdk:"name"`
	URL            types.String `tfsdk:"url"`
	Filepath       types.String `tfsdk:"filepath"`
	LinkType       types.String `tfsdk:"link_type"`
	LinkID         types.Int64  `tfsdk:"link_id"`
	DirectAssetURL types.String `tfsdk:"direct_asset_url"`
	External       types.Bool   `tfsdk:"external"`
}

type gitlabReleaseLinkResource struct {
	client *gitlab.Client
}

func (r *gitlabReleaseLinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release_link"
}

func (r *gitlabReleaseLinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	validLinkTypes := []string{"other", "runbook", "image", "package"}
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_release_link`" + ` resource allows to manage the lifecycle of a release link.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/releases/links/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource. In the format `<project:tag-name:release-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or Namespace path of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tag_name": schema.StringAttribute{
				MarkdownDescription: "The tag associated with the Release.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the link. Link names must be unique within the release.",
				Required:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL of the link. Link URLs must be unique within the release.",
				Required:            true,
			},
			"filepath": schema.StringAttribute{
				MarkdownDescription: "Relative path for a [Direct Asset link](https://docs.gitlab.com/user/project/releases/release_fields/#permanent-links-to-latest-release-assets).",
				Optional:            true,
			},
			"link_type": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The type of the link. Valid values are %s. Defaults to %s.", utils.RenderValueListForDocs(validLinkTypes), validLinkTypes[0]),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(validLinkTypes[0]),
				Validators:          []validator.String{stringvalidator.OneOf(validLinkTypes...)},
			},
			"link_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the link.",
				Computed:            true,
			},
			"direct_asset_url": schema.StringAttribute{
				MarkdownDescription: "Full path for a [Direct Asset link](https://docs.gitlab.com/user/project/releases/release_fields/#permanent-links-to-latest-release-assets).",
				Computed:            true,
			},
			"external": schema.BoolAttribute{
				MarkdownDescription: "External or internal link.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabReleaseLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabReleaseLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabReleaseLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabReleaseLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	tagName := data.TagName.ValueString()
	name := data.Name.ValueString()
	url := data.URL.ValueString()

	options := &gitlab.CreateReleaseLinkOptions{
		Name: gitlab.Ptr(name),
		URL:  gitlab.Ptr(url),
	}
	if !data.Filepath.IsNull() && !data.Filepath.IsUnknown() {
		options.FilePath = data.Filepath.ValueStringPointer()
	}
	if !data.LinkType.IsNull() && !data.LinkType.IsUnknown() {
		linkTypeValue := gitlab.LinkTypeValue(data.LinkType.ValueString())
		options.LinkType = &linkTypeValue
	}

	tflog.Debug(ctx, fmt.Sprintf("create release link project/tagName/name: %s/%s/%s", project, tagName, name))
	releaseLink, _, err := r.client.ReleaseLinks.CreateReleaseLink(project, tagName, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create release link: %s", err.Error()))
		return
	}
	data.ID = types.StringValue(resourceGitLabReleaseLinkBuildId(project, tagName, releaseLink.ID))

	data.modelToStateModel(project, tagName, releaseLink)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabReleaseLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabReleaseLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, tagName, linkID, err := resourceGitLabReleaseLinkParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read release link project/tagName/linkID: %s/%s/%d", project, tagName, linkID))
	releaseLink, _, err := r.client.ReleaseLinks.GetReleaseLink(project, tagName, linkID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, fmt.Sprintf("recieved 404 for release link project/tagName/linkID: %s/%s/%d. Removing from state", project, tagName, linkID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read release link: %s", err.Error()))
		return
	}

	data.modelToStateModel(project, tagName, releaseLink)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabReleaseLinkResourceModel) modelToStateModel(project string, tagName string, releaseLink *gitlab.ReleaseLink) {
	d.Project = types.StringValue(project)
	d.TagName = types.StringValue(tagName)
	d.Name = types.StringValue(releaseLink.Name)
	d.URL = types.StringValue(releaseLink.URL)
	d.LinkType = types.StringValue(string(releaseLink.LinkType))
	d.LinkID = types.Int64Value(int64(releaseLink.ID))
	d.DirectAssetURL = types.StringValue(releaseLink.DirectAssetURL)
	d.External = types.BoolValue(releaseLink.External)

	directAssetLinkArray := strings.SplitN(releaseLink.DirectAssetURL, "downloads", 2)
	if len(directAssetLinkArray) > 1 {
		d.Filepath = types.StringValue(directAssetLinkArray[1])
	} else {
		d.Filepath = types.StringNull()
	}
}

func (r *gitlabReleaseLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabReleaseLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, tagName, linkID, err := resourceGitLabReleaseLinkParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	options := &gitlab.UpdateReleaseLinkOptions{}
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		options.Name = data.Name.ValueStringPointer()
	}
	if !data.URL.IsNull() && !data.URL.IsUnknown() {
		options.URL = data.URL.ValueStringPointer()
	}
	if !data.Filepath.IsNull() && !data.Filepath.IsUnknown() {
		options.FilePath = data.Filepath.ValueStringPointer()
	}
	if !data.LinkType.IsNull() && !data.LinkType.IsUnknown() {
		linkTypeValue := gitlab.LinkTypeValue(data.LinkType.ValueString())
		options.LinkType = &linkTypeValue
	}

	tflog.Debug(ctx, fmt.Sprintf("update release link project/tagName/linkID: %s/%s/%d", project, tagName, linkID))
	releaseLink, _, err := r.client.ReleaseLinks.UpdateReleaseLink(project, tagName, linkID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update release link: %s", err.Error()))
		return
	}

	data.modelToStateModel(project, tagName, releaseLink)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabReleaseLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabReleaseLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, tagName, linkID, err := resourceGitLabReleaseLinkParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("delete release link project/tagName/linkID: %s/%s/%d", project, tagName, linkID))
	_, _, err = r.client.ReleaseLinks.DeleteReleaseLink(project, tagName, linkID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete release link: %s", err.Error()))
		return
	}
}

func resourceGitLabReleaseLinkParseId(id string) (string, string, int64, error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("unexpected ID format (%q). Expected project:tagName:linkID", id)
	}

	linkID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return "", "", 0, err
	}

	return parts[0], parts[1], linkID, nil
}

func resourceGitLabReleaseLinkBuildId(project string, tagName string, linkID int64) string {
	id := fmt.Sprintf("%s:%s:%d", project, tagName, linkID)
	return id
}
