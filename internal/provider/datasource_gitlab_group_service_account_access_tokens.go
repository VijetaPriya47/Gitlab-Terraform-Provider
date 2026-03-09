package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabGroupServiceAccountAccessTokensDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupServiceAccountAccessTokensDataSource{}
)

func init() {
	registerDataSource(NewGitlabGroupServiceAccountAccessTokensDataSource)
}

// NewGitlabGroupServiceAccountAccessTokensDataSource is a helper function to simplify the provider implementation.
func NewGitlabGroupServiceAccountAccessTokensDataSource() datasource.DataSource {
	return &gitlabGroupServiceAccountAccessTokensDataSource{}
}

// gitlabGroupServiceAccountAccessTokensDataSource is the data source implementation.
type gitlabGroupServiceAccountAccessTokensDataSource struct {
	client *gitlab.Client
}

// gitlabGroupServiceAccountAccessTokensDataSourceModel describes the data source data model.
type gitlabGroupServiceAccountAccessTokensDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Group            types.String `tfsdk:"group"`
	ServiceAccountID types.Int64  `tfsdk:"service_account_id"`
	AccessTokens     types.List   `tfsdk:"access_tokens"`
}

func groupServiceAccountAccessTokenDataAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":         types.StringType,
		"name":       types.StringType,
		"scopes":     types.SetType{ElemType: types.StringType},
		"created_at": types.StringType,
		"expires_at": types.StringType,
		"active":     types.BoolType,
		"revoked":    types.BoolType,
	}
}

// Metadata returns the data source type name.
func (d *gitlabGroupServiceAccountAccessTokensDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_service_account_access_tokens"
}

// Schema defines the schema for the data source.
func (d *gitlabGroupServiceAccountAccessTokensDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_service_account_access_tokens`" + ` data source allows to retrieve all access tokens for a group service account.

~> **Note:** The data source returns the token metadata only. The token value is not available.

~> **Permissions:** You must have administrator access or be an Owner of the group to list the tokens of a service account.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/personal_access_tokens/#list-all-personal-access-tokens)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group>:<user_id>`.",
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the group containing the service account. Must be a top level group.",
				Required:            true,
			},
			"service_account_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the service account user.",
				Required:            true,
			},
			"access_tokens": schema.ListAttribute{
				MarkdownDescription: "The list of access tokens for the service account.",
				Computed:            true,
				ElementType: types.ObjectType{
					AttrTypes: groupServiceAccountAccessTokenDataAttributes(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabGroupServiceAccountAccessTokensDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabGroupServiceAccountAccessTokensDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitlabGroupServiceAccountAccessTokensDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	group := state.Group.ValueString()
	userID := state.ServiceAccountID.ValueInt64()
	userIDStr := strconv.FormatInt(userID, 10)

	// Validate that the service account belongs to the specified group
	_, _, err := findGitlabServiceAccount(d.client, group, userIDStr)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to find service account %s in group %s: %s", userIDStr, group, err.Error()))
		return
	}

	options := &gitlab.ListPersonalAccessTokensOptions{
		UserID: gitlab.Ptr(userID),
	}
	options.Page = 1
	options.PerPage = 20

	var accessTokens []*gitlab.PersonalAccessToken
	for options.Page != 0 {
		paginatedAccessTokens, apiResp, err := d.client.PersonalAccessTokens.ListPersonalAccessTokens(options, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read service account access tokens: %s", err.Error()))
			return
		}

		accessTokens = append(accessTokens, paginatedAccessTokens...)
		options.Page = apiResp.NextPage
	}

	values, diags := flattenGroupServiceAccountAccessTokens(ctx, accessTokens)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	state.ID = types.StringValue(utils.BuildTwoPartID(&group, &userIDStr))
	state.AccessTokens, diags = types.ListValue(types.ObjectType{AttrTypes: groupServiceAccountAccessTokenDataAttributes()}, values)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func flattenGroupServiceAccountAccessTokens(ctx context.Context, accessTokens []*gitlab.PersonalAccessToken) ([]attr.Value, diag.Diagnostics) {
	values := []attr.Value{}
	for _, accessToken := range accessTokens {
		value, diags := groupServiceAccountAccessTokenToObjectValue(ctx, accessToken)
		if diags.HasError() {
			return nil, diags
		}
		values = append(values, value)
	}
	return values, nil
}

func groupServiceAccountAccessTokenToObjectValue(ctx context.Context, accessToken *gitlab.PersonalAccessToken) (basetypes.ObjectValue, diag.Diagnostics) {
	scopes, diags := types.SetValueFrom(ctx, types.StringType, accessToken.Scopes)

	expiresAt := types.StringNull()
	if accessToken.ExpiresAt != nil {
		expiresAt = types.StringValue(accessToken.ExpiresAt.String())
	}

	createdAt := types.StringNull()
	if accessToken.CreatedAt != nil {
		createdAt = types.StringValue(accessToken.CreatedAt.String())
	}

	return types.ObjectValueMust(groupServiceAccountAccessTokenDataAttributes(), map[string]attr.Value{
		"id":         types.StringValue(strconv.FormatInt(accessToken.ID, 10)),
		"name":       types.StringValue(accessToken.Name),
		"created_at": createdAt,
		"expires_at": expiresAt,
		"active":     types.BoolValue(accessToken.Active),
		"revoked":    types.BoolValue(accessToken.Revoked),
		"scopes":     scopes,
	}), diags
}
