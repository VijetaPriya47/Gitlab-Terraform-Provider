package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure GitLabProvider satisfies various provider interfaces.
var _ provider.Provider = &GitLabProvider{}

// GitLabProvider defines the provider implementation.
type GitLabProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance testing.
	version string
}

// GitLabProviderModel describes the provider data model.
type GitLabProviderModel struct {
	Token          types.String `tfsdk:"token"`
	BaseUrl        types.String `tfsdk:"base_url"`
	CACertFile     types.String `tfsdk:"cacert_file"`
	Insecure       types.Bool   `tfsdk:"insecure"`
	ClientCert     types.String `tfsdk:"client_cert"`
	ClientKey      types.String `tfsdk:"client_key"`
	EarlyAuthCheck types.Bool   `tfsdk:"early_auth_check"`
	Retries        types.Int64  `tfsdk:"retries"`
	Headers        types.Map    `tfsdk:"headers"`
}

func (p *GitLabProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "gitlab"
	resp.Version = p.version
}

func (p *GitLabProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "The OAuth2 Token, Project, Group, Personal Access Token or CI Job Token used to connect to GitLab. The OAuth method is used in this provider for authentication (using Bearer authorization token). See https://docs.gitlab.com/api/#authentication for details. It may be sourced from the `GITLAB_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "This is the target GitLab base API endpoint. Providing a value is a requirement when working with GitLab CE or GitLab Enterprise e.g. `https://my.gitlab.server/api/v4/`. It is optional to provide this value and it can also be sourced from the `GITLAB_BASE_URL` environment variable. The value must end with a slash.",
				Optional:            true,
			},
			"cacert_file": schema.StringAttribute{
				MarkdownDescription: "This is a file containing the ca cert to verify the gitlab instance. This is available for use when working with GitLab CE or Gitlab Enterprise with a locally-issued or self-signed certificate chain.",
				Optional:            true,
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "When set to true this disables SSL verification of the connection to the GitLab instance.",
				Optional:            true,
			},
			"client_cert": schema.StringAttribute{
				MarkdownDescription: "File path to client certificate when GitLab instance is behind company proxy. File must contain PEM encoded data.",
				Optional:            true,
			},
			"client_key": schema.StringAttribute{
				MarkdownDescription: "File path to client key when GitLab instance is behind company proxy. File must contain PEM encoded data. Required when `client_cert` is set.",
				Optional:            true,
			},
			"early_auth_check": schema.BoolAttribute{
				MarkdownDescription: "(Experimental) By default the provider does a dummy request to get the current user in order to verify that the provider configuration is correct and the GitLab API is reachable. Set this to `false` to skip this check. This may be useful if the GitLab instance does not yet exist and is created within the same terraform module. It may be sourced from the `GITLAB_EARLY_AUTH_CHECK`. This is an experimental feature and may change in the future. Please make sure to always keep backups of your state.",
				Optional:            true,
			},
			"retries": schema.Int64Attribute{
				MarkdownDescription: "The number of retries to execute when receiving a 429 Rate Limit error. Each retry will exponentially back off.",
				Optional:            true,
			},
			"headers": schema.MapAttribute{
				MarkdownDescription: "A map of headers to append to all API request to the GitLab instance.",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (p *GitLabProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Parse the provider configuration
	var config GitLabProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Prevent an unexpectedly misconfigured client, if Terraform configuration values are only known after another resource is applied.
	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown GitLab Token",
			"The provider cannot create the GitLab API client as there is an unknown configuration value for the GitLab Token. "+
				"Either apply the source of the value first, set the token attribute value statically in the configuration, or use the GITLAB_TOKEN environment variable.",
		)
	}
	if config.BaseUrl.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Unknown GitLab Base URL for the API endpoint",
			"The provider cannot create the GitLab API client as there is an unknown configuration value for the GitLab Base URL. "+
				"Either apply the source of the value first, set the token attribute value statically in the configuration, or use the GITLAB_BASE_URL environment variable.",
		)
	}
	if config.CACertFile.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("cacert_file"),
			"Unknown GitLab CA Certificate File",
			"The provider cannot create the GitLab API client as there is an unknown configuration value for the GitLab CA Certificate File. "+
				"Either apply the source of the value first, set the token attribute value statically in the configuration.",
		)
	}
	if config.Insecure.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("insecure"),
			"Unknown GitLab Insecure Flag Value",
			"The provider cannot create the GitLab API client as there is an unknown configuration value for the GitLab Insecure flag. "+
				"Either apply the source of the value first, set the token attribute value statically in the configuration.",
		)
	}
	if config.ClientCert.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_cert"),
			"Unknown GitLab Client Certificate",
			"The provider cannot create the GitLab API client as there is an unknown configuration value for the GitLab Client Certificate. "+
				"Either apply the source of the value first, set the token attribute value statically in the configuration.",
		)
	}
	if config.ClientKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_key"),
			"Unknown GitLab Client Key",
			"The provider cannot create the GitLab API client as there is an unknown configuration value for the GitLab Client Key. "+
				"Either apply the source of the value first, set the token attribute value statically in the configuration.",
		)
	}
	if config.EarlyAuthCheck.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("early_auth_check"),
			"Unknown GitLab Early Auth Check Flag Value",
			"The provider cannot create the GitLab API client as there is an unknown configuration value for the GitLab Early Auth Check flag. "+
				"Either apply the source of the value first, set the token attribute value statically in the configuration, or use the GITLAB_EARLY_AUTH_CHECK environment variable.",
		)
	}
	if config.Headers.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("headers"),
			"Unknown Headers for the GitLab API calls",
			"The provider cannot create the GitLab API client as there is an unknown configuration value for the GitLab Headers flag. "+
				"Either apply the source of the value first or set the headers attribute value statically in the configuration.",
		)
	}

	// If no value is explicitly provided for Retries, default to 10
	if config.Retries.IsUnknown() || config.Retries.IsNull() {
		config.Retries = types.Int64Value(10)
	}

	earlyAuthCheck, err := utils.ParseConfigBoolFromEnv("GITLAB_EARLY_AUTH_CHECK", true)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("early_auth_check"),
			"Invalid GitLab Early Auth Check Flag Value from Env Variable",
			err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Provider Configuration containing the values after evaluation of defaults etc.
	// Initialized with the defaults which get overridden later if config is set.
	evaluatedConfig := api.Config{
		Token:         os.Getenv("GITLAB_TOKEN"),
		BaseURL:       os.Getenv("GITLAB_BASE_URL"),
		CACertFile:    "",
		Insecure:      false,
		ClientCert:    "",
		ClientKey:     "",
		EarlyAuthFail: earlyAuthCheck,
		Headers:       nil,
	}

	// Evaluate Provider Attribute Default values now that they are all "known"
	if !config.Token.IsNull() {
		evaluatedConfig.Token = config.Token.ValueString()
	}
	if !config.BaseUrl.IsNull() {
		evaluatedConfig.BaseURL = config.BaseUrl.ValueString()
	}
	if !config.CACertFile.IsNull() {
		evaluatedConfig.CACertFile = config.CACertFile.ValueString()
	}
	if !config.Insecure.IsNull() {
		evaluatedConfig.Insecure = config.Insecure.ValueBool()
	}
	if !config.ClientCert.IsNull() {
		evaluatedConfig.ClientCert = config.ClientCert.ValueString()
	}
	if !config.ClientKey.IsNull() {
		evaluatedConfig.ClientKey = config.ClientKey.ValueString()
	}
	if !config.EarlyAuthCheck.IsNull() {
		evaluatedConfig.EarlyAuthFail = config.EarlyAuthCheck.ValueBool()
	}
	if !config.Retries.IsNull() {
		evaluatedConfig.Retries = int(config.Retries.ValueInt64())
	}
	if !config.Headers.IsNull() {
		// Convert the terraform types.Map (map[str]attr.Value) into one that have types.String as the key
		stringMap := make(map[string]types.String, len(config.Headers.Elements()))
		resp.Diagnostics.Append(config.Headers.ElementsAs(ctx, &stringMap, false)...)

		// Further reduce that to ma[string]string
		headersMap := map[string]any{}
		for key, val := range stringMap {
			headersMap[key] = val.ValueString()
		}

		evaluatedConfig.Headers = headersMap
	}

	// TODO(@timofurrer): validate configuration values

	// Configure our logger masking
	ctx = utils.ApplyLogMaskingToContext(ctx)

	// Creating a new GitLab Client from the provider configuration
	clientFactory := newGitLabClient(evaluatedConfig, req.TerraformVersion, p.version)
	gitlabClient, err := clientFactory(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create GitLab Client from provider configuration", err.Error())
		return
	}

	// Attach the client to the response so that it will be available for the Data Sources and Resources
	resp.DataSourceData = &GitLabDatasourceData{
		Client: gitlabClient,
	}
	resp.ResourceData = &GitLabResourceData{
		Client:          gitlabClient,
		NewGitLabClient: clientFactory,
	}
}

