package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabProjectAccessTokensDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectAccessTokensDataSource{}
)

func init() {
	registerDataSource(newGitlabProjectAccessTokensDataSource)
}

// newGitlabProjectAccessTokensDataSource is a helper function to simplify the provider implementation.
func newGitlabProjectAccessTokensDataSource() datasource.DataSource {
	return &gitlabProjectAccessTokensDataSource{}
}

// gitlabProjectAccessTokensDataSource is the data source implementation.
type gitlabProjectAccessTokensDataSource struct {
	client *gitlab.Client
}

// gitlabProjectAccessTokensDataSourceModel describes the data source data model.
type gitlabProjectAccessTokensDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Project      types.String `tfsdk:"project"`
	State        types.String `tfsdk:"state"`
	AccessTokens types.List   `tfsdk:"access_tokens"`
}

func projectAccessTokenDataAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":           types.StringType,
		"project":      types.StringType,
		"description":  types.StringType,
		"name":         types.StringType,
		"user_id":      types.Int64Type,
		"access_level": types.StringType,
		"scopes":       types.SetType{ElemType: types.StringType},
		"created_at":   types.StringType,
		"expires_at":   types.StringType,
		"active":       types.BoolType,
		"revoked":      types.BoolType,
		"last_used_at": types.StringType,
	}
}

// Metadata returns the data source type name.
func (d *gitlabProjectAccessTokensDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_access_tokens"
}

// Schema defines the schema for the data source.
func (d *gitlabProjectAccessTokensDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	allowedStateValues := []string{"active", "inactive"}
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_access_tokens`" + ` data source allows to retrieve all project access tokens for a given project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_access_tokens/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project.",
				Required:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("List all project access token that match the specified state. Valid values are %s. Returns all project access token if not set.", utils.RenderValueListForDocs(allowedStateValues)),
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf(allowedStateValues...)},
			},
			"access_tokens": schema.ListAttribute{
				MarkdownDescription: "The list of access tokens returned by the search",
				Computed:            true,
				ElementType: types.ObjectType{
					AttrTypes: projectAccessTokenDataAttributes(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabProjectAccessTokensDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabProjectAccessTokensDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config gitlabProjectAccessTokensDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project := config.Project.ValueString()
	options := &gitlab.ListProjectAccessTokensOptions{}

	if !config.State.IsNull() && !config.State.IsUnknown() {
		options.State = config.State.ValueStringPointer()
	}

	accessTokens, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.ProjectAccessToken, *gitlab.Response, error) {
		return d.client.ProjectAccessTokens.ListProjectAccessTokens(project, options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project access tokens: %s", err.Error()))
		return
	}

	values, diags := flattenGitlabProjectAccessTokens(ctx, project, accessTokens)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	config.ID = types.StringValue(project)
	config.AccessTokens, diags = types.ListValue(types.ObjectType{AttrTypes: projectAccessTokenDataAttributes()}, values)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}

func flattenGitlabProjectAccessTokens(ctx context.Context, project string, accessTokens []*gitlab.ProjectAccessToken) ([]attr.Value, diag.Diagnostics) {
	values := []attr.Value{}
	for _, accessToken := range accessTokens {
		value, diags := gitlabProjectAccessTokenToObjectValue(ctx, project, accessToken)
		if diags.HasError() {
			return nil, diags
		}
		values = append(values, value)
	}
	return values, nil
}

func gitlabProjectAccessTokenToObjectValue(ctx context.Context, project string, accessToken *gitlab.ProjectAccessToken) (basetypes.ObjectValue, diag.Diagnostics) {
	scopes, diags := types.SetValueFrom(ctx, types.StringType, accessToken.Scopes)

	lastUsedAt := types.StringNull()
	if accessToken.LastUsedAt != nil {
		lastUsedAt = types.StringValue(accessToken.LastUsedAt.String())
	}

	return types.ObjectValueMust(projectAccessTokenDataAttributes(), map[string]attr.Value{
		"id":           types.StringValue(strconv.Itoa((accessToken.ID))),
		"project":      types.StringValue(project),
		"name":         types.StringValue(accessToken.Name),
		"description":  types.StringValue(accessToken.Description),
		"user_id":      types.Int64Value(int64(accessToken.UserID)),
		"access_level": types.StringValue(api.AccessLevelValueToName[accessToken.AccessLevel]),
		"created_at":   types.StringValue(accessToken.CreatedAt.String()),
		"expires_at":   types.StringValue(accessToken.ExpiresAt.String()),
		"active":       types.BoolValue(accessToken.Active),
		"revoked":      types.BoolValue(accessToken.Revoked),
		"scopes":       scopes,
		"last_used_at": lastUsedAt,
	}), diags
}
