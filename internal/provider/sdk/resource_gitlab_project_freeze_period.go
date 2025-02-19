package sdk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_project_freeze_period", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_project_freeze_period` + "`" + ` resource allows to manage the lifecycle of a freeze period for a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/freeze_periods/)`,

		CreateContext: resourceGitlabProjectFreezePeriodCreate,
		ReadContext:   resourceGitlabProjectFreezePeriodRead,
		UpdateContext: resourceGitlabProjectFreezePeriodUpdate,
		DeleteContext: resourceGitlabProjectFreezePeriodDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema:        gitlabProjectFreezePeriodSchemaV1(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabProjectFreezePeriodResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabProjectFreezePeriodStateUpgradeV0,
				Version: 0,
			},
		},
	}
})

func gitlabProjectFreezePeriodSchemaV1() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"project": {
			Description: "The ID or URL-encoded path of the project to add the schedule to.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"freeze_start": {
			Description: "Start of the Freeze Period in cron format (e.g. `0 1 * * *`).",
			Type:        schema.TypeString,
			Required:    true,
		},
		"freeze_end": {
			Description: "End of the Freeze Period in cron format (e.g. `0 2 * * *`).",
			Type:        schema.TypeString,
			Required:    true,
		},
		"cron_timezone": {
			Description: "The timezone.",
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "UTC",
		},
	}
}

func resourceGitlabProjectFreezePeriodResourceV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"project_id": {
				Description: "The ID of the project to add the schedule to.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"freeze_start": {
				Description: "Start of the Freeze Period in cron format (e.g. `0 1 * * *`).",
				Type:        schema.TypeString,
				Required:    true,
			},
			"freeze_end": {
				Description: "End of the Freeze Period in cron format (e.g. `0 2 * * *`).",
				Type:        schema.TypeString,
				Required:    true,
			},
			"cron_timezone": {
				Description: "The timezone.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "UTC",
			},
		},
	}
}

// resourceGitlabProjectFreezePeriodStateUpgradeV0 performs the state migration from V0 to V1.
func resourceGitlabProjectFreezePeriodStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	if rawState["project_id"] != nil {
		projectId, ok := rawState["project_id"].(string)
		if !ok {
			projectId = strconv.FormatInt(int64(rawState["project_id"].(float64)), 10)
		}
		rawState["project"] = projectId
		delete(rawState, "project_id")
		tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `project_id` attribute to `project`", map[string]interface{}{"project_id": projectId})
	}
	return rawState, nil
}

func resourceGitlabProjectFreezePeriodCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	project := d.Get("project").(string)

	options := gitlab.CreateFreezePeriodOptions{
		FreezeStart:  gitlab.Ptr(d.Get("freeze_start").(string)),
		FreezeEnd:    gitlab.Ptr(d.Get("freeze_end").(string)),
		CronTimezone: gitlab.Ptr(d.Get("cron_timezone").(string)),
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Project %s create gitlab project-level freeze period %+v", project, options))

	client := meta.(*gitlab.Client)
	FreezePeriod, _, err := client.FreezePeriods.CreateFreezePeriodOptions(project, &options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	FreezePeriodIDString := fmt.Sprintf("%d", FreezePeriod.ID)
	d.SetId(utils.BuildTwoPartID(&project, &FreezePeriodIDString))

	return resourceGitlabProjectFreezePeriodRead(ctx, d, meta)
}

func resourceGitlabProjectFreezePeriodRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, freezePeriodID, err := projectAndFreezePeriodIDFromID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab FreezePeriod %s/%d", project, freezePeriodID))

	freezePeriod, _, err := client.FreezePeriods.GetFreezePeriod(project, freezePeriodID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] project freeze period for %s not found so removing it from state", d.Id()))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("freeze_start", freezePeriod.FreezeStart)
	d.Set("freeze_end", freezePeriod.FreezeEnd)
	d.Set("cron_timezone", freezePeriod.CronTimezone)
	d.Set("project", project)

	return nil
}

func resourceGitlabProjectFreezePeriodUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, freezePeriodID, err := projectAndFreezePeriodIDFromID(d.Id())
	options := &gitlab.UpdateFreezePeriodOptions{}

	if err != nil {
		return diag.Errorf("%s cannot be converted to int", d.Id())
	}

	if d.HasChange("freeze_start") {
		options.FreezeStart = gitlab.Ptr(d.Get("freeze_start").(string))
	}

	if d.HasChange("freeze_end") {
		options.FreezeEnd = gitlab.Ptr(d.Get("freeze_end").(string))
	}

	if d.HasChange("cron_timezone") {
		options.CronTimezone = gitlab.Ptr(d.Get("cron_timezone").(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab FreezePeriod %s", d.Id()))

	_, _, err = client.FreezePeriods.UpdateFreezePeriodOptions(project, freezePeriodID, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabProjectFreezePeriodRead(ctx, d, meta)
}

func resourceGitlabProjectFreezePeriodDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, freezePeriodID, err := projectAndFreezePeriodIDFromID(d.Id())
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab FreezePeriod %s", d.Id()))

	if err != nil {
		return diag.Errorf("%s cannot be converted to int", d.Id())
	}

	if _, err = client.FreezePeriods.DeleteFreezePeriod(project, freezePeriodID, gitlab.WithContext(ctx)); err != nil {
		return diag.Errorf("failed to delete pipeline schedule %q: %v", d.Id(), err)
	}

	return nil
}

func projectAndFreezePeriodIDFromID(id string) (string, int, error) {
	project, freezePeriodIDString, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	freezePeriodID, err := strconv.Atoi(freezePeriodIDString)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get freezePeriodId: %v", err)
	}

	return project, freezePeriodID, nil
}
