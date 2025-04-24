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

var milestoneStateToStateEvent = map[string]string{
	"active": "activate",
	"closed": "close",
}

var _ = registerResource("gitlab_project_milestone", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_project_milestone`" + ` resource allows to manage the lifecycle of a project milestone.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/milestones/)`,

		CreateContext: resourceGitlabProjectMilestoneCreate,
		ReadContext:   resourceGitlabProjectMilestoneRead,
		UpdateContext: resourceGitlabProjectMilestoneUpdate,
		DeleteContext: resourceGitlabProjectMilestoneDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: gitlabProjectMilestoneGetSchema(),
	}
})

func resourceGitlabProjectMilestoneCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	title := d.Get("title").(string)

	options := &gitlab.CreateMilestoneOptions{
		Title: &title,
	}
	if description, ok := d.GetOk("description"); ok {
		options.Description = gitlab.Ptr(description.(string))
	}
	if startDate, ok := d.GetOk("start_date"); ok {
		parsedStartDate, err := parseISO8601Date(startDate.(string))
		if err != nil {
			return diag.Errorf("Failed to parse start_date: %s. %v", startDate.(string), err)
		}
		options.StartDate = parsedStartDate
	}
	if dueDate, ok := d.GetOk("due_date"); ok {
		parsedDueDate, err := parseISO8601Date(dueDate.(string))
		if err != nil {
			return diag.Errorf("Failed to parse due_date: %s. %v", dueDate.(string), err)
		}
		options.DueDate = parsedDueDate
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab milestone in project %s with title %s", project, title))
	milestone, resp, err := client.Milestones.CreateMilestone(project, options, gitlab.WithContext(ctx))
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("[WARN] failed to create gitlab milestone in project %s with title %s (response %v)", project, title, resp))
		return diag.FromErr(err)
	}
	d.SetId(resourceGitLabProjectMilestoneBuildId(project, milestone.ID))

	updateOptions := gitlab.UpdateMilestoneOptions{}
	if stateEvent, ok := d.GetOk("state"); ok {
		updateOptions.StateEvent = gitlab.Ptr(milestoneStateToStateEvent[stateEvent.(string)])
	}
	if updateOptions != (gitlab.UpdateMilestoneOptions{}) {
		_, _, err := client.Milestones.UpdateMilestone(project, milestone.ID, &updateOptions, gitlab.WithContext(ctx))
		if err != nil {
			return diag.Errorf("Failed to update milestone ID %d in project %s right after creation: %v", milestone.ID, project, err)
		}
	}

	return resourceGitlabProjectMilestoneRead(ctx, d, meta)
}

func resourceGitlabProjectMilestoneRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, milestoneID, err := resourceGitLabProjectMilestoneParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab milestone in project %s with ID %d", project, milestoneID))
	milestone, resp, err := client.Milestones.GetMilestone(project, milestoneID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, fmt.Sprintf("[WARN] recieved 404 for gitlab milestone ID %d in project %s, removing from state", milestoneID, project))
			d.SetId("")
			return nil
		}
		tflog.Warn(ctx, fmt.Sprintf("[WARN] failed to read gitlab milestone ID %d in project %s. Response %v", milestoneID, project, resp))
		return diag.FromErr(err)
	}

	stateMap := gitlabProjectMilestoneToStateMap(project, milestone)
	if err = setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabProjectMilestoneUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, milestoneID, err := resourceGitLabProjectMilestoneParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.UpdateMilestoneOptions{}
	if d.HasChange("title") {
		options.Title = gitlab.Ptr(d.Get("title").(string))
	}
	if d.HasChange("description") {
		options.Description = gitlab.Ptr(d.Get("description").(string))
	}
	if d.HasChange("start_date") {
		startDate := d.Get("start_date").(string)
		parsedStartDate, err := parseISO8601Date(startDate)
		if err != nil {
			return diag.Errorf("Failed to parse due_date: %s. %v", startDate, err)
		}
		options.StartDate = parsedStartDate
	}
	if d.HasChange("due_date") {
		dueDate := d.Get("due_date").(string)
		parsedDueDate, err := parseISO8601Date(dueDate)
		if err != nil {
			return diag.Errorf("Failed to parse due_date: %s. %v", dueDate, err)
		}
		options.DueDate = parsedDueDate
	}
	if d.HasChange("state") {
		options.StateEvent = gitlab.Ptr(milestoneStateToStateEvent[d.Get("state").(string)])
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab milestone in project %s with ID %d", project, milestoneID))
	_, _, err = client.Milestones.UpdateMilestone(project, milestoneID, options, gitlab.WithContext(ctx))
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("[WARN] failed to update gitlab milestone in project %s with ID %d", project, milestoneID))
		return diag.FromErr(err)
	}

	return resourceGitlabProjectMilestoneRead(ctx, d, meta)
}

func resourceGitlabProjectMilestoneDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, milestoneID, err := resourceGitLabProjectMilestoneParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] delete gitlab milestone in project %s with ID %d", project, milestoneID))
	resp, err := client.Milestones.DeleteMilestone(project, milestoneID, gitlab.WithContext(ctx))
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] failed to delete gitlab milestone in project %s with ID %d. Response %v", project, milestoneID, resp))
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitLabProjectMilestoneParseId(id string) (string, int, error) {
	project, milestone, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	milestoneID, err := strconv.Atoi(milestone)
	if err != nil {
		return "", 0, err
	}

	return project, milestoneID, nil
}

func resourceGitLabProjectMilestoneBuildId(project string, milestoneID int) string {
	stringMilestoneID := fmt.Sprintf("%d", milestoneID)
	return utils.BuildTwoPartID(&project, &stringMilestoneID)
}
