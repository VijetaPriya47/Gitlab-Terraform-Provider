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

var _ = registerResource("gitlab_integration_jira", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_integration_jira`" + ` resource allows to manage the lifecycle of a project integration with Jira.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#jira-issues)`,

		CreateContext: resourceGitlabIntegrationJiraCreate,
		ReadContext:   resourceGitlabIntegrationJiraRead,
		UpdateContext: resourceGitlabIntegrationJiraUpdate,
		DeleteContext: resourceGitlabIntegrationJiraDelete,
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
			// This value is always set to `true` when creating the service, even if `false` is passed into the API. Deleting the service sets it to `false`.
			"active": {
				Description: "Whether the integration is active.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"url": {
				Description:  "The URL to the JIRA project which is being linked to this GitLab project. For example, https://jira.example.com.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateURLFunc,
			},
			"api_url": {
				Description:  "The base URL to the Jira instance API. Web URL value is used if not set. For example, https://jira-api.example.com.",
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateURLFunc,
			},
			"project_key": {
				Description: "The short identifier for your JIRA project. Must be all uppercase. For example, `PROJ`.",
				Deprecated:  "`project_key` is deprecated. Use `project_keys` instead.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},
			"username": {
				Description: "The email or username to be used with Jira. For Jira Cloud use an email, for Jira Data Center and Jira Server use a username. Required when using Basic authentication (jira_auth_type is 0).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"password": {
				Description: "The Jira API token, password, or personal access token to be used with Jira. When your authentication method is basic (jira_auth_type is 0), use an API token for Jira Cloud or a password for Jira Data Center or Jira Server. When your authentication method is a Jira personal access token (jira_auth_type is 1), use the personal access token.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"jira_auth_type": {
				Description: "The authentication method to be used with Jira. 0 means Basic Authentication. 1 means Jira personal access token. Defaults to 0.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"jira_issue_prefix": {
				Description: "Prefix to match Jira issue keys.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"jira_issue_regex": {
				Description: "Regular expression to match Jira issue keys.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"jira_issue_transition_automatic": {
				Description: "Enable automatic issue transitions. Takes precedence over jira_issue_transition_id if enabled. Defaults to false. This value cannot be imported, and will not perform drift detection if changed outside Terraform.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"jira_issue_transition_id": {
				Description: "The ID of a transition that moves issues to a closed state. You can find this number under the JIRA workflow administration (Administration > Issues > Workflows) by selecting View under Operations of the desired workflow of your project. By default, this ID is set to 2.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"commit_events": {
				Description: "Enable notifications for commit events",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"merge_requests_events": {
				Description: "Enable notifications for merge request events",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"comment_on_event_enabled": {
				Description: "Enable comments inside Jira issues on each GitLab event (commit / merge request)",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"issues_enabled": {
				Description: "Enable viewing Jira issues in GitLab.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"project_keys": {
				Description: "Keys of Jira projects. When issues_enabled is true, this setting specifies which Jira projects to view issues from in GitLab.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"use_inherited_settings": {
				Description: "Indicates whether or not to inherit default settings. Defaults to false.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
})

func resourceGitlabIntegrationJiraCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	project := d.Get("project").(string)

	opts := &gitlab.SetJiraServiceOptions{}

	jiraProjectKey := d.Get("project_key").(string)
	opts.ProjectKeys = &[]string{jiraProjectKey}

	jiraAuthType := gitlab.Ptr(d.Get("jira_auth_type").(int))
	if *jiraAuthType == 0 {
		opts.Username = gitlab.Ptr(d.Get("username").(string))
		opts.Password = gitlab.Ptr(d.Get("password").(string))
	} else {
		opts.Password = gitlab.Ptr(d.Get("password").(string))
	}
	opts.JiraAuthType = jiraAuthType

	opts.URL = gitlab.Ptr(d.Get("url").(string))
	opts.CommitEvents = gitlab.Ptr(d.Get("commit_events").(bool))
	opts.MergeRequestsEvents = gitlab.Ptr(d.Get("merge_requests_events").(bool))
	opts.CommentOnEventEnabled = gitlab.Ptr(d.Get("comment_on_event_enabled").(bool))
	opts.APIURL = gitlab.Ptr(d.Get("api_url").(string))
	opts.JiraIssueTransitionID = gitlab.Ptr(d.Get("jira_issue_transition_id").(string))
	opts.JiraIssuePrefix = gitlab.Ptr(d.Get("jira_issue_prefix").(string))
	opts.JiraIssueRegex = gitlab.Ptr(d.Get("jira_issue_regex").(string))
	opts.JiraIssueTransitionAutomatic = gitlab.Ptr(d.Get("jira_issue_transition_automatic").(bool))
	opts.IssuesEnabled = gitlab.Ptr(d.Get("issues_enabled").(bool))
	opts.UseInheritedSettings = gitlab.Ptr(d.Get("use_inherited_settings").(bool))

	tflog.Debug(ctx, "[DEBUG] Create Gitlab Jira integration")
	if _, _, err := client.Services.SetJiraService(project, opts, gitlab.WithContext(ctx)); err != nil {
		return diag.Errorf("couldn't create Gitlab Jira service: %v", err)
	}

	d.SetId(project)

	return resourceGitlabIntegrationJiraRead(ctx, d, meta)
}

func resourceGitlabIntegrationJiraRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Read Gitlab Jira integration %s", project))

	jiraService, _, err := client.Services.GetJiraService(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab jira integration not found %s, removing from state", project))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("project", project)
	d.Set("url", jiraService.Properties.URL)
	d.Set("api_url", jiraService.Properties.APIURL)
	d.Set("username", jiraService.Properties.Username)
	d.Set("title", jiraService.Title)
	d.Set("created_at", jiraService.CreatedAt.String())
	d.Set("updated_at", jiraService.UpdatedAt.String())
	d.Set("jira_auth_type", jiraService.Properties.JiraAuthType)
	d.Set("jira_issue_prefix", jiraService.Properties.JiraIssuePrefix)
	d.Set("jira_issue_regex", jiraService.Properties.JiraIssueRegex)
	// Note for support - if someone is using provider version 16.0+, it's not compatible with GitLab 15.2-, because there
	// was an issue with how the JIRA transition IDs were formatted in the API. Support for that was removed in 16.0.
	d.Set("jira_issue_transition_id", jiraService.Properties.JiraIssueTransitionID)
	// Note: jira_issue_transition_automtaic is not returned via API so we cannot read it for the state.
	// d.Set("jira_issue_transition_automatic", jiraService.Properties.JiraIssueTransitionAutomatic)
	d.Set("commit_events", jiraService.CommitEvents)
	d.Set("merge_requests_events", jiraService.MergeRequestsEvents)
	d.Set("comment_on_event_enabled", jiraService.CommentOnEventEnabled)
	d.Set("issues_enabled", jiraService.Properties.IssuesEnabled)
	d.Set("use_inherited_settings", jiraService.Inherited)
	d.Set("active", jiraService.Active)

	// Match pre-existing behavior of a single key until we support the new multi-key approach.
	// If we're running before 17.0, we have to use the deprecated ProjectKey (singular)
	isVersionAtLeast17, err := api.IsGitLabVersionAtLeast(ctx, client, "17.0")()
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to determine version of GitLab. Cannot determine which API property to read from. Error: %v", err))
	}
	if isVersionAtLeast17 {
		if len(jiraService.Properties.ProjectKeys) > 0 {
			d.Set("project_key", jiraService.Properties.ProjectKeys[0])
		}
	} else {
		// While technically the TF provider seemed to support project_key pre 17, it's not documented in the 16.11
		// API documentation, and even when passed into the API it returns blank from the read API, so it causes issues.
		tflog.Debug(ctx, "Skipping setting JIRA Project Key since it isn't supported pre-GitLab 17.0")
	}

	return nil
}

func resourceGitlabIntegrationJiraUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceGitlabIntegrationJiraCreate(ctx, d, meta)
}

func resourceGitlabIntegrationJiraDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	project := d.Get("project").(string)

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete Gitlab Jira integration %s", d.Id()))

	_, err := client.Services.DeleteJiraService(project, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
