package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabGroupServiceAccountDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupServiceAccountDataSource{}
)

func init() {
	registerDataSource(NewGitlabGroupServiceAccountDataSource)
}

// NewGitlabGroupServiceAccountDataSource is a helper function to simplify the provider implementation.
func NewGitlabGroupServiceAccountDataSource() datasource.DataSource {
	return &gitlabGroupServiceAccountDataSource{}
}

// gitlabGroupServiceAccountDataSource is the data source implementation.
type gitlabGroupServiceAccountDataSource struct {
	client *gitlab.Client
}

// gitlabGroupServiceAccountDataSourceModel describes the data source data model.
type gitlabGroupServiceAccountDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
	Group            types.String `tfsdk:"group"`
	Name             types.String `tfsdk:"name"`
	Username         types.String `tfsdk:"username"`
}

// Metadata returns the data source type name.
func (d *gitlabGroupServiceAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_service_account"
}

// Schema defines the schema for the data source.
func (d *gitlabGroupServiceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_service_account`" + ` data source retrieves information about a gitlab service account for a group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/group_service_accounts.html#list-service-account-users)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group>:<service_account_id>`.",
				Computed:            true,
			},
			"service_account_id": schema.StringAttribute{
				MarkdownDescription: "The service account id.",
				Required:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the target group. Must be a top-level group.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user. If not specified, the default Service account user name is used.",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the user. If not specified, it's automatically generated.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabGroupServiceAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabGroupServiceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitlabGroupServiceAccountDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Make API call to read service accounts
	serviceAccount, err := findGitlabServiceAccount(d.client, state.Group.ValueString(), state.ServiceAccountID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read service account details: %s", err.Error()))
		return
	}

	// Set the ID
	state.ID = types.StringValue(utils.BuildTwoPartID(state.Group.ValueStringPointer(), state.ServiceAccountID.ValueStringPointer()))
	state.ServiceAccountID = types.StringValue(strconv.Itoa(serviceAccount.ID))
	state.Group = types.StringValue(state.Group.ValueString())
	state.Name = types.StringValue(serviceAccount.Name)
	state.Username = types.StringValue(serviceAccount.UserName)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
