package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabMetadataDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabMetadataDataSource{}
)

func init() {
	registerDataSource(NewGitLabMetadataDataSource)
}

// NewGitLabMetadataDataSource is a helper function to simplify the provider implementation.
func NewGitLabMetadataDataSource() datasource.DataSource {
	return &gitlabMetadataDataSource{}
}

// gitlabMetadataDataSource is the data source implementation.
type gitlabMetadataDataSource struct {
	client *gitlab.Client
}

// gitLabMetadataDataSourceModel describes the data source data model.
type gitLabMetadataDataSourceModel struct {
	Id       string `tfsdk:"id"`
	Version  string `tfsdk:"version" json:"version"`
	Revision string `tfsdk:"revision" json:"revision"`
	KAS      struct {
		Enabled             bool   `tfsdk:"enabled" json:"enabled"`
		ExternalUrl         string `tfsdk:"external_url" json:"externalUrl"`
		ExternalK8SProxyUrl string `tfsdk:"external_k8s_proxy_url" json:"externalK8sProxyUrl"`
		Version             string `tfsdk:"version" json:"version"`
	} `tfsdk:"kas" json:"kas"`
	Enterprise bool `tfsdk:"enterprise" json:"enterprise"`
}

// Metadata returns the data source type name.
func (d *gitlabMetadataDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metadata"
}

// GetSchema defines the schema for the data source.
func (d *gitlabMetadataDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_metadata`" + ` data source retrieves the metadata of the GitLab instance.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/metadata/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The id of the data source. It will always be `1`",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Version of the GitLab instance.",
				Computed:            true,
			},
			"revision": schema.StringAttribute{
				MarkdownDescription: "Revision of the GitLab instance.",
				Computed:            true,
			},
			"kas": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata about the GitLab agent server for Kubernetes (KAS).",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Indicates whether KAS is enabled.",
						Computed:            true,
					},
					"external_url": schema.StringAttribute{
						MarkdownDescription: "URL used by the agents to communicate with KAS. It’s null if kas.enabled is false.",
						Computed:            true,
					},
					"external_k8s_proxy_url": schema.StringAttribute{
						MarkdownDescription: "URL used by the Kubernetes tooling to communicate with the KAS Kubernetes API proxy. It’s null if kas.enabled is false.",
						Computed:            true,
					},
					"version": schema.StringAttribute{
						MarkdownDescription: "Version of KAS. It’s null if kas.enabled is false.",
						Computed:            true,
					},
				},
			},
			"enterprise": schema.BoolAttribute{
				MarkdownDescription: "If the GitLab instance is an enterprise instance or not.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabMetadataDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabMetadataDataSourceModel

	// Map response model to state model
	state.Id = "1"

	// Make API call to read metadata
	tflog.Trace(ctx, "reading GitLab Metadata from API")
	err := func() error {
		req, err := d.client.NewRequest(http.MethodGet, "metadata", nil, []gitlab.RequestOptionFunc{gitlab.WithContext(ctx)})
		if err != nil {
			return err
		}

		_, err = d.client.Do(req, &state)
		return err
	}()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to fetch GitLab Metadata from API",
			err.Error(),
		)
		return
	}

	// Set state
	tflog.Trace(ctx, "setting GitLab Metadata from API into state", map[string]any{
		"id": state.Id,
	})
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
