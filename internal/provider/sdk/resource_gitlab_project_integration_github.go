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
	return getProjectIntegrationGithubResourceSchema(`The ` + "`gitlab_integration_github`" + ` resource manages the lifecycle of a project integration with GitHub.

-> This resource requires a GitLab Enterprise instance.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`gitlab_project_integration_github`" + ` instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#github)`)
})

var _ = registerResource("gitlab_project_integration_github", func() *schema.Resource {
	return getProjectIntegrationGithubResourceSchema(`The ` + "`gitlab_project_integration_github`" + ` resource manages the lifecycle of a project integration with GitHub.

-> This resource requires a GitLab Enterprise instance.
	
**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#github)`)
})

func getProjectIntegrationGithubResourceSchema(description string) *schema.Resource {
	return &schema.Resource{
		Description: description,

		CreateContext: resourceGitlabProjectIntegrationGithubCreate,
		ReadContext:   resourceGitlabProjectIntegrationGithubRead,
		UpdateContext: resourceGitlabProjectIntegrationGithubUpdate,
		DeleteContext: resourceGitlabProjectIntegrationGithubDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGitlabProjectIntegrationGithubImportState,
		},

		Schema: map[string]*schema.Schema{
			"project": {
				Description: "ID of the project you want to activate the integration on.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"token": {
				Description: "A GitHub personal access token with at least the `repo:status` scope.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"repository_url": {
				Description: "The URL of the GitHub repo to integrate with. For example, https://github.com/gitlabhq/terraform-provider-gitlab.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"static_context": {
				Description: "Append the instance name instead of the branch to the status. Must enable to set a GitLab status check as _required_ in GitHub. See [Static / dynamic status check names] to learn more.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},

			// Computed from the GitLab API. Omitted event fields because they're always true in Github.
			"title": {
				Description: "The title of this resource.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"created_at": {
				Description: "Creation time.",
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

func resourceGitlabProjectIntegrationGithubSetToState(d *schema.ResourceData, service *gitlab.GithubService) {
	d.SetId(fmt.Sprintf("%d", service.ID))
	d.Set("repository_url", service.Properties.RepositoryURL)
	d.Set("static_context", service.Properties.StaticContext)

	d.Set("title", service.Title)
	d.Set("created_at", service.CreatedAt.String())
	d.Set("updated_at", service.UpdatedAt.String())
	d.Set("active", service.Active)
}

func resourceGitlabProjectIntegrationGithubCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)

	tflog.Debug(ctx, fmt.Sprintf("create gitlab github service for project %s", project))

	opts := &gitlab.SetGithubServiceOptions{
		Token:         gitlab.Ptr(d.Get("token").(string)),
		RepositoryURL: gitlab.Ptr(d.Get("repository_url").(string)),
		StaticContext: gitlab.Ptr(d.Get("static_context").(bool)),
	}

	_, _, err := client.Services.SetGithubService(project, opts, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabProjectIntegrationGithubRead(ctx, d, meta)
}

func resourceGitlabProjectIntegrationGithubRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)

	tflog.Debug(ctx, fmt.Sprintf("read gitlab github service for project %s", project))

	service, _, err := client.Services.GetGithubService(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("gitlab service github not found for project %s", project))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	resourceGitlabProjectIntegrationGithubSetToState(d, service)

	return nil
}

func resourceGitlabProjectIntegrationGithubUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return resourceGitlabProjectIntegrationGithubCreate(ctx, d, meta)
}

func resourceGitlabProjectIntegrationGithubDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)

	tflog.Debug(ctx, fmt.Sprintf("delete gitlab github service for project %s", project))

	_, err := client.Services.DeleteGithubService(project, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitlabProjectIntegrationGithubImportState(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	d.Set("project", d.Id())

	return []*schema.ResourceData{d}, nil
}
