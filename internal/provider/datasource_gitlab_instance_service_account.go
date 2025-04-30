package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabInstanceServiceAccountDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabInstanceServiceAccountDataSource{}
)

func init() {
	registerDataSource(NewGitlabInstanceServiceAccountDataSource)
}

// NewGitlabInstanceServiceAccountDataSource is a helper function to simplify the provider implementation.
func NewGitlabInstanceServiceAccountDataSource() datasource.DataSource {
	return &gitlabInstanceServiceAccountDataSource{}
}

// gitlabInstanceServiceAccountDataSource is the data source implementation.
type gitlabInstanceServiceAccountDataSource struct {
	client *gitlab.Client
}

// gitlabGroupServiceAccountDataSourceModel describes the data source data model.
type gitlabInstanceServiceAccountDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
	Name             types.String `tfsdk:"name"`
	Username         types.String `tfsdk:"username"`
	Email            types.String `tfsdk:"email"`
}

// Metadata returns the data source type name.
func (d *gitlabInstanceServiceAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_service_account"
}

// Schema defines the schema for the data source.
func (d *gitlabInstanceServiceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_instance_service_account`" + ` data source retrieves information about a gitlab service account.

~> In order for a user to create a user account, they must have admin privileges at the instance level. This makes this feature unavailable on ` + "`gitlab.com`" + `

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/user_service_accounts/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. This matches the service account id.",
				Computed:            true,
			},
			"service_account_id": schema.StringAttribute{
				MarkdownDescription: "The service account id.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user.",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the user.",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email of the user.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabInstanceServiceAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabInstanceServiceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitlabInstanceServiceAccountDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccountID, err := strconv.Atoi(state.ServiceAccountID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to convert resource ID: %s", err.Error()))
		return
	}

	serviceAccount, _, err := d.client.Users.GetUser(serviceAccountID, gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read service account: %s", err.Error()))
		return
	}

	// Set the ID
	serviceAccountIDStr := strconv.Itoa(serviceAccount.ID)
	state.ID = types.StringValue(serviceAccountIDStr)
	state.ServiceAccountID = types.StringValue(serviceAccountIDStr)
	state.Name = types.StringValue(serviceAccount.Name)
	state.Username = types.StringValue(serviceAccount.Username)
	state.Email = types.StringValue(serviceAccount.Email)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
