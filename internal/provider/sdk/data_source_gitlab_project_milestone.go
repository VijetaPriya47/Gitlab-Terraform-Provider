package sdk

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var _ = registerDataSource("gitlab_project_milestone", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_project_milestone`" + ` data source allows get details of a project milestone.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/milestones/)`,

		ReadContext: dataSourceGitlabProjectMilestoneRead,
		Schema:      datasourceSchemaFromResourceSchema(gitlabProjectMilestoneGetSchema(), []string{"project", "milestone_id"}, nil),
	}
})

func dataSourceGitlabProjectMilestoneRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	milestoneID := d.Get("milestone_id").(int)

	milestone, _, err := client.Milestones.GetMilestone(project, milestoneID, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(resourceGitLabProjectMilestoneBuildId(project, milestone.ID))
	stateMap := gitlabProjectMilestoneToStateMap(project, milestone)

	if err := setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
