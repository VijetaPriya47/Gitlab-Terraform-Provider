package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ datasource.DataSource              = &gitlabProjectIDsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectIDsDataSource{}
)

func NewGitLabProjectIDsDataSource() datasource.DataSource {
	return &gitlabProjectIDsDataSource{}
}

func init() {
	registerDataSource(NewGitLabProjectIDsDataSource)
}

// gitlabProjectIDsDataSource is the data source implementation.
type gitlabProjectIDsDataSource struct {
	client *gitlab.Client
}

// gitlabProjectIDsDataSourceModel describes the data source data model.
type gitlabProjectIDsDataSourceModel struct {
	Id               types.String `tfsdk:"id"`
	Project          types.String `tfsdk:"project"`
	ProjectId        types.String `tfsdk:"project_id"`
	ProjectFullPath  types.String `tfsdk:"project_full_path"`
	ProjectGraphQLID types.String `tfsdk:"project_graphql_id"`
}

// Metadata returns the data source type name.
func (d *gitlabProjectIDsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_ids"
}

// GetSchema defines the schema for the data source.
func (d *gitlabProjectIDsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_ids`" + ` data source identification information for a given project, allowing a user to translate a full path or ID into the GraphQL ID of the project.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/#queryproject)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project_id>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Computed:            true,
			},
			"project_full_path": schema.StringAttribute{
				MarkdownDescription: "The full path of the project.",
				Computed:            true,
			},
			"project_graphql_id": schema.StringAttribute{
				MarkdownDescription: "The GraphQL ID of the project.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabProjectIDsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectIDsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabProjectIDsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the API helper method to retrieve project identifiers
	ids, err := api.GetProjectGIDFromID(ctx, d.client, data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to retrieve project IDs from the API", fmt.Sprintf("Unable to retrieve project identifiers from the api based on the project value provided: %s", err))
		return
	}

	data.ProjectFullPath = types.StringValue(ids.ProjectFullPath)
	data.ProjectId = types.StringValue(strconv.FormatInt(ids.ProjectID, 10))
	data.ProjectGraphQLID = types.StringValue(ids.ProjectGQLID)

	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
