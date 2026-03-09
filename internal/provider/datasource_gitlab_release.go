package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabReleaseDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabReleaseDataSource{}
)

func init() {
	registerDataSource(NewGitLabReleaseDataSource)
}

// NewGitLabApplicationDataSource is a helper function to simplify the provider implementation.
func NewGitLabReleaseDataSource() datasource.DataSource {
	return &gitlabReleaseDataSource{}
}

// gitlabReleaseDataSource is the data source implementation.
type gitlabReleaseDataSource struct {
	client *gitlab.Client
}

// gitLabReleaseDataSourceModel describes the data source data model.
type gitLabReleaseDataSourceModel struct {
	Id          types.String             `tfsdk:"id"`
	ProjectId   types.String             `tfsdk:"project_id"`
	TagName     types.String             `tfsdk:"tag_name"`
	Description types.String             `tfsdk:"description"`
	Name        types.String             `tfsdk:"name"`
	CreatedAt   types.String             `tfsdk:"created_at"`
	ReleasedAt  types.String             `tfsdk:"released_at"`
	Assets      *gitlabReleaseDataAssets `tfsdk:"assets"`
}

// The assets for a release
type gitlabReleaseDataAssets struct {
	Count   types.Int64                     `tfsdk:"count"`
	Sources []*gitlabReleaseDataAssetSource `tfsdk:"sources"`
	Links   []*gitlabReleaseDataAssetLink   `tfsdk:"links"`
}

// The source assets for a release
type gitlabReleaseDataAssetSource struct {
	Format types.String `tfsdk:"format"`
	Url    types.String `tfsdk:"url"`
}

// The link assets for a release
type gitlabReleaseDataAssetLink struct {
	ID       types.Int64  `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Url      types.String `tfsdk:"url"`
	LinkType types.String `tfsdk:"link_type"`
}

// Metadata returns the data source type name.
func (d *gitlabReleaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release"
}

// GetSchema defines the schema for the data source.
func (d *gitlabReleaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_release`" + ` data source retrieves information about a gitlab release for a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/releases/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project_id:tag_name>`.",
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
			},
			"tag_name": schema.StringAttribute{
				MarkdownDescription: "The Git tag the release is associated with.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "An HTML rendered description of the release.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the release.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The date the release was created.",
				Computed:            true,
			},
			"released_at": schema.StringAttribute{
				MarkdownDescription: "The date the release was created.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{

			// The assets block stores the assets, both source and link,
			// for the reslse
			"assets": schema.SingleNestedBlock{
				MarkdownDescription: "The assets for a release",
				Attributes: map[string]schema.Attribute{
					"count": schema.Int64Attribute{
						MarkdownDescription: "The number of assets for a release",
						Computed:            true,
					},
				},
				Blocks: map[string]schema.Block{

					// The "Sources" stores the source links
					// for the release
					"sources": schema.ListNestedBlock{
						MarkdownDescription: "The sources for a release",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"format": schema.StringAttribute{
									MarkdownDescription: "The format of the source",
									Computed:            true,
								},
								"url": schema.StringAttribute{
									MarkdownDescription: "The URL of the source",
									Computed:            true,
								},
							},
						},
					},

					// The "Links" stores the custom release links
					// for the release
					"links": schema.ListNestedBlock{
						MarkdownDescription: "The links for a release",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.Int64Attribute{
									MarkdownDescription: "The ID of the link",
									Computed:            true,
								},
								"name": schema.StringAttribute{
									MarkdownDescription: "The name of the link",
									Computed:            true,
								},
								"url": schema.StringAttribute{
									MarkdownDescription: "The URL of the link",
									Computed:            true,
								},
								"link_type": schema.StringAttribute{
									MarkdownDescription: "The type of the link",
									Computed:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabReleaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabReleaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabReleaseDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Make API call to read applications
	release, _, err := d.client.Releases.GetRelease(state.ProjectId.ValueString(), state.TagName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read release details: %s", err.Error()))
		return
	}

	// Set the ID
	state.Id = types.StringValue(utils.BuildTwoPartID(state.ProjectId.ValueStringPointer(), state.TagName.ValueStringPointer()))
	state.Description = types.StringValue(release.Description)
	state.Name = types.StringValue(release.Name)
	state.CreatedAt = types.StringValue(release.CreatedAt.Format(time.RFC3339))
	state.ReleasedAt = types.StringValue(release.ReleasedAt.Format(time.RFC3339))
	state.Assets = &gitlabReleaseDataAssets{
		Count: types.Int64Value(int64(release.Assets.Count)),
	}

	// Loop over the asset sources, and store each in the state
	for _, v := range release.Assets.Sources {
		state.Assets.Sources = append(state.Assets.Sources, &gitlabReleaseDataAssetSource{
			Format: types.StringValue(v.Format),
			Url:    types.StringValue(v.URL),
		})
	}

	// Loop over the asset Links, and store each in the state
	for _, v := range release.Assets.Links {
		state.Assets.Links = append(state.Assets.Links, &gitlabReleaseDataAssetLink{
			ID:   types.Int64Value(int64(v.ID)),
			Name: types.StringValue(v.Name),
			Url:  types.StringValue(v.URL),

			// this is a basetype of string, so the cast is safe
			LinkType: types.StringValue(string(v.LinkType)),
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
