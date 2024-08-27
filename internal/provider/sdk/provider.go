package sdk

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	allDataSources = make(map[string]func() *schema.Resource)
	allResources   = make(map[string]func() *schema.Resource)
)

// registerDataSource may be called during package initialization to register a new data source with
// the provider.
var registerDataSource = makeRegisterResourceFunc(allDataSources, "data source")

// registerResource may be called during package initialization to register a new resource with the
// provider.
var registerResource = makeRegisterResourceFunc(allResources, "resource")

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown
}

func NewV6(ctx context.Context, version string) (tfprotov6.ProviderServer, error) {
	upgradedSdkProvider, err := tf5to6server.UpgradeServer(ctx, New(version)().GRPCProvider)
	if err != nil {
		return nil, err
	}
	return upgradedSdkProvider, nil
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		provider := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"token": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The OAuth2 Token, Project, Group, Personal Access Token or CI Job Token used to connect to GitLab. The OAuth method is used in this provider for authentication (using Bearer authorization token). See https://docs.gitlab.com/ee/api/#authentication for details. It may be sourced from the `GITLAB_TOKEN` environment variable.",
					Sensitive:   true,
				},
				"base_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "This is the target GitLab base API endpoint. Providing a value is a requirement when working with GitLab CE or GitLab Enterprise e.g. `https://my.gitlab.server/api/v4/`. It is optional to provide this value and it can also be sourced from the `GITLAB_BASE_URL` environment variable. The value must end with a slash.",
				},
				"cacert_file": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "This is a file containing the ca cert to verify the gitlab instance. This is available for use when working with GitLab CE or Gitlab Enterprise with a locally-issued or self-signed certificate chain.",
				},
				"insecure": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "When set to true this disables SSL verification of the connection to the GitLab instance.",
				},
				"client_cert": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "File path to client certificate when GitLab instance is behind company proxy. File must contain PEM encoded data.",
				},
				"client_key": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "File path to client key when GitLab instance is behind company proxy. File must contain PEM encoded data. Required when `client_cert` is set.",
				},
				"early_auth_check": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "(Experimental) By default the provider does a dummy request to get the current user in order to verify that the provider configuration is correct and the GitLab API is reachable. Set this to `false` to skip this check. This may be useful if the GitLab instance does not yet exist and is created within the same terraform module. It may be sourced from the `GITLAB_EARLY_AUTH_CHECK`. This is an experimental feature and may change in the future. Please make sure to always keep backups of your state.",
				},
				"retries": {
					Type:        schema.TypeInt,
					Optional:    true,
					Default:     10,
					Description: "The number of retries to execute when receiving a 429 Rate Limit error. Each retry will exponentially back off.",
				},
			},

			DataSourcesMap: resourceFactoriesToMap(allDataSources),
			ResourcesMap:   resourceFactoriesToMap(allResources),
		}

		provider.ConfigureContextFunc = configure(version, provider)
		return provider
	}

}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		config := api.Config{
			Token:         d.Get("token").(string),
			BaseURL:       d.Get("base_url").(string),
			CACertFile:    d.Get("cacert_file").(string),
			Insecure:      d.Get("insecure").(bool),
			ClientCert:    d.Get("client_cert").(string),
			ClientKey:     d.Get("client_key").(string),
			EarlyAuthFail: d.Get("early_auth_check").(bool),
			Retries:       d.Get("retries").(int),
		}
		if _, ok := d.GetOk("token"); !ok {
			config.Token = os.Getenv("GITLAB_TOKEN")
		}
		if _, ok := d.GetOk("base_url"); !ok {
			config.BaseURL = os.Getenv("GITLAB_BASE_URL")
		}
		// It is the only way to differentiate between unset boolean attributes and attributes set to false
		//nolint:staticcheck, tfproviderlint
		if _, ok := d.GetOkExists("early_auth_check"); !ok {
			earlyAuthCheck, err := utils.ParseConfigBoolFromEnv("GITLAB_EARLY_AUTH_CHECK", true)
			if err != nil {
				return nil, diag.FromErr(err)
			}
			config.EarlyAuthFail = earlyAuthCheck
		}

		// Configure our logger masking
		ctx = utils.ApplyLogMaskingToContext(ctx)

		gitlabClient, err := config.NewGitLabClient(ctx)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		userAgent := p.UserAgent("terraform-provider-gitlab", version)
		gitlabClient.UserAgent = userAgent

		return gitlabClient, nil
	}
}

func makeRegisterResourceFunc(factories map[string]func() *schema.Resource, resourceType string) func(name string, fn func() *schema.Resource) interface{} {
	// lintignore: R009 // panic() during package initialization is ok
	return func(name string, fn func() *schema.Resource) interface{} {
		if strings.ToLower(name) != name {
			panic(fmt.Sprintf("cannot register %s %q: name must be lowercase", resourceType, name))
		}

		const wantPrefix = "gitlab_"
		if !strings.HasPrefix(name, wantPrefix) {
			panic(fmt.Sprintf("cannot register %s %q: name must begin with %q", resourceType, name, wantPrefix))
		}

		if _, exists := factories[name]; exists {
			panic(fmt.Sprintf("cannot register %s %q: a %s with the same name already exists", resourceType, name, resourceType))
		}

		factories[name] = fn

		return nil
	}
}

func resourceFactoriesToMap(factories map[string]func() *schema.Resource) map[string]*schema.Resource {
	resourcesMap := make(map[string]*schema.Resource)

	for name, fn := range factories {
		resourcesMap[name] = fn()
	}

	return resourcesMap
}
