package sdk

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var issueStateToStateEvent = map[string]string{
	"opened": "reopen",
	"closed": "close",
}

var _ = registerResource("gitlab_project_issue", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_project_issue`" + ` resource allows to manage the lifecycle of an issue within a project.

-> During a terraform destroy this resource will close the issue. Set the delete_on_destroy flag to true to delete the issue instead of closing it.

~> **Experimental** While the base functionality of this resource works, it may be subject to minor change.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/issues/)
		`,

		CreateContext: resourceGitlabProjectIssueCreate,
		ReadContext:   resourceGitlabProjectIssueRead,
		UpdateContext: resourceGitlabProjectIssueUpdate,
		DeleteContext: resourceGitlabProjectIssueDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: constructSchema(
			gitlabProjectIssueGetSchema(),
			map[string]*schema.Schema{
				"delete_on_destroy": {
					Description: "Whether the issue is deleted instead of closed during destroy.",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
			},
		),
	}
})

func resourceGitlabProjectIssueCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)

	options := &gitlab.CreateIssueOptions{
		Title: gitlab.Ptr(d.Get("title").(string)),
	}
	if iid, ok := d.GetOk("iid"); ok {
		options.IID = gitlab.Ptr(iid.(int))
	}
	if assigneeIDs, ok := d.GetOk("assignee_ids"); ok {
		options.AssigneeIDs = intSetToIntSlice(assigneeIDs.(*schema.Set))
	}
	if confidential, ok := d.GetOk("confidential"); ok {
		options.Confidential = gitlab.Ptr(confidential.(bool))
	}
	if createdAt, ok := d.GetOk("created_at"); ok {
		parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt.(string))
		if err != nil {
			return diag.Errorf("failed to parse created_at: %s. It must be in valid RFC3339 format.", err)
		}
		options.CreatedAt = gitlab.Ptr(parsedCreatedAt)
	}
	if description, ok := d.GetOk("description"); ok {
		options.Description = gitlab.Ptr(description.(string))
	}
	if discussionToResolve, ok := d.GetOk("discussion_to_resolve"); ok {
		options.DiscussionToResolve = gitlab.Ptr(discussionToResolve.(string))
	}
	if dueDate, ok := d.GetOk("due_date"); ok {
		parsedDueDate, err := parseISO8601Date(dueDate.(string))
		if err != nil {
			return diag.Errorf("failed to parse due_date: %s. %v", dueDate.(string), err)
		}
		options.DueDate = parsedDueDate
	}
	if issueType, ok := d.GetOk("issue_type"); ok {
		options.IssueType = gitlab.Ptr(issueType.(string))
	}
	if labels, ok := d.GetOk("labels"); ok {
		gitlabLabels := gitlab.LabelOptions(*stringSetToStringSlice(labels.(*schema.Set)))
		options.Labels = &gitlabLabels
	}
	if mergeRequestToResolveDiscussionsOf, ok := d.GetOk("merge_request_to_resolve_discussions_of"); ok {
		options.MergeRequestToResolveDiscussionsOf = gitlab.Ptr(mergeRequestToResolveDiscussionsOf.(int))
	}
	if milestoneID, ok := d.GetOk("milestone_id"); ok {
		options.MilestoneID = gitlab.Ptr(milestoneID.(int))
	}
	if weight, ok := d.GetOk("weight"); ok {
		options.Weight = gitlab.Ptr(weight.(int))
	}

	issue, _, err := client.Issues.CreateIssue(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(resourceGitLabProjectIssueBuildId(project, issue.IID))

	updateOptions := gitlab.UpdateIssueOptions{}
	if discussionLocked, ok := d.GetOk("discussion_locked"); ok {
		updateOptions.DiscussionLocked = gitlab.Ptr(discussionLocked.(bool))
	}
	if stateEvent, ok := d.GetOk("state"); ok {
		updateOptions.StateEvent = gitlab.Ptr(issueStateToStateEvent[stateEvent.(string)])
	}
	if updateOptions != (gitlab.UpdateIssueOptions{}) {
		_, _, err := client.Issues.UpdateIssue(project, issue.IID, &updateOptions, gitlab.WithContext(ctx))
		if err != nil {
			return diag.Errorf("failed to update issue %d in project %s right after creation: %v", issue.IID, project, err)
		}
	}

	return resourceGitlabProjectIssueRead(ctx, d, meta)
}

func resourceGitlabProjectIssueRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, issueIID, err := resourceGitLabProjectIssueParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	issue, _, err := client.Issues.GetIssue(project, issueIID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, fmt.Sprintf("[WARN] issue %d in project %s not found, removing from state", issueIID, project))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	stateMap := gitlabProjectIssueToStateMap(project, issue)
	if err = setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabProjectIssueUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, issueIID, err := resourceGitLabProjectIssueParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.UpdateIssueOptions{}
	if d.HasChange("title") {
		options.Title = gitlab.Ptr(d.Get("title").(string))
	}
	if d.HasChange("assignee_ids") {
		options.AssigneeIDs = intSetToIntSlice(d.Get("assignee_ids").(*schema.Set))
	}
	if d.HasChange("confidential") {
		options.Confidential = gitlab.Ptr(d.Get("confidential").(bool))
	}
	if d.HasChange("description") {
		options.Description = gitlab.Ptr(d.Get("description").(string))
	}
	if d.HasChange("due_date") {
		dueDate := d.Get("due_date").(string)

		parsedDueDate, err := parseISO8601Date(dueDate)
		if err != nil {
			return diag.Errorf("failed to parse due_date: %s. %v", dueDate, err)
		}
		options.DueDate = parsedDueDate
	}
	if d.HasChange("issue_type") {
		options.IssueType = gitlab.Ptr(d.Get("issue_type").(string))
	}
	if d.HasChange("labels") {
		gitlabLabels := gitlab.LabelOptions(*stringSetToStringSlice(d.Get("labels").(*schema.Set)))
		options.Labels = &gitlabLabels
	}
	if d.HasChange("milestone_id") {
		options.MilestoneID = gitlab.Ptr(d.Get("milestone_id").(int))
	}
	if d.HasChange("weight") {
		options.Weight = gitlab.Ptr(d.Get("weight").(int))
	}
	if d.HasChange("state") {
		options.StateEvent = gitlab.Ptr(issueStateToStateEvent[d.Get("state").(string)])
	}
	if d.HasChange("discussion_locked") {
		options.DiscussionLocked = gitlab.Ptr(d.Get("discussion_locked").(bool))
	}

	_, _, err = client.Issues.UpdateIssue(project, issueIID, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabProjectIssueRead(ctx, d, meta)
}

func resourceGitlabProjectIssueDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, issueIID, err := resourceGitLabProjectIssueParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	deleteOnDestroy := d.Get("delete_on_destroy").(bool)

	if deleteOnDestroy {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Deleting issue %d in project %s for destroy", issueIID, project))
		resp, err := client.Issues.DeleteIssue(project, issueIID, gitlab.WithContext(ctx))
		if err != nil {
			return diag.Errorf("%s failed to delete issue %d in project %s: (%s) %v", d.Id(), issueIID, project, resp.Status, err)
		}
	} else {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Closing issue %d in project %s for destroy", issueIID, project))
		_, resp, err := client.Issues.UpdateIssue(project, issueIID, &gitlab.UpdateIssueOptions{StateEvent: gitlab.Ptr("close")}, gitlab.WithContext(ctx))
		if err != nil {
			return diag.Errorf("%s failed to delete issue %d in project %s: (%s) %v", d.Id(), issueIID, project, resp.Status, err)
		}
	}

	return nil
}

func resourceGitLabProjectIssueParseId(id string) (string, int, error) {
	project, issue, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	issueIID, err := strconv.Atoi(issue)
	if err != nil {
		return "", 0, err
	}

	return project, issueIID, nil
}

func resourceGitLabProjectIssueBuildId(project string, issueIID int) string {
	stringIssueIID := fmt.Sprintf("%d", issueIID)
	return utils.BuildTwoPartID(&project, &stringIssueIID)
}
