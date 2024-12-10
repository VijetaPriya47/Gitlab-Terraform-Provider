package sdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var _ = registerResource("gitlab_integration_pipelines_email", func() *schema.Resource {
	return resourceGitlabIntegrationPipelinesEmailSchema(`The ` + "`gitlab_integration_pipelines_email`" + ` resource allows to manage the lifecycle of a project integration with Pipeline Emails Service.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/integrations.html#pipeline-emails)`)
})

var _ = registerResource("gitlab_service_pipelines_email", func() *schema.Resource {
	schema := resourceGitlabIntegrationPipelinesEmailSchema(`The ` + "`gitlab_service_pipelines_email`" + ` resource allows to manage the lifecycle of a project integration with Pipeline Emails Service.

~> This resource is deprecated. use ` + "`gitlab_integration_pipelines_email`" + `instead!

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/integrations.html#pipeline-emails)`)
	schema.DeprecationMessage = `This resource is deprecated. use ` + "`gitlab_integration_pipelines_email`" + `instead!`
	return schema
})

func resourceGitlabIntegrationPipelinesEmailSchema(description string) *schema.Resource {
	return &schema.Resource{
		Description: description,

		CreateContext: resourceGitlabIntegrationPipelinesEmailCreate,
		ReadContext:   resourceGitlabIntegrationPipelinesEmailRead,
		UpdateContext: resourceGitlabIntegrationPipelinesEmailCreate,
		DeleteContext: resourceGitlabIntegrationPipelinesEmailDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"project": {
				Description: "ID of the project you want to activate integration on.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"recipients": {
				Description: ") email addresses where notifications are sent.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"notify_only_broken_pipelines": {
				Description: "Notify only broken pipelines. Default is true.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"branches_to_be_notified": {
				Description:  "Branches to send notifications for. Valid options are `all`, `default`, `protected`, and `default_and_protected`. Default is `default`",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"all", "default", "protected", "default_and_protected"}, true),
				Default:      "default",
			},
		},
	}
}

func resourceGitlabIntegrationPipelinesEmailSetToState(d *schema.ResourceData, service *gitlab.PipelinesEmailService) {
	d.Set("recipients", strings.Split(service.Properties.Recipients, ",")) // lintignore: XR004 // TODO: Resolve this tfproviderlint issue
	d.Set("notify_only_broken_pipelines", service.Properties.NotifyOnlyBrokenPipelines)
	d.Set("branches_to_be_notified", service.Properties.BranchesToBeNotified)
}

func resourceGitlabIntegrationPipelinesEmailCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	d.SetId(project)
	options := &gitlab.SetPipelinesEmailServiceOptions{
		Recipients:                gitlab.Ptr(strings.Join(*stringSetToStringSlice(d.Get("recipients").(*schema.Set)), ",")),
		NotifyOnlyBrokenPipelines: gitlab.Ptr(d.Get("notify_only_broken_pipelines").(bool)),
		BranchesToBeNotified:      gitlab.Ptr(d.Get("branches_to_be_notified").(string)),
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab pipelines emails integration for project %s", project))

	_, _, err := client.Services.SetPipelinesEmailService(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabIntegrationPipelinesEmailRead(ctx, d, meta)
}

func resourceGitlabIntegrationPipelinesEmailRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab pipelines emails integration for project %s", project))

	service, _, err := client.Services.GetPipelinesEmailService(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab pipelines emails integration not found for project %s", project))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("project", project)
	resourceGitlabIntegrationPipelinesEmailSetToState(d, service)
	return nil
}

func resourceGitlabIntegrationPipelinesEmailDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] delete gitlab pipelines email integration for project %s", project))

	_, err := client.Services.DeletePipelinesEmailService(project, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
