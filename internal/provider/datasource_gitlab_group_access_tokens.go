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
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabGroupAccessTokensDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupAccessTokensDataSource{}
)

func init() {
	registerDataSource(NewGitlabGroupAccessTokensDataSource)
}

// NewGitlabGroupAccessTokensDataSource is a helper function to simplify the provider implementation.
func NewGitlabGroupAccessTokensDataSource() datasource.DataSource {
	return &gitlabGroupAccessTokensDataSource{}
}

// gitlabGroupAccessTokensDataSource is the data source implementation.
type gitlabGroupAccessTokensDataSource struct {
	client *gitlab.Client
}

// gitlabGroupAccessTokensDataSourceModel describes the data source data model.
type gitlabGroupAccessTokensDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Group        types.String `tfsdk:"group"`
	AccessTokens types.List   `tfsdk:"access_tokens"`
}

func accessTokenDataAttibutes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":           types.StringType,
		"group":        types.StringType,
		"name":         types.StringType,
		"user_id":      types.Int64Type,
		"access_level": types.StringType,
		"scopes":       types.SetType{ElemType: types.StringType},
		"created_at":   types.StringType,
		"expires_at":   types.StringType,
		"active":       types.BoolType,
		"revoked":      types.BoolType,
	}
}

// Metadata returns the data source type name.
func (d *gitlabGroupAccessTokensDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_access_tokens"
}

// Schema defines the schema for the data source.
func (d *gitlabGroupAccessTokensDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_access_tokens`" + ` data source allows to retrieve all group-level access tokens.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_access_tokens/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource.",
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The name or id of the group.",
				Required:            true,
			},
			"access_tokens": schema.ListAttribute{
				MarkdownDescription: "The list of access tokens returned by the search",
				Computed:            true,
				ElementType: types.ObjectType{
					AttrTypes: accessTokenDataAttibutes(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabGroupAccessTokensDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabGroupAccessTokensDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitlabGroupAccessTokensDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	group := state.Group.ValueString()
	options := &gitlab.ListGroupAccessTokensOptions{}
	options.Page = 1
	options.PerPage = 20

	var accessTokens []*gitlab.GroupAccessToken
	for options.Page != 0 {
		paginatedAccessTokens, resp1, err := d.client.GroupAccessTokens.ListGroupAccessTokens(group, options, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read group access tokens: %s", err.Error()))
			return
		}

		accessTokens = append(accessTokens, paginatedAccessTokens...)
		options.Page = resp1.NextPage
	}

	values, diags := flattenGitlabGroupAccessTokens(ctx, group, accessTokens)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	state.ID = types.StringValue(group)
	state.AccessTokens, diags = types.ListValue(types.ObjectType{AttrTypes: accessTokenDataAttibutes()}, values)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func flattenGitlabGroupAccessTokens(ctx context.Context, group string, accessTokens []*gitlab.GroupAccessToken) ([]attr.Value, diag.Diagnostics) {
	values := []attr.Value{}
	for _, accessToken := range accessTokens {
		value, diags := gitlabGroupAccessTokenToObjectValue(ctx, group, accessToken)
		if diags.HasError() {
			return nil, diags
		}
		values = append(values, value)
	}
	return values, nil
}

func gitlabGroupAccessTokenToObjectValue(ctx context.Context, group string, accessToken *gitlab.GroupAccessToken) (basetypes.ObjectValue, diag.Diagnostics) {
	scopes, diags := types.SetValueFrom(ctx, types.StringType, accessToken.Scopes)

	return types.ObjectValueMust(accessTokenDataAttibutes(), map[string]attr.Value{
		"id":           types.StringValue(strconv.FormatInt(accessToken.ID, 10)),
		"group":        types.StringValue(group),
		"name":         types.StringValue(accessToken.Name),
		"user_id":      types.Int64Value(int64(accessToken.UserID)),
		"access_level": types.StringValue(api.AccessLevelValueToName[accessToken.AccessLevel]),
		"created_at":   types.StringValue(accessToken.CreatedAt.String()),
		"expires_at":   types.StringValue(accessToken.ExpiresAt.String()),
		"active":       types.BoolValue(accessToken.Active),
		"revoked":      types.BoolValue(accessToken.Revoked),
		"scopes":       scopes,
	}), diags
}
