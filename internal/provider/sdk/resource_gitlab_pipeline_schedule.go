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

var _ = registerResource("gitlab_pipeline_schedule", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_pipeline_schedule` " + `resource allows to manage the lifecycle of a scheduled pipeline.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/pipeline_schedules.html)`,

		CreateContext: resourceGitlabPipelineScheduleCreate,
		ReadContext:   resourceGitlabPipelineScheduleRead,
		UpdateContext: resourceGitlabPipelineScheduleUpdate,
		DeleteContext: resourceGitlabPipelineScheduleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema:        gitlabPipelineScheduleSchema(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabPipelineScheduleResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabPipelineScheduleStateUpgradeV0,
				Version: 0,
			},
		},
	}
})

func gitlabPipelineScheduleSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"pipeline_schedule_id": {
			Description: "The pipeline schedule id.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		"project": {
			Description: "The name or id of the project to add the schedule to.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"description": {
			Description: "The description of the pipeline schedule.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"ref": {
			Description: "The branch/tag name to be triggered.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"cron": {
			Description: "The cron (e.g. `0 1 * * *`).",
			Type:        schema.TypeString,
			Required:    true,
		},
		"cron_timezone": {
			Description: "The timezone.",
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "UTC",
		},
		"active": {
			Description: "The activation of pipeline schedule. If false is set, the pipeline schedule will deactivated initially.",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
		},
		"take_ownership": {
			Description: "When set to `true`, the user represented by the token running Terraform will take ownership of the scheduled pipeline prior to editing it. This can help when managing scheduled pipeline drift when other users are making changes outside Terraform.",
			Type:        schema.TypeBool,
			Optional:    true,
		},
		"owner": {
			Description: "The ID of the user that owns the pipeline schedule.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
	}
}

// resourceGitlabPipelineScheduleResourceV0 returns the V0 schema definition.
// From V0-V1 the `id` attribute value format changed from `<pipeline-schedule-id>` to `<project-id>:<pipeline-schedule-id>:<key>`,
// which means that the actual schema definition was not impacted and we can just return the
// V1 schema as V0 schema.
func resourceGitlabPipelineScheduleResourceV0() *schema.Resource {
	return &schema.Resource{Schema: gitlabPipelineScheduleSchema()}
}

// resourceGitlabPipelineScheduleStateUpgradeV0 performs the state migration from V0 to V1.
func resourceGitlabPipelineScheduleStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	project := rawState["project"].(string)
	oldId := rawState["id"].(string)

	pipelineScheduleId, err := strconv.Atoi(oldId)
	if err != nil {
		return nil, fmt.Errorf("unable to convert pipeline schedule id %q to integer to migrate to new schema: %w", oldId, err)
	}

	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]interface{}{"project": project, "v0-id": oldId})
	rawState["id"] = resourceGitlabPipelineScheduleBuildId(project, pipelineScheduleId)
	tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]interface{}{"v0-id": oldId, "v1-id": rawState["id"]})
	return rawState, nil
}

func resourceGitlabPipelineScheduleBuildId(project string, pipelineScheduleId int) string {
	id := fmt.Sprintf("%d", pipelineScheduleId)
	return utils.BuildTwoPartID(&project, &id)
}

func resourceGitlabPipelineScheduleParseId(id string) (string, int, error) {
	project, rawPipelineScheduleId, err := utils.ParseTwoPartID(id)
	e := fmt.Errorf("unable to parse id %q. Expected format <project>:<pipeline-schedule-id>", id)
	if err != nil {
		return "", 0, e
	}

	pipelineScheduleId, err := strconv.Atoi(rawPipelineScheduleId)
	if err != nil {
		return "", 0, e
	}

	return project, pipelineScheduleId, nil
}

func resourceGitlabPipelineScheduleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	options := &gitlab.CreatePipelineScheduleOptions{
		Description:  gitlab.String(d.Get("description").(string)),
		Ref:          gitlab.String(d.Get("ref").(string)),
		Cron:         gitlab.String(d.Get("cron").(string)),
		CronTimezone: gitlab.String(d.Get("cron_timezone").(string)),
		Active:       gitlab.Bool(d.Get("active").(bool)),
	}

	log.Printf("[DEBUG] create gitlab PipelineSchedule %s", *options.Description)

	pipelineSchedule, _, err := client.PipelineSchedules.CreatePipelineSchedule(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceGitlabPipelineScheduleBuildId(project, pipelineSchedule.ID))
	return resourceGitlabPipelineScheduleRead(ctx, d, meta)
}

func resourceGitlabPipelineScheduleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, pipelineScheduleId, err := resourceGitlabPipelineScheduleParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] read gitlab PipelineSchedule %s/%d", project, pipelineScheduleId)

	pipelineSchedule, _, err := client.PipelineSchedules.GetPipelineSchedule(project, pipelineScheduleId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			log.Printf("[DEBUG] PipelineSchedule %d in project %s does not exist, removing from state", pipelineScheduleId, project)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("pipeline_schedule_id", pipelineSchedule.ID)
	d.Set("project", project)
	d.Set("description", pipelineSchedule.Description)
	d.Set("ref", pipelineSchedule.Ref)
	d.Set("cron", pipelineSchedule.Cron)
	d.Set("cron_timezone", pipelineSchedule.CronTimezone)
	d.Set("active", pipelineSchedule.Active)
	d.Set("owner", pipelineSchedule.Owner.ID)
	return nil
}

func resourceGitlabPipelineScheduleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, pipelineScheduleId, err := resourceGitlabPipelineScheduleParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.EditPipelineScheduleOptions{
		Description:  gitlab.String(d.Get("description").(string)),
		Ref:          gitlab.String(d.Get("ref").(string)),
		Cron:         gitlab.String(d.Get("cron").(string)),
		CronTimezone: gitlab.String(d.Get("cron_timezone").(string)),
		Active:       gitlab.Bool(d.Get("active").(bool)),
	}

	if d.HasChange("description") {
		options.Description = gitlab.String(d.Get("description").(string))
	}

	if d.HasChange("ref") {
		options.Ref = gitlab.String(d.Get("ref").(string))
	}

	if d.HasChange("cron") {
		options.Cron = gitlab.String(d.Get("cron").(string))
	}

	if d.HasChange("cron_timezone") {
		options.CronTimezone = gitlab.String(d.Get("cron_timezone").(string))
	}

	if d.HasChange("active") {
		options.Active = gitlab.Bool(d.Get("active").(bool))
	}

	if _, ok := d.GetOk("take_ownership"); ok {
		tflog.Debug(ctx, "[DEBUG] Taking ownership of gitlab PipelineSchedule.", map[string]interface{}{"scheduledPipeline": d.Id()})

		_, _, err := client.PipelineSchedules.TakeOwnershipOfPipelineSchedule(project, pipelineScheduleId, nil, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "[DEBUG] Updating gitlab PipelineSchedule", map[string]interface{}{"scheduledPipeline": d.Id()})
	_, _, err = client.PipelineSchedules.EditPipelineSchedule(project, pipelineScheduleId, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabPipelineScheduleRead(ctx, d, meta)
}

func resourceGitlabPipelineScheduleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, pipelineScheduleId, err := resourceGitlabPipelineScheduleParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] Delete gitlab PipelineSchedule %s", d.Id())

	if _, err = client.PipelineSchedules.DeletePipelineSchedule(project, pipelineScheduleId, gitlab.WithContext(ctx)); err != nil {
		return diag.Errorf("failed to delete pipeline schedule %q: %v", d.Id(), err)
	}
	return nil
}
