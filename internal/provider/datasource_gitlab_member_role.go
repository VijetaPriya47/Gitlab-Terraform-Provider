package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitLabMemberRoleDataSource{}
	_ datasource.DataSourceWithConfigure = &gitLabMemberRoleDataSource{}
)

func init() {
	registerDataSource(NewGitLabMemberRoleDataSource)
}

// NewGitLabMemberRoleDataSource is a helper function to simplify the provider implementation.
func NewGitLabMemberRoleDataSource() datasource.DataSource {
	return &gitLabMemberRoleDataSource{}
}

// gitLabMemberRoleDataSource is the data source implementation.
type gitLabMemberRoleDataSource struct {
	client *gitlab.Client
}

// gitLabMemberRoleDataSourceModel describes the data source data model.
type gitLabMemberRoleDataSourceModel struct {
	Id                 types.String   `tfsdk:"id"`
	Iid                types.Int64    `tfsdk:"iid"`
	Name               types.String   `tfsdk:"name"`
	Description        types.String   `tfsdk:"description"`
	EditPath           types.String   `tfsdk:"edit_path"`
	CreatedAt          types.String   `tfsdk:"created_at"`
	BaseAccessLevel    types.String   `tfsdk:"base_access_level"`
	EnabledPermissions []types.String `tfsdk:"enabled_permissions"`
}

// Metadata returns the data source type name.
func (d *gitLabMemberRoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_member_role"
}

// Schema defines the schema for the data source.
func (d *gitLabMemberRoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_member_role`" + ` data source allows details of a custom member role to be retrieved.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/#querymemberrole)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Globally unique ID of the member role. In the format of `gid://gitlab/MemberRole/1`",
				Required:            true,
			},
			"iid": schema.Int64Attribute{
				MarkdownDescription: "The member role ID integer value extracted from the `id` attribute",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the member role.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the member role.",
				Computed:            true,
			},
			"edit_path": schema.StringAttribute{
				MarkdownDescription: "The Web UI path to edit the member role",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the member role was created.",
				Computed:            true,
			},
			"base_access_level": schema.StringAttribute{
				MarkdownDescription: "The base access level of the custom role.",
				Computed:            true,
			},
			"enabled_permissions": schema.SetAttribute{
				MarkdownDescription: "All permissions enabled for the custom role.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitLabMemberRoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitLabMemberRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitLabMemberRoleDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id := data.Id.ValueString()

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			query {
				memberRole(id: "%s") {
					baseAccessLevel {
						stringValue
					},
					createdAt,
					description,
					editPath,
					enabledPermissions {
						nodes {
							value
						}
					},
					id,
					name,
				}
			}`, id),
	}
	tflog.Debug(ctx, "executing GraphQL Query to retrieve current custom member role", map[string]any{
		"query": query.Query,
	})

	var response MemberRoleResponse
	if _, err := d.client.GraphQL.Do(query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read member role details: %s", err.Error()))
		return
	}
	if response.Data.MemberRole.ID == "" {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Member role does not exist: %s", id))
		return
	}

	data.Id = types.StringValue(response.Data.MemberRole.ID)
	data.Name = types.StringValue(response.Data.MemberRole.Name)
	data.Description = types.StringValue(response.Data.MemberRole.Description)
	data.EditPath = types.StringValue(response.Data.MemberRole.EditPath)
	data.CreatedAt = types.StringValue(response.Data.MemberRole.CreatedAt)
	data.BaseAccessLevel = types.StringValue(response.Data.MemberRole.BaseAccessLevel.StringValue)

	enabledPermissions := []basetypes.StringValue{}
	for _, v := range response.Data.MemberRole.EnabledPermissions.Nodes {
		enabledPermissions = append(enabledPermissions, types.StringValue(v.Value))
	}
	data.EnabledPermissions = enabledPermissions

	iid, err := api.ExtractIIDFromGlobalID(response.Data.MemberRole.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID format", fmt.Sprintf("Unable to extract IID from global ID: %s", err.Error()))
		return
	}
	data.Iid = types.Int64Value(int64(iid))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
