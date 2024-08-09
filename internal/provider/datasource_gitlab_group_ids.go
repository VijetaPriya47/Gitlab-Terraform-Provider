package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ datasource.DataSource              = &gitlabGroupIDsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupIDsDataSource{}
)

func NewGitLabGroupIDsDataSource() datasource.DataSource {
	return &gitlabGroupIDsDataSource{}
}

func init() {
	registerDataSource(NewGitLabGroupIDsDataSource)
}

// gitlabGroupIDsDataSource is the data source implementation.
type gitlabGroupIDsDataSource struct {
	client *gitlab.Client
}

// gitlabGroupIDsDataSourceModel describes the data source data model.
type gitlabGroupIDsDataSourceModel struct {
	Id             types.String `tfsdk:"id"`
	Group          types.String `tfsdk:"group"`
	GroupId        types.String `tfsdk:"group_id"`
	GroupFullPath  types.String `tfsdk:"group_full_path"`
	GroupGraphQLID types.String `tfsdk:"group_graphql_id"`
}

// Metadata returns the data source type name.
func (d *gitlabGroupIDsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_ids"
}

// GetSchema defines the schema for the data source.
func (d *gitlabGroupIDsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_ids`" + ` data source identification information for a given group, allowing a user to translate a full path or ID into the GraphQL ID of the group.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/ee/api/graphql/reference/#querygroup)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group_id>`.",
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the group.",
				Required:            true,
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the group.",
				Computed:            true,
			},
			"group_full_path": schema.StringAttribute{
				MarkdownDescription: "The full path of the group.",
				Computed:            true,
			},
			"group_graphql_id": schema.StringAttribute{
				MarkdownDescription: "The GraphQL ID of the group.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabGroupIDsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*gitlab.Client)
}

func (d *gitlabGroupIDsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabGroupIDsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the API helper method to retrieve group identifiers
	ids, err := api.GetGroupGIDFromID(ctx, d.client, data.Group.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to retrieve group IDs from the API", fmt.Sprintf("Unable to retrieve group identifiers from the api based on the group value provided: %s", err))
		return
	}

	data.GroupFullPath = types.StringValue(ids.GroupFullPath)
	data.GroupId = types.StringValue(strconv.Itoa(ids.GroupID))
	data.GroupGraphQLID = types.StringValue(ids.GroupGQLID)

	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