func (p *GitLabProvider) Resources(ctx context.Context) []func() resource.Resource {
	return allResources
}

func (p *GitLabProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return allDataSources
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GitLabProvider{
			version: version,
		}
	}
}

type (
	GitLabClientOptionApplyFunc = func(api.Config) api.Config
	GitLabClientFactory         = func(ctx context.Context, configFuncs ...GitLabClientOptionApplyFunc) (*gitlab.Client, error)
)

// Attributes passed into Datasources from the Provider
type GitLabDatasourceData struct {
	Client *gitlab.Client
}

// Attributes passed into Resources from the Provider
type GitLabResourceData struct {
	Client          *gitlab.Client
	NewGitLabClient GitLabClientFactory
}

func newGitLabClient(config api.Config, tfVersion, providerVersion string) GitLabClientFactory {
	return func(ctx context.Context, configFuncs ...GitLabClientOptionApplyFunc) (*gitlab.Client, error) {
		for _, f := range configFuncs {
			config = f(config)
		}

		client, err := config.NewGitLabClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("The provider failed to create a new GitLab Client from the given configuration: %w", err)
		}

		// NOTE: there is no helper function for this available yet in the terraform-plugin-framework,
		//       see https://github.com/hashicorp/terraform-plugin-framework/issues/280
		client.UserAgent = fmt.Sprintf("Terraform/%s (+https://www.terraform.io) Terraform-Plugin-Framework terraform-provider-gitlab/%s", tfVersion, providerVersion)
		return client, nil
	}
}

func WithToken(token string) GitLabClientOptionApplyFunc {
	return func(config api.Config) api.Config {
		config.Token = token
		return config
	}
}

func WithEarlyAuth(earlyAuth bool) GitLabClientOptionApplyFunc {
	return func(config api.Config) api.Config {
		config.EarlyAuthFail = earlyAuth
		return config
	}
}
