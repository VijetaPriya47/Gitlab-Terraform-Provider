package sdk

import (
	"context"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_project_hook", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_project_hook` + "`" + ` resource allows to manage the lifecycle of a project hook.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/projects.html#hooks)`,

		CreateContext: resourceGitlabProjectHookCreate,
		ReadContext:   resourceGitlabProjectHookRead,
		UpdateContext: resourceGitlabProjectHookUpdate,
		DeleteContext: resourceGitlabProjectHookDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema:        gitlabProjectHookSchema(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabProjectHookResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabProjectHookStateUpgradeV0,
				Version: 0,
			},
		},
	}
})

// resourceGitlabProjectHookResourceV0 returns the V0 schema definition.
// From V0-V1 the `id` attribute value format changed from `<hook-id>` to `<project-id>:<hook-id>`,
// which means that the actual schema definition was not impacted and we can just return the
// V1 schema as V0 schema.
func resourceGitlabProjectHookResourceV0() *schema.Resource {
	return &schema.Resource{Schema: gitlabProjectHookSchema()}
}

// resourceGitlabProjectHookStateUpgradeV0 performs the state migration from V0 to V1.
func resourceGitlabProjectHookStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	project := rawState["project"].(string)
	oldId := rawState["id"].(string)
	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]interface{}{"project": project, "v0-id": oldId})
	rawState["id"] = utils.BuildTwoPartID(&project, &oldId)
	tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]interface{}{"v0-id": oldId, "v1-id": rawState["id"]})
	return rawState, nil
}

func resourceGitlabProjectHookBuildId(project string, hookId int) string {
	h := strconv.Itoa(hookId)
	return utils.BuildTwoPartID(&project, &h)
}

func resourceGitlabProjectHookParseId(id string) (string, int, error) {
	project, rawHookId, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	hookId, err := strconv.Atoi(rawHookId)
	if err != nil {
		return "", 0, err
	}

	return project, hookId, nil
}

func resourceGitlabProjectHookCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	options := &gitlab.AddProjectHookOptions{
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
		EnableSSLVerification:    gitlab.Ptr(d.Get("enable_ssl_verification").(bool)),
	}

	if v, ok := d.GetOk("token"); ok {
		options.Token = gitlab.Ptr(v.(string))
	}

	log.Printf("[DEBUG] create gitlab project hook %q", *options.URL)

	hook, _, err := client.Projects.AddProjectHook(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceGitlabProjectHookBuildId(project, hook.ID))
	d.Set("token", options.Token)

	return resourceGitlabProjectHookRead(ctx, d, meta)
}

func resourceGitlabProjectHookRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, hookId, err := resourceGitlabProjectHookParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] read gitlab project hook %s/%d", project, hookId)

	hook, _, err := client.Projects.GetProjectHook(project, hookId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			log.Printf("[DEBUG] gitlab project hook not found %s/%d, removing from state", project, hookId)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	stateMap := gitlabProjectHookToStateMap(project, hook)
	if err = setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabProjectHookUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, hookId, err := resourceGitlabProjectHookParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	options := &gitlab.EditProjectHookOptions{
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
		EnableSSLVerification:    gitlab.Ptr(d.Get("enable_ssl_verification").(bool)),
	}

	if d.HasChange("token") {
		options.Token = gitlab.Ptr(d.Get("token").(string))
	}

	log.Printf("[DEBUG] update gitlab project hook %s", d.Id())

	_, _, err = client.Projects.EditProjectHook(project, hookId, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabProjectHookRead(ctx, d, meta)
}

func resourceGitlabProjectHookDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, hookId, err := resourceGitlabProjectHookParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] Delete gitlab project hook %s", d.Id())

	_, err = client.Projects.DeleteProjectHook(project, hookId, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
