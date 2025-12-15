package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabReleaseLinksDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabReleaseLinksDataSource{}
)

func init() {
	registerDataSource(NewGitlabReleaseLinksDataSource)
}

func NewGitlabReleaseLinksDataSource() datasource.DataSource {
	return &gitlabReleaseLinksDataSource{}
}

type gitlabReleaseLinksDataSource struct {
	client *gitlab.Client
}

type gitlabReleaseLinksDataSourceModel struct {
	ID           types.String                                  `tfsdk:"id"`
	Project      types.String                                  `tfsdk:"project"`
	TagName      types.String                                  `tfsdk:"tag_name"`
	ReleaseLinks []gitlabReleaseLinksIndividualDataSourceModel `tfsdk:"release_links"`
}

type gitlabReleaseLinksIndividualDataSourceModel struct {
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

func (d *gitlabReleaseLinksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release_links"
}

func (d *gitlabReleaseLinksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_release_links`" + ` data source allows get details of release links.

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
			"release_links": schema.ListNestedAttribute{
				Description: "List of release links",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project": schema.StringAttribute{
							MarkdownDescription: "The ID or Namespace path of the project.",
							Computed:            true,
						},
						"tag_name": schema.StringAttribute{
							MarkdownDescription: "The tag associated with the Release.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *gitlabReleaseLinksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabReleaseLinksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabReleaseLinksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	tagName := data.TagName.ValueString()
	options := gitlab.ListReleaseLinksOptions{}

	releaseLinks, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.ReleaseLink, *gitlab.Response, error) {
		return d.client.ReleaseLinks.ListReleaseLinks(project, tagName, &options, p, gitlab.WithContext(ctx))
	})
	if err != nil && !api.Is404(err) {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project release links: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("get list release links project/tagName: %s/%s", project, tagName))
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &tagName))
	data.Project = types.StringValue(project)
	data.TagName = types.StringValue(tagName)

	data.ReleaseLinks = []gitlabReleaseLinksIndividualDataSourceModel{}
	for _, releaseLink := range releaseLinks {
		modelReleaseLink := gitlabReleaseLinksIndividualDataSourceModel{
			Project:        types.StringValue(project),
			TagName:        types.StringValue(tagName),
			Name:           types.StringValue(releaseLink.Name),
			URL:            types.StringValue(releaseLink.URL),
			LinkType:       types.StringValue(string(releaseLink.LinkType)),
			LinkID:         types.Int64Value(int64(releaseLink.ID)),
			DirectAssetURL: types.StringValue(releaseLink.DirectAssetURL),
			External:       types.BoolValue(releaseLink.External),
		}
		directAssetLinkArray := strings.SplitN(releaseLink.DirectAssetURL, "downloads", 2)
		if len(directAssetLinkArray) > 1 {
			modelReleaseLink.Filepath = types.StringValue(directAssetLinkArray[1])
		} else {
			modelReleaseLink.Filepath = types.StringValue("")
		}
		data.ReleaseLinks = append(data.ReleaseLinks, modelReleaseLink)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
