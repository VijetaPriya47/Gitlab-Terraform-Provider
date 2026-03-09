package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ datasource.DataSource              = &gitlabCurrentUserDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabCurrentUserDataSource{}
)

func init() {
	registerDataSource(NewGitlabCurrentUserDataSource)
}

func NewGitlabCurrentUserDataSource() datasource.DataSource {
	return &gitlabCurrentUserDataSource{}
}

type gitlabCurrentUserDataSource struct {
	client *gitlab.Client
}

type gitlabCurrentUserDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	GlobalID          types.String `tfsdk:"global_id"`
	Username          types.String `tfsdk:"username"`
	Name              types.String `tfsdk:"name"`
	Bot               types.Bool   `tfsdk:"bot"`
	GroupCount        types.Int64  `tfsdk:"group_count"`
	NamespaceID       types.String `tfsdk:"namespace_id"`
	GlobalNamespaceID types.String `tfsdk:"global_namespace_id"`
	PublicEmail       types.String `tfsdk:"public_email"`
}

func (d *gitlabCurrentUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_current_user"
}

func (d *gitlabCurrentUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_current_user`" + ` data source allows details of the current user (determined by ` + "`token`" + ` provider attribute) to be retrieved.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/index/#querycurrentuser)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the user.",
				Computed:            true,
			},
			"global_id": schema.StringAttribute{
				MarkdownDescription: "Global ID of the user. This is in the form of a GraphQL globally unique ID.",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username of the user. Unique within this instance of GitLab.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the user. Returns **** if the user is a project bot and the requester does not have permission to view the project.",
				Computed:            true,
			},
			"bot": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the user is a bot.",
				Computed:            true,
			},
			"group_count": schema.Int64Attribute{
				MarkdownDescription: "Group count for the user.",
				Computed:            true,
			},
			"namespace_id": schema.StringAttribute{
				MarkdownDescription: "Personal namespace of the user.",
				Computed:            true,
			},
			"global_namespace_id": schema.StringAttribute{
				MarkdownDescription: "Personal namespace of the user. This is in the form of a GraphQL globally unique ID.",
				Computed:            true,
			},
			"public_email": schema.StringAttribute{
				MarkdownDescription: "User's public email.",
				Computed:            true,
			},
		},
	}
}

func (d *gitlabCurrentUserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabCurrentUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabCurrentUserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := gitlab.GraphQLQuery{
		Query: `query {currentUser {name, bot, groupCount, id, namespace{id}, publicEmail, username}}`,
	}
	tflog.Debug(ctx, fmt.Sprintf("executing GraphQL Query %s to retrieve current user", query.Query))

	var response CurrentUserResponse
	if _, err := d.client.GraphQL.Do(query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read current user: %s", err.Error()))
		return
	}

	userID, err := api.ExtractIIDFromGlobalID(response.Data.CurrentUser.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing User ID", fmt.Sprintf("Unable to parse Global ID '%s' into user IID: %s", response.Data.CurrentUser.ID, err.Error()))
		return
	}

	namespaceID, err := api.ExtractIIDFromGlobalID(response.Data.CurrentUser.Namespace.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing Namespace ID", fmt.Sprintf("Unable to parse GlobalID '%s' into user namespace ID: %s", response.Data.CurrentUser.Namespace.ID, err.Error()))
		return
	}

	data.ID = types.StringValue(strconv.Itoa(userID))
	data.GlobalID = types.StringValue(response.Data.CurrentUser.ID)
	data.Username = types.StringValue(response.Data.CurrentUser.Username)
	data.Name = types.StringValue(response.Data.CurrentUser.Name)
	data.Bot = types.BoolValue(response.Data.CurrentUser.Bot)
	data.GroupCount = types.Int64Value(int64(response.Data.CurrentUser.GroupCount))
	data.NamespaceID = types.StringValue(strconv.Itoa(namespaceID))
	data.GlobalNamespaceID = types.StringValue(response.Data.CurrentUser.Namespace.ID)
	data.PublicEmail = types.StringValue(response.Data.CurrentUser.PublicEmail)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Struct representing current user based on the input API token
type CurrentUserResponse struct {
	Data struct {
		CurrentUser GraphQLUser `json:"currentUser"`
	} `json:"data"`
}

type GraphQLUser struct {
	Name       string `json:"name"`
	Bot        bool   `json:"bot"`
	GroupCount int    `json:"groupCount"`
	ID         string `json:"id"` // This is purposefully a string, as in some APIs it comes back as a globally unique ID
	Namespace  struct {
		ID string `json:"id"`
	} `json:"namespace"`
	PublicEmail string `json:"publicEmail"`
	Username    string `json:"username"`
}
