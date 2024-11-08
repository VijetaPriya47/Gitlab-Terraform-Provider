package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/xanzy/go-gitlab"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabApplicationDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabApplicationDataSource{}
)

func init() {
	registerDataSource(NewGitLabApplicationDataSource)
}

// NewGitLabApplicationDataSource is a helper function to simplify the provider implementation.
func NewGitLabApplicationDataSource() datasource.DataSource {
	return &gitlabApplicationDataSource{}
}

// gitlabMetadataDataSource is the data source implementation.
type gitlabApplicationDataSource struct {
	client *gitlab.Client
}

// gitLabMetadataDataSourceModel describes the data source data model.
type gitLabApplicationDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	ApplicationId types.String `tfsdk:"application_id"`
	Name          types.String `tfsdk:"name"`
	RedirectURL   types.String `tfsdk:"redirect_url"`
	Confidential  types.Bool   `tfsdk:"confidential"`
}

// Metadata returns the data source type name.
func (d *gitlabApplicationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

// GetSchema defines the schema for the data source.
func (d *gitlabApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_application`" + ` data source retrieves information about a gitlab application.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/applications.html)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<application_id>`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "Internal GitLab application id.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the GitLab application.",
				Computed:            true,
			},
			"redirect_url": schema.StringAttribute{
				MarkdownDescription: "The redirect url of the application.",
				Computed:            true,
			},
			"confidential": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the application is kept confidential.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabApplicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabApplicationDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}
	// Make API call to read applications
	application, err := findGitlabApplication(d.client, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read application details: %s", err.Error()))
		return
	}

	state.ApplicationId = types.StringValue(application.ApplicationID)
	state.Confidential = types.BoolValue(application.Confidential)
	state.Name = types.StringValue(application.ApplicationName)
	state.RedirectURL = types.StringValue(application.CallbackURL)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
