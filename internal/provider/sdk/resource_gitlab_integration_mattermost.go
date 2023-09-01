package sdk

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var _ = registerResource("gitlab_integration_mattermost", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_integration_mattermost`" + ` resource allows to manage the lifecycle of a project integration with Mattermost.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/integrations.html#mattermost-notifications)`,

		CreateContext: resourceGitlabIntegrationMattermostCreate,
		ReadContext:   resourceGitlabIntegrationMattermostRead,
		UpdateContext: resourceGitlabIntegrationMattermostUpdate,
		DeleteContext: resourceGitlabIntegrationMattermostDelete,
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
			"webhook": {
				Description: "Webhook URL (Example, https://mattermost.yourdomain.com/hooks/...). This value cannot be imported.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"username": {
				Description: "Username to use.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			// Channel is not yet implemented by GitLab, contrary to what the documentation states
			// See https://gitlab.com/gitlab-org/gitlab/-/blob/902aaf4b412dc61a165b588611885cd60afb7a69/app/models/integrations/base_chat_notification.rb#L170
			//"channel": {
			//	Description: "Default channel to use if others are not configured.",
			//	Type:        schema.TypeString,
			//	Optional:    true,
			//},
			"notify_only_broken_pipelines": {
				Description: "Send notifications for broken pipelines.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"branches_to_be_notified": {
				Description: "Branches to send notifications for. Valid options are \"all\", \"default\", \"protected\", and \"default_and_protected\".",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"push_events": {
				Description: "Enable notifications for push events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"issues_events": {
				Description: "Enable notifications for issues events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"confidential_issues_events": {
				Description: "Enable notifications for confidential issues events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"merge_requests_events": {
				Description: "Enable notifications for merge requests events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"tag_push_events": {
				Description: "Enable notifications for tag push events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"note_events": {
				Description: "Enable notifications for note events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"confidential_note_events": {
				Description: "Enable notifications for confidential note events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"pipeline_events": {
				Description: "Enable notifications for pipeline events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"wiki_page_events": {
				Description: "Enable notifications for wiki page events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"push_channel": {
				Description: "The name of the channel to receive push events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"issue_channel": {
				Description: "The name of the channel to receive issue events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"confidential_issue_channel": {
				Description: "The name of the channel to receive confidential issue events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"merge_request_channel": {
				Description: "The name of the channel to receive merge request events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"note_channel": {
				Description: "The name of the channel to receive note events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"confidential_note_channel": {
				Description: "The name of the channel to receive confidential note events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"tag_push_channel": {
				Description: "The name of the channel to receive tag push events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"pipeline_channel": {
				Description: "The name of the channel to receive pipeline events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"wiki_page_channel": {
				Description: "The name of the channel to receive wiki page events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}
})

func resourceGitlabIntegrationMattermostCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	d.SetId(project)

	log.Printf("[DEBUG] create gitlab mattermost integration for project %s", project)

	opts := &gitlab.SetMattermostServiceOptions{
		WebHook: gitlab.String(d.Get("webhook").(string)),
	}

	opts.Username = gitlab.String(d.Get("username").(string))
	// Channel is not yet implemented by GitLab, contrary to what the documentation states
	// See https://gitlab.com/gitlab-org/gitlab/-/blob/902aaf4b412dc61a165b588611885cd60afb7a69/app/models/integrations/base_chat_notification.rb#L170
	//opts.Channel = gitlab.String(d.Get("channel").(string))
	opts.NotifyOnlyBrokenPipelines = gitlab.Bool(d.Get("notify_only_broken_pipelines").(bool))
	opts.BranchesToBeNotified = gitlab.String(d.Get("branches_to_be_notified").(string))
	opts.PushEvents = gitlab.Bool(d.Get("push_events").(bool))
	opts.IssuesEvents = gitlab.Bool(d.Get("issues_events").(bool))
	opts.ConfidentialIssuesEvents = gitlab.Bool(d.Get("confidential_issues_events").(bool))
	opts.MergeRequestsEvents = gitlab.Bool(d.Get("merge_requests_events").(bool))
	opts.TagPushEvents = gitlab.Bool(d.Get("tag_push_events").(bool))
	opts.NoteEvents = gitlab.Bool(d.Get("note_events").(bool))
	opts.ConfidentialNoteEvents = gitlab.Bool(d.Get("confidential_note_events").(bool))
	opts.PipelineEvents = gitlab.Bool(d.Get("pipeline_events").(bool))
	opts.WikiPageEvents = gitlab.Bool(d.Get("wiki_page_events").(bool))
	opts.PushChannel = gitlab.String(d.Get("push_channel").(string))
	opts.IssueChannel = gitlab.String(d.Get("issue_channel").(string))
	opts.ConfidentialIssueChannel = gitlab.String(d.Get("confidential_issue_channel").(string))
	opts.MergeRequestChannel = gitlab.String(d.Get("merge_request_channel").(string))
	opts.NoteChannel = gitlab.String(d.Get("note_channel").(string))
	opts.ConfidentialNoteChannel = gitlab.String(d.Get("confidential_note_channel").(string))
	opts.TagPushChannel = gitlab.String(d.Get("tag_push_channel").(string))
	opts.PipelineChannel = gitlab.String(d.Get("pipeline_channel").(string))
	opts.WikiPageChannel = gitlab.String(d.Get("wiki_page_channel").(string))

	_, err := client.Services.SetMattermostService(project, opts, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabIntegrationMattermostRead(ctx, d, meta)
}

func resourceGitlabIntegrationMattermostRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*gitlab.Client)
	project := d.Id()

	log.Printf("[DEBUG] read gitlab mattermost integration for project %s", project)

	service, _, err := client.Services.GetMattermostService(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			log.Printf("[DEBUG] gitlab mattermost integration not found %s", project)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// The webhook is explicitly not set anymore, due to being removed from the API. It will now
	// use whatever is in the configuration to determine the value.
	// See https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/1421 for more info.
	//d.Set("webhook", service.Properties.WebHook)
	// Channel is not yet implemented by GitLab, contrary to what the documentation states
	// See https://gitlab.com/gitlab-org/gitlab/-/blob/902aaf4b412dc61a165b588611885cd60afb7a69/app/models/integrations/base_chat_notification.rb#L170
	//d.Set("channel", service.Properties.Channel)

	d.Set("project", project)
	d.Set("username", service.Properties.Username)
	d.Set("notify_only_broken_pipelines", bool(service.Properties.NotifyOnlyBrokenPipelines))
	d.Set("branches_to_be_notified", service.Properties.BranchesToBeNotified)
	d.Set("push_events", service.PushEvents)
	d.Set("issues_events", service.IssuesEvents)
	d.Set("confidential_issues_events", service.ConfidentialIssuesEvents)
	d.Set("merge_requests_events", service.MergeRequestsEvents)
	d.Set("tag_push_events", service.TagPushEvents)
	d.Set("note_events", service.NoteEvents)
	d.Set("confidential_note_events", service.ConfidentialNoteEvents)
	d.Set("pipeline_events", service.PipelineEvents)
	d.Set("wiki_page_events", service.WikiPageEvents)
	d.Set("push_channel", service.Properties.PushChannel)
	d.Set("issue_channel", service.Properties.IssueChannel)
	d.Set("confidential_issue_channel", service.Properties.ConfidentialIssueChannel)
	d.Set("merge_request_channel", service.Properties.MergeRequestChannel)
	d.Set("note_channel", service.Properties.NoteChannel)
	d.Set("confidential_note_channel", service.Properties.ConfidentialNoteChannel)
	d.Set("tag_push_channel", service.Properties.TagPushChannel)
	d.Set("pipeline_channel", service.Properties.PipelineChannel)
	d.Set("wiki_page_channel", service.Properties.WikiPageChannel)

	return diags
}

func resourceGitlabIntegrationMattermostUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceGitlabIntegrationMattermostCreate(ctx, d, meta)
}

func resourceGitlabIntegrationMattermostDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Id()

	log.Printf("[DEBUG] delete gitlab mattermost service for project %s", project)

	_, err := client.Services.DeleteMattermostService(project, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
