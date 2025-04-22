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

var _ = registerResource("gitlab_integration_microsoft_teams", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_integration_microsoft_teams`" + ` resource allows you to manage the lifecycle of a project integration with Microsoft Teams.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#microsoft-teams-notifications)`,

		CreateContext: resourceGitlabIntegrationMicrosoftTeamsCreate,
		ReadContext:   resourceGitlabIntegrationMicrosoftTeamsRead,
		UpdateContext: resourceGitlabIntegrationMicrosoftTeamsUpdate,
		DeleteContext: resourceGitlabIntegrationMicrosoftTeamsDelete,
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
			"webhook": {
				Description:  "The Microsoft Teams webhook (Example, https://outlook.office.com/webhook/...). This value cannot be imported.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateURLFunc,
			},
			"notify_only_broken_pipelines": {
				Description: "Send notifications for broken pipelines",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"branches_to_be_notified": {
				Description: "Branches to send notifications for. Valid options are “all”, “default”, “protected”, and “default_and_protected”. The default value is “default”",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"push_events": {
				Description: "Enable notifications for push events",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"issues_events": {
				Description: "Enable notifications for issue events",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"confidential_issues_events": {
				Description: "Enable notifications for confidential issue events",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"merge_requests_events": {
				Description: "Enable notifications for merge request events",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"tag_push_events": {
				Description: "Enable notifications for tag push events",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"note_events": {
				Description: "Enable notifications for note events",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"confidential_note_events": {
				Description: "Enable notifications for confidential note events",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"pipeline_events": {
				Description: "Enable notifications for pipeline events",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"wiki_page_events": {
				Description: "Enable notifications for wiki page events",
				Type:        schema.TypeBool,
				Optional:    true,
			},
		},
	}
})

func resourceGitlabIntegrationMicrosoftTeamsCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	d.SetId(project)

	options := &gitlab.SetMicrosoftTeamsServiceOptions{
		WebHook:                   gitlab.Ptr(d.Get("webhook").(string)),
		NotifyOnlyBrokenPipelines: gitlab.Ptr(d.Get("notify_only_broken_pipelines").(bool)),
		BranchesToBeNotified:      gitlab.Ptr(d.Get("branches_to_be_notified").(string)),
		PushEvents:                gitlab.Ptr(d.Get("push_events").(bool)),
		IssuesEvents:              gitlab.Ptr(d.Get("issues_events").(bool)),
		ConfidentialIssuesEvents:  gitlab.Ptr(d.Get("confidential_issues_events").(bool)),
		MergeRequestsEvents:       gitlab.Ptr(d.Get("merge_requests_events").(bool)),
		TagPushEvents:             gitlab.Ptr(d.Get("tag_push_events").(bool)),
		NoteEvents:                gitlab.Ptr(d.Get("note_events").(bool)),
		ConfidentialNoteEvents:    gitlab.Ptr(d.Get("confidential_note_events").(bool)),
		PipelineEvents:            gitlab.Ptr(d.Get("pipeline_events").(bool)),
		WikiPageEvents:            gitlab.Ptr(d.Get("wiki_page_events").(bool)),
	}

	tflog.Debug(ctx, "[DEBUG] Create Gitlab Microsoft Teams integration")

	if _, _, err := client.Services.SetMicrosoftTeamsService(project, options, gitlab.WithContext(ctx)); err != nil {
		return diag.Errorf("couldn't create Gitlab Microsoft Teams integration: %v", err)
	}

	return resourceGitlabIntegrationMicrosoftTeamsRead(ctx, d, meta)
}

func resourceGitlabIntegrationMicrosoftTeamsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Read Gitlab Microsoft Teams integration for project %s", d.Id()))

	teamsService, _, err := client.Services.GetMicrosoftTeamsService(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Unable to find Gitlab Microsoft Teams integration in project %s, removing from state", project))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// The webhook is explicitly not set anymore, due to being removed from the API. It will now
	// use whatever is in the configuration to determine the value.
	// See https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/1421 for more info.
	// d.Set("webhook", teamsService.Properties.WebHook)

	d.Set("project", project)
	d.Set("created_at", teamsService.CreatedAt.String())
	d.Set("updated_at", teamsService.UpdatedAt.String())
	d.Set("active", teamsService.Active)
	d.Set("notify_only_broken_pipelines", teamsService.Properties.NotifyOnlyBrokenPipelines)
	d.Set("branches_to_be_notified", teamsService.Properties.BranchesToBeNotified)
	d.Set("push_events", teamsService.PushEvents)
	d.Set("issues_events", teamsService.IssuesEvents)
	d.Set("confidential_issues_events", teamsService.ConfidentialIssuesEvents)
	d.Set("merge_requests_events", teamsService.MergeRequestsEvents)
	d.Set("tag_push_events", teamsService.TagPushEvents)
	d.Set("note_events", teamsService.NoteEvents)
	d.Set("confidential_note_events", teamsService.ConfidentialNoteEvents)
	d.Set("pipeline_events", teamsService.PipelineEvents)
	d.Set("wiki_page_events", teamsService.WikiPageEvents)

	return nil
}

func resourceGitlabIntegrationMicrosoftTeamsUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceGitlabIntegrationMicrosoftTeamsCreate(ctx, d, meta)
}

func resourceGitlabIntegrationMicrosoftTeamsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete Gitlab Microsoft Teams integration for project %s", d.Id()))

	_, err := client.Services.DeleteMicrosoftTeamsService(project, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(err)
}
