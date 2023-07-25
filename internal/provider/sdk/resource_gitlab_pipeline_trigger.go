package sdk

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_pipeline_trigger", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_pipeline_trigger` + "`" + ` resource allows to manage the lifecycle of a pipeline trigger.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/pipeline_triggers.html)`,

		CreateContext: resourceGitlabPipelineTriggerCreate,
		ReadContext:   resourceGitlabPipelineTriggerRead,
		UpdateContext: resourceGitlabPipelineTriggerUpdate,
		DeleteContext: resourceGitlabPipelineTriggerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema:        gitlabPipelineTriggerSchema(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabPipelineTriggerResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabPipelineTriggerStateUpgradeV0,
				Version: 0,
			},
		},
	}
})

func gitlabPipelineTriggerSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"pipeline_trigger_id": {
			Description: "The pipeline trigger id.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		"project": {
			Description: "The name or id of the project to add the trigger to.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"description": {
			Description: "The description of the pipeline trigger.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"token": {
			Description: "The pipeline trigger token.",
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
		},
	}
}

// resourceGitlabPipelineTriggerResourceV0 returns the V0 schema definition.
// From V0-V1 the `id` attribute value format changed from `<pipeline-trigger-id>` to `<project-id>:<pipeline-trigger-id>:<key>`,
// which means that the actual schema definition was not impacted and we can just return the
// V1 schema as V0 schema.
func resourceGitlabPipelineTriggerResourceV0() *schema.Resource {
	return &schema.Resource{Schema: gitlabPipelineTriggerSchema()}
}

// resourceGitlabPipelineTriggerStateUpgradeV0 performs the state migration from V0 to V1.
func resourceGitlabPipelineTriggerStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	project := rawState["project"].(string)
	oldId := rawState["id"].(string)

	pipelineTriggerId, err := strconv.Atoi(oldId)
	if err != nil {
		return nil, fmt.Errorf("unable to convert pipeline trigger id %q to integer to migrate to new schema: %w", oldId, err)
	}

	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]interface{}{"project": project, "v0-id": oldId})
	rawState["id"] = resourceGitlabPipelineTriggerBuildId(project, pipelineTriggerId)
	tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]interface{}{"v0-id": oldId, "v1-id": rawState["id"]})
	return rawState, nil
}

func resourceGitlabPipelineTriggerBuildId(project string, pipelineTriggerId int) string {
	id := fmt.Sprintf("%d", pipelineTriggerId)
	return utils.BuildTwoPartID(&project, &id)
}

func resourceGitlabPipelineTriggerParseId(id string) (string, int, error) {
	project, rawPipelineTriggerId, err := utils.ParseTwoPartID(id)
	e := fmt.Errorf("unable to parse id %q. Expected format <project>:<pipeline-trigger-id>", id)
	if err != nil {
		return "", 0, e
	}

	pipelineTriggerId, err := strconv.Atoi(rawPipelineTriggerId)
	if err != nil {
		return "", 0, e
	}

	return project, pipelineTriggerId, nil
}

func resourceGitlabPipelineTriggerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	options := &gitlab.AddPipelineTriggerOptions{
		Description: gitlab.String(d.Get("description").(string)),
	}

	log.Printf("[DEBUG] create gitlab PipelineTrigger %s", *options.Description)

	pipelineTrigger, _, err := client.PipelineTriggers.AddPipelineTrigger(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceGitlabPipelineTriggerBuildId(project, pipelineTrigger.ID))
	return resourceGitlabPipelineTriggerRead(ctx, d, meta)
}

func resourceGitlabPipelineTriggerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, pipelineTriggerId, err := resourceGitlabPipelineTriggerParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] read gitlab PipelineTrigger %s/%d", project, pipelineTriggerId)

	pipelineTrigger, _, err := client.PipelineTriggers.GetPipelineTrigger(project, pipelineTriggerId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			log.Printf("[DEBUG] gitlab pipeline trigger not found %s/%d", project, pipelineTriggerId)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("pipeline_trigger_id", pipelineTrigger.ID)
	d.Set("project", project)
	d.Set("description", pipelineTrigger.Description)
	d.Set("token", pipelineTrigger.Token)

	return nil
}

func resourceGitlabPipelineTriggerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, pipelineTriggerId, err := resourceGitlabPipelineTriggerParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.EditPipelineTriggerOptions{
		Description: gitlab.String(d.Get("description").(string)),
	}

	if d.HasChange("description") {
		options.Description = gitlab.String(d.Get("description").(string))
	}

	log.Printf("[DEBUG] update gitlab PipelineTrigger %s", d.Id())

	_, _, err = client.PipelineTriggers.EditPipelineTrigger(project, pipelineTriggerId, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabPipelineTriggerRead(ctx, d, meta)
}

func resourceGitlabPipelineTriggerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, pipelineTriggerId, err := resourceGitlabPipelineTriggerParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] Delete gitlab PipelineTrigger %s", d.Id())

	_, err = client.PipelineTriggers.DeletePipelineTrigger(project, pipelineTriggerId, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
