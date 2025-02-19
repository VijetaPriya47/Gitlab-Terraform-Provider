package sdk

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var _ = registerResource("gitlab_integration_github", func() *schema.Resource {
	return resourceGitlabIntegrationGithubResource(`The ` + "`gitlab_integration_github`" + ` resource allows to manage the lifecycle of a project integration with GitHub.

-> This resource requires a GitLab Enterprise instance.
	
**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/integrations/#github)`)
})

var _ = registerResource("gitlab_service_github", func() *schema.Resource {
	resource := resourceGitlabIntegrationGithubResource(`The ` + "`gitlab_service_github`" + ` resource allows to manage the lifecycle of a project integration with GitHub.

-> This resource requires a GitLab Enterprise instance.

~> This resource is deprecated. use ` + "`gitlab_integration_github`" + `instead!
	
**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/integrations/#github)`,
	)
	resource.DeprecationMessage = `This resource is deprecated. use ` + "`gitlab_integration_github`" + `instead!`
	return resource
})

func resourceGitlabIntegrationGithubResource(description string) *schema.Resource {
	return &schema.Resource{
		Description: description,

		CreateContext: resourceGitlabIntegrationGithubCreate,
		ReadContext:   resourceGitlabIntegrationGithubRead,
		UpdateContext: resourceGitlabIntegrationGithubUpdate,
		DeleteContext: resourceGitlabIntegrationGithubDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGitlabIntegrationGithubImportState,
		},

		Schema: map[string]*schema.Schema{
			"project": {
				Description: "ID of the project you want to activate integration on.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"token": {
				Description: "A GitHub personal access token with at least `repo:status` scope.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"repository_url": {
				Description: "The URL of the GitHub repo to integrate with, e,g, https://github.com/gitlabhq/terraform-provider-gitlab.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"static_context": {
				Description: "Append instance name instead of branch to the status. Must enable to set a GitLab status check as _required_ in GitHub. See [Static / dynamic status check names] to learn more.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},

			// Computed from the GitLab API. Omitted event fields because they're always true in Github.
			"title": {
				Description: "Title.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"created_at": {
				Description: "Create time.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"updated_at": {
				Description: "Update time.",
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

func resourceGitlabIntegrationGithubSetToState(d *schema.ResourceData, service *gitlab.GithubService) {
	d.SetId(fmt.Sprintf("%d", service.ID))
	d.Set("repository_url", service.Properties.RepositoryURL)
	d.Set("static_context", service.Properties.StaticContext)

	d.Set("title", service.Title)
	d.Set("created_at", service.CreatedAt.String())
	d.Set("updated_at", service.UpdatedAt.String())
	d.Set("active", service.Active)
}

func resourceGitlabIntegrationGithubCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab github service for project %s", project))

	opts := &gitlab.SetGithubServiceOptions{
		Token:         gitlab.Ptr(d.Get("token").(string)),
		RepositoryURL: gitlab.Ptr(d.Get("repository_url").(string)),
		StaticContext: gitlab.Ptr(d.Get("static_context").(bool)),
	}

	_, _, err := client.Services.SetGithubService(project, opts, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabIntegrationGithubRead(ctx, d, meta)
}

func resourceGitlabIntegrationGithubRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab github service for project %s", project))

	service, _, err := client.Services.GetGithubService(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab service github not found %s / %s / %s",
				project,
				service.Title,
				service.Properties.RepositoryURL))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	resourceGitlabIntegrationGithubSetToState(d, service)

	return nil
}

func resourceGitlabIntegrationGithubUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceGitlabIntegrationGithubCreate(ctx, d, meta)
}

func resourceGitlabIntegrationGithubDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] delete gitlab github service for project %s", project))

	_, err := client.Services.DeleteGithubService(project, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitlabIntegrationGithubImportState(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("project", d.Id())

	return []*schema.ResourceData{d}, nil
}
