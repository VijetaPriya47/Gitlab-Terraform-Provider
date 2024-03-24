package sdk

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_group_hook", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_group_hook` + "`" + ` resource allows to manage the lifecycle of a group hook.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/groups.html#hooks)`,

		CreateContext: resourceGitlabGroupHookCreate,
		ReadContext:   resourceGitlabGroupHookRead,
		UpdateContext: resourceGitlabGroupHookUpdate,
		DeleteContext: resourceGitlabGroupHookDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: gitlabGroupHookSchema(),
	}
})

func resourceGitlabGroupHookCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	group := d.Get("group").(string)
	options := &gitlab.AddGroupHookOptions{
		URL:                      gitlab.Ptr(d.Get("url").(string)),
		PushEvents:               gitlab.Ptr(d.Get("push_events").(bool)),
		PushEventsBranchFilter:   gitlab.Ptr(d.Get("push_events_branch_filter").(string)),
		IssuesEvents:             gitlab.Ptr(d.Get("issues_events").(bool)),
		ConfidentialIssuesEvents: gitlab.Ptr(d.Get("confidential_issues_events").(bool)),
		MergeRequestsEvents:      gitlab.Ptr(d.Get("merge_requests_events").(bool)),
		TagPushEvents:            gitlab.Ptr(d.Get("tag_push_events").(bool)),
		NoteEvents:               gitlab.Ptr(d.Get("note_events").(bool)),
		ConfidentialNoteEvents:   gitlab.Ptr(d.Get("confidential_note_events").(bool)),
		JobEvents:                gitlab.Ptr(d.Get("job_events").(bool)),
		PipelineEvents:           gitlab.Ptr(d.Get("pipeline_events").(bool)),
		WikiPageEvents:           gitlab.Ptr(d.Get("wiki_page_events").(bool)),
		DeploymentEvents:         gitlab.Ptr(d.Get("deployment_events").(bool)),
		ReleasesEvents:           gitlab.Ptr(d.Get("releases_events").(bool)),
		SubGroupEvents:           gitlab.Ptr(d.Get("subgroup_events").(bool)),
		EnableSSLVerification:    gitlab.Ptr(d.Get("enable_ssl_verification").(bool)),
		CustomWebhookTemplate:    gitlab.Ptr(d.Get("custom_webhook_template").(string)),
	}

	if v, ok := d.GetOk("token"); ok {
		options.Token = gitlab.Ptr(v.(string))
	}

	log.Printf("[DEBUG] create gitlab group hook %q", *options.URL)

	hook, _, err := client.Groups.AddGroupHook(group, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceGitlabGroupHookBuildID(group, hook.ID))
	d.Set("token", options.Token)

	return resourceGitlabGroupHookRead(ctx, d, meta)
}

func resourceGitlabGroupHookRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	group, hookID, err := resourceGitlabGroupHookParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] read gitlab group hook %s/%d", group, hookID)

	client := meta.(*gitlab.Client)
	hook, _, err := client.Groups.GetGroupHook(group, hookID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			log.Printf("[DEBUG] gitlab group hook not found %s/%d, removing from state", group, hookID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	stateMap := gitlabGroupHookToStateMap(group, hook)
	if err = setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabGroupHookUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	group, hookID, err := resourceGitlabGroupHookParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	client := meta.(*gitlab.Client)
	options := &gitlab.EditGroupHookOptions{
		URL:                      gitlab.Ptr(d.Get("url").(string)),
		PushEvents:               gitlab.Ptr(d.Get("push_events").(bool)),
		PushEventsBranchFilter:   gitlab.Ptr(d.Get("push_events_branch_filter").(string)),
		IssuesEvents:             gitlab.Ptr(d.Get("issues_events").(bool)),
		ConfidentialIssuesEvents: gitlab.Ptr(d.Get("confidential_issues_events").(bool)),
		MergeRequestsEvents:      gitlab.Ptr(d.Get("merge_requests_events").(bool)),
		TagPushEvents:            gitlab.Ptr(d.Get("tag_push_events").(bool)),
		NoteEvents:               gitlab.Ptr(d.Get("note_events").(bool)),
		ConfidentialNoteEvents:   gitlab.Ptr(d.Get("confidential_note_events").(bool)),
		JobEvents:                gitlab.Ptr(d.Get("job_events").(bool)),
		PipelineEvents:           gitlab.Ptr(d.Get("pipeline_events").(bool)),
		WikiPageEvents:           gitlab.Ptr(d.Get("wiki_page_events").(bool)),
		DeploymentEvents:         gitlab.Ptr(d.Get("deployment_events").(bool)),
		ReleasesEvents:           gitlab.Ptr(d.Get("releases_events").(bool)),
		SubGroupEvents:           gitlab.Ptr(d.Get("subgroup_events").(bool)),
		EnableSSLVerification:    gitlab.Ptr(d.Get("enable_ssl_verification").(bool)),
		CustomWebhookTemplate:    gitlab.Ptr(d.Get("custom_webhook_template").(string)),
	}

	if d.HasChange("token") {
		options.Token = gitlab.Ptr(d.Get("token").(string))
	}

	log.Printf("[DEBUG] update gitlab group hook %s", d.Id())

	_, _, err = client.Groups.EditGroupHook(group, hookID, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabGroupHookRead(ctx, d, meta)
}

func resourceGitlabGroupHookDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	group, hookID, err := resourceGitlabGroupHookParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] Delete gitlab group hook %s/%d", group, hookID)

	client := meta.(*gitlab.Client)
	_, err = client.Groups.DeleteGroupHook(group, hookID, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitlabGroupHookBuildID(group string, agentID int) string {
	return fmt.Sprintf("%s:%d", group, agentID)
}

func resourceGitlabGroupHookParseID(id string) (string, int, error) {
	groupID, rawHookID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	hookID, err := strconv.Atoi(rawHookID)
	if err != nil {
		return "", 0, err
	}

	return groupID, hookID, nil
}
