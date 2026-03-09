package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabReleaseLinkDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabReleaseLinkDataSource{}
)

func init() {
	registerDataSource(NewGitlabReleaseLinkDataSource)
}

func NewGitlabReleaseLinkDataSource() datasource.DataSource {
	return &gitlabReleaseLinkDataSource{}
}

type gitlabReleaseLinkDataSource struct {
	client *gitlab.Client
}

type gitlabReleaseLinkDataSourceModel struct {
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

func (d *gitlabReleaseLinkDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release_link"
}

var validLinkTypes = []string{"other", "runbook", "image", "package"}

func (d *gitlabReleaseLinkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_release_link`" + ` data source allows you to get details of a release link.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/releases/links/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this data source.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or Namespace path of the project.",
				Required:            true,
			},
			"tag_name": schema.StringAttribute{
				MarkdownDescription: "The tag associated with the Release.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the link. Link names must be unique within the release.",
				Computed:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL of the link. Link URLs must be unique within the release.",
				Computed:            true,
			},
			"filepath": schema.StringAttribute{
				MarkdownDescription: "Relative path for a [Direct Asset link](https://docs.gitlab.com/user/project/releases/release_fields/#permanent-links-to-latest-release-assets).",
				Computed:            true,
			},
			"link_type": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The type of the link. Valid values are %s.", utils.RenderValueListForDocs(validLinkTypes)),
				Computed:            true,
			},
			"link_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the link.",
				Required:            true,
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

func (d *gitlabReleaseLinkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabReleaseLinkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabReleaseLinkDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	tagName := data.TagName.ValueString()
	linkID := data.LinkID.ValueInt64()

	releaseLink, _, err := d.client.ReleaseLinks.GetReleaseLink(project, tagName, linkID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project release links: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%d", project, tagName, linkID))
	data.Project = types.StringValue(project)
	data.TagName = types.StringValue(tagName)
	data.Name = types.StringValue(releaseLink.Name)
	data.URL = types.StringValue(releaseLink.URL)
	data.LinkType = types.StringValue(string(releaseLink.LinkType))
	data.LinkID = types.Int64Value(int64(releaseLink.ID))
	data.DirectAssetURL = types.StringValue(releaseLink.DirectAssetURL)
	data.External = types.BoolValue(releaseLink.External)

	directAssetLinkArray := strings.SplitN(releaseLink.DirectAssetURL, "downloads", 2)
	if len(directAssetLinkArray) > 1 {
		data.Filepath = types.StringValue(directAssetLinkArray[1])
	} else {
		data.Filepath = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
