package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var _ = registerResource("gitlab_integration_external_wiki", func() *schema.Resource {
	return resourceGitlabIntegrationEmailsOnPushResource(`The ` + "`gitlab_integration_external_wiki`" + ` resource allows to manage the lifecycle of a project integration with External Wiki Service.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/integrations.html#external-wiki)`,
	)
})

var _ = registerResource("gitlab_service_external_wiki", func() *schema.Resource {
	resource := resourceGitlabIntegrationEmailsOnPushResource(`The ` + "`gitlab_service_external_wiki`" + ` resource allows to manage the lifecycle of a project integration with External Wiki Service.

~> This resource is deprecated. use ` + "`gitlab_integration_external_wiki`" + `instead!

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/integrations.html#external-wiki)`,
	)
	resource.DeprecationMessage = `This resource is deprecated. use ` + "`gitlab_integration_external_wiki`" + `instead!`
	return resource
})

func resourceGitlabIntegrationEmailsOnPushResource(description string) *schema.Resource {
	return &schema.Resource{
		Description: description,

		CreateContext: resourceGitlabIntegrationExternalWikiCreate,
		ReadContext:   resourceGitlabIntegrationExternalWikiRead,
		UpdateContext: resourceGitlabIntegrationExternalWikiCreate,
		DeleteContext: resourceGitlabIntegrationExternalWikiDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"project": {
				Description:  "ID of the project you want to activate integration on.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"external_wiki_url": {
				Description:  "The URL of the external wiki.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"title": {
				Description: "Title of the integration.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"created_at": {
				Description: "The ISO8601 date/time that this integration was activated at in UTC.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"updated_at": {
				Description: "The ISO8601 date/time that this integration was last updated at in UTC.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"slug": {
				Description: "The name of the integration in lowercase, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"active": {
				Description: "Whether the integration is active.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
		},
	}
}

func resourceGitlabIntegrationExternalWikiCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	d.SetId(project)

	options := &gitlab.SetExternalWikiServiceOptions{
		ExternalWikiURL: gitlab.Ptr(d.Get("external_wiki_url").(string)),
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab external wiki service for project %s", project))

	_, _, err := client.Services.SetExternalWikiService(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabIntegrationExternalWikiRead(ctx, d, meta)
}

func resourceGitlabIntegrationExternalWikiRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab external wiki service for project %s", project))

	service, _, err := client.Services.GetExternalWikiService(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab external wiki service not found for project %s", project))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("project", project)
	d.Set("external_wiki_url", service.Properties.ExternalWikiURL)
	d.Set("active", service.Active)
	d.Set("slug", service.Slug)
	d.Set("title", service.Title)
	d.Set("created_at", service.CreatedAt.Format(time.RFC3339))
	if service.UpdatedAt != nil {
		d.Set("updated_at", service.UpdatedAt.Format(time.RFC3339))
	}

	return nil
}

func resourceGitlabIntegrationExternalWikiDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] delete gitlab external wiki service for project %s", project))

	_, err := client.Services.DeleteExternalWikiService(project, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
