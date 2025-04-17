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

var _ = registerResource("gitlab_integration_slack", func() *schema.Resource {
	return resourceGitlabIntegrationSlackSchema(`The ` + "`gitlab_integration_slack`" + ` resource allows to manage the lifecycle of a project integration with Slack.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#slack-notifications)`)
})

var _ = registerResource("gitlab_service_slack", func() *schema.Resource {
	schema := resourceGitlabIntegrationSlackSchema(`The ` + "`gitlab_service_slack`" + ` resource allows to manage the lifecycle of a project integration with Slack.

~> This resource is deprecated. use ` + "`gitlab_integration_slack`" + `instead!

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#slack-notifications)`)
	schema.DeprecationMessage = `This resource is deprecated. use ` + "`gitlab_integration_slack`" + `instead!`
	return schema
})

func resourceGitlabIntegrationSlackSchema(description string) *schema.Resource {
	return &schema.Resource{
		Description: description,

		CreateContext: resourceGitlabIntegrationSlackCreate,
		ReadContext:   resourceGitlabIntegrationSlackRead,
		UpdateContext: resourceGitlabIntegrationSlackUpdate,
		DeleteContext: resourceGitlabIntegrationSlackDelete,
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
				Description: "Webhook URL (Example, https://hooks.slack.com/services/...). This value cannot be imported.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"username": {
				Description: "Username to use.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"notify_only_broken_pipelines": {
				Description: "Send notifications for broken pipelines.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"notify_only_default_branch": {
				Description: "This parameter has been replaced with `branches_to_be_notified`.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Deprecated:  "use 'branches_to_be_notified' argument instead",
			},
			"branches_to_be_notified": {
				Description: "Branches to send notifications for. Valid options are \"all\", \"default\", \"protected\", and \"default_and_protected\".",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			// TODO: Currently, client-go doesn't implement this option yet.
			//       see https://gitlab.com/gitlab-org/api/client-go/issues/1354
			// "commit_events": {
			// 	Description: "Enable notifications for commit events.",
			// 	Type:        schema.TypeBool,
			// 	Optional:    true,
			// 	Computed:    true,
			// },
			"confidential_issue_channel": {
				Description: "The name of the channel to receive confidential issue events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"confidential_issues_events": {
				Description: "Enable notifications for confidential issues events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"confidential_note_channel": {
				Description: "The name of the channel to receive confidential note events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"confidential_note_events": {
				Description: "Enable notifications for confidential note events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			// TODO: Currently, GitLab doesn't correctly implement the API, so this is
			//       impossible to implement here at the moment.
			//       see https://gitlab.com/gitlab-org/gitlab/-/issues/28903
			// "deployment_channel": {
			// 	Description: "The name of the channel to receive deployment events notifications.",
			// 	Type:        schema.TypeString,
			// 	Optional:    true,
			// },
			// "deployment_events": {
			// 	Description: "Enable notifications for deployment events.",
			// 	Type:        schema.TypeBool,
			// 	Optional:    true,
			// 	Computed:    true,
			// },
			"issue_channel": {
				Description: "The name of the channel to receive issue events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"issues_events": {
				Description: "Enable notifications for issues events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			// TODO: Currently, client-go doesn't implement this option yet.
			//       see https://gitlab.com/gitlab-org/api/client-go/issues/1354
			"job_events": {
				Description: "Enable notifications for job events. **ATTENTION**: This attribute is currently not being submitted to the GitLab API, due to https://gitlab.com/gitlab-org/api/client-go/issues/1354.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"merge_request_channel": {
				Description: "The name of the channel to receive merge request events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"merge_requests_events": {
				Description: "Enable notifications for merge requests events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"note_channel": {
				Description: "The name of the channel to receive note events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"note_events": {
				Description: "Enable notifications for note events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"pipeline_channel": {
				Description: "The name of the channel to receive pipeline events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"pipeline_events": {
				Description: "Enable notifications for pipeline events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"push_channel": {
				Description: "The name of the channel to receive push events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"push_events": {
				Description: "Enable notifications for push events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"tag_push_channel": {
				Description: "The name of the channel to receive tag push events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"tag_push_events": {
				Description: "Enable notifications for tag push events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"wiki_page_channel": {
				Description: "The name of the channel to receive wiki page events notifications.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"wiki_page_events": {
				Description: "Enable notifications for wiki page events.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func resourceGitlabIntegrationSlackCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	d.SetId(project)

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab slack integration for project %s", project))

	opts := &gitlab.SetSlackServiceOptions{
		WebHook: gitlab.Ptr(d.Get("webhook").(string)),
	}

	opts.Username = gitlab.Ptr(d.Get("username").(string))
	opts.NotifyOnlyBrokenPipelines = gitlab.Ptr(d.Get("notify_only_broken_pipelines").(bool))
	opts.NotifyOnlyDefaultBranch = gitlab.Ptr(d.Get("notify_only_default_branch").(bool))
	opts.BranchesToBeNotified = gitlab.Ptr(d.Get("branches_to_be_notified").(string))
	// TODO: Currently, client-go doesn't implement this option yet.
	//       see https://gitlab.com/gitlab-org/api/client-go/issues/1354
	// opts.CommitEvents = gitlab.Ptr(d.Get("commit_events").(bool))
	opts.ConfidentialIssueChannel = gitlab.Ptr(d.Get("confidential_issue_channel").(string))
	opts.ConfidentialIssuesEvents = gitlab.Ptr(d.Get("confidential_issues_events").(bool))
	opts.ConfidentialNoteChannel = gitlab.Ptr(d.Get("confidential_note_channel").(string))
	opts.ConfidentialNoteEvents = gitlab.Ptr(d.Get("confidential_note_events").(bool))
	// TODO: Currently, GitLab doesn't correctly implement the API, so this is
	//       impossible to implement here at the moment.
	//       see https://gitlab.com/gitlab-org/gitlab/-/issues/28903
	// opts.DeploymentChannel = gitlab.Ptr(d.Get("deployment_channel").(string))
	// opts.DeploymentEvents = gitlab.Ptr(d.Get("deployment_events").(bool))
	opts.IssueChannel = gitlab.Ptr(d.Get("issue_channel").(string))
	opts.IssuesEvents = gitlab.Ptr(d.Get("issues_events").(bool))
	// TODO: Currently, client-go doesn't implement this option yet.
	//       see https://gitlab.com/gitlab-org/api/client-go/issues/1354
	// opts.JobEvents = gitlab.Ptr(d.Get("job_events").(bool))
	opts.MergeRequestChannel = gitlab.Ptr(d.Get("merge_request_channel").(string))
	opts.MergeRequestsEvents = gitlab.Ptr(d.Get("merge_requests_events").(bool))
	opts.NoteChannel = gitlab.Ptr(d.Get("note_channel").(string))
	opts.NoteEvents = gitlab.Ptr(d.Get("note_events").(bool))
	opts.PipelineChannel = gitlab.Ptr(d.Get("pipeline_channel").(string))
	opts.PipelineEvents = gitlab.Ptr(d.Get("pipeline_events").(bool))
	opts.PushChannel = gitlab.Ptr(d.Get("push_channel").(string))
	opts.PushEvents = gitlab.Ptr(d.Get("push_events").(bool))
	opts.TagPushChannel = gitlab.Ptr(d.Get("tag_push_channel").(string))
	opts.TagPushEvents = gitlab.Ptr(d.Get("tag_push_events").(bool))
	opts.WikiPageChannel = gitlab.Ptr(d.Get("wiki_page_channel").(string))
	opts.WikiPageEvents = gitlab.Ptr(d.Get("wiki_page_events").(bool))

	_, _, err := client.Services.SetSlackService(project, opts, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabIntegrationSlackRead(ctx, d, meta)
}

func resourceGitlabIntegrationSlackRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab slack integration for project %s", project))

	service, _, err := client.Services.GetSlackService(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab slack integration not found %s", project))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// The webhook is explicitly not set anymore, due to being removed from the API. It will now
	// use whatever is in the configuration to determine the value.
	// See https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/1421 for more info.
	// d.Set("webhook", service.Properties.WebHook)

	d.Set("project", project)
	d.Set("username", service.Properties.Username)
	d.Set("notify_only_broken_pipelines", bool(service.Properties.NotifyOnlyBrokenPipelines))
	d.Set("notify_only_default_branch", bool(service.Properties.NotifyOnlyDefaultBranch))
	d.Set("branches_to_be_notified", service.Properties.BranchesToBeNotified)
	d.Set("confidential_issue_channel", service.Properties.ConfidentialIssueChannel)
	d.Set("confidential_issues_events", service.ConfidentialIssuesEvents)
	d.Set("confidential_note_channel", service.Properties.ConfidentialNoteChannel)
	d.Set("confidential_note_events", service.ConfidentialNoteEvents)
	// TODO: Currently, GitLab doesn't correctly implement the API, so this is
	//       impossible to implement here at the moment.
	//       see https://gitlab.com/gitlab-org/gitlab/-/issues/28903
	// d.Set("deployment_channel", service.Properties.DeploymentChannel)
	// d.Set("deployment_events", service.DeploymentEvents)
	d.Set("issue_channel", service.Properties.IssueChannel)
	d.Set("issues_events", service.IssuesEvents)
	// TODO: Currently, client-go doesn't implement this option yet.
	//       see https://gitlab.com/gitlab-org/api/client-go/issues/1354
	d.Set("job_events", service.JobEvents)
	d.Set("merge_request_channel", service.Properties.MergeRequestChannel)
	d.Set("merge_requests_events", service.MergeRequestsEvents)
	d.Set("note_channel", service.Properties.NoteChannel)
	d.Set("note_events", service.NoteEvents)
	d.Set("pipeline_channel", service.Properties.PipelineChannel)
	d.Set("pipeline_events", service.PipelineEvents)
	d.Set("push_channel", service.Properties.PushChannel)
	d.Set("push_events", service.PushEvents)
	d.Set("tag_push_channel", service.Properties.TagPushChannel)
	d.Set("tag_push_events", service.TagPushEvents)
	d.Set("wiki_page_channel", service.Properties.WikiPageChannel)
	d.Set("wiki_page_events", service.WikiPageEvents)

	return nil
}

func resourceGitlabIntegrationSlackUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceGitlabIntegrationSlackCreate(ctx, d, meta)
}

func resourceGitlabIntegrationSlackDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] delete gitlab slack service for project %s", project))

	_, err := client.Services.DeleteSlackService(project, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
