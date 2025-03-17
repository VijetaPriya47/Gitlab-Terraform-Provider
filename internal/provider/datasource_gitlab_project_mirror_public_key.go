package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitLabProjectMirrorPublicKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &gitLabProjectMirrorPublicKeyDataSource{}
)

func init() {
	registerDataSource(NewGitLabProjectMirrorPublicKeyDataSource)
}

// NewGitLabProjectMirrorPublicKeyDataSource is a helper function to simplify the provider implementation.
func NewGitLabProjectMirrorPublicKeyDataSource() datasource.DataSource {
	return &gitLabProjectMirrorPublicKeyDataSource{}
}

// gitLabProjectMirrorPublicKeyDataSource is the data source implementation.
type gitLabProjectMirrorPublicKeyDataSource struct {
	client *gitlab.Client
}

// gitLabProjectMirrorPublicKeyDataSourceModel describes the data source data model.
type gitLabProjectMirrorPublicKeyDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	MirrorId  types.Int64  `tfsdk:"mirror_id"`
	PublicKey types.String `tfsdk:"public_key"`
}

// Metadata returns the data source type name.
func (d *gitLabProjectMirrorPublicKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_mirror_public_key"
}

// Schema defines the schema for the data source.
func (d *gitLabProjectMirrorPublicKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_mirror_public_key`" + ` data source allows the public key of a project mirror to be retrieved by its mirror id and the project it belongs to.

**Note**: Supported on GitLab 17.9 or higher.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/remote_mirrors/#get-a-single-projects-remote-mirror-public-key)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project_id>:<mirror_id>`.",
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The integer or path with namespace that uniquely identifies the project.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"mirror_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the remote mirror.",
				Required:            true,
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "Public key of the remote mirror.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitLabProjectMirrorPublicKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitLabProjectMirrorPublicKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabProjectMirrorPublicKeyDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// call get project mirror public key API
	mirrorPublicKey, _, err := d.client.ProjectMirrors.GetProjectMirrorPublicKey(state.ProjectId.ValueString(), int(state.MirrorId.ValueInt64()), gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read project mirror public key: %s", err.Error()))
		return
	}

	state.Id = types.StringValue(utils.BuildTwoPartID(state.ProjectId.ValueStringPointer(), gitlab.Ptr(strconv.Itoa(int(state.MirrorId.ValueInt64())))))
	state.PublicKey = types.StringValue(mirrorPublicKey.PublicKey)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
