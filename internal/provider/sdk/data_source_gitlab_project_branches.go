package sdk

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/xanzy/go-gitlab"
)

var _ = registerDataSource("gitlab_project_branches", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_project_branches`" + ` data source allows details of the branches of a given project to be retrieved.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/branches.html#list-repository-branches)`,

		ReadContext: dataSourceGitlabProjectBranchesRead,
		Schema: map[string]*schema.Schema{
			"project": {
				Description:  "ID or URL-encoded path of the project owned by the authenticated user.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"branches": {
				Description: "The list of branches of the project, as defined below.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "The name of the branch.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"merged": {
							Description: "Bool, true if the branch has been merged into it's parent.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"protected": {
							Description: "Bool, true if branch has branch protection.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"default": {
							Description: "Bool, true if branch is the default branch for the project.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"developers_can_push": {
							Description: "Bool, true if developer level access allows git push.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"developers_can_merge": {
							Description: "Bool, true if developer level access allows to merge branch.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"can_push": {
							Description: "Bool, true if you can push to the branch.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"web_url": {
							Description: "URL that can be used to find the branch in a browser.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"commit": {
							Description: "The commit associated with this branch.",
							Type:        schema.TypeSet,
							Computed:    true,
							Set:         schema.HashResource(commitSchema),
							Elem:        commitSchema,
						},
					},
				},
			},
		},
	}
})

func dataSourceGitlabProjectBranchesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	tflog.Info(ctx, "[INFO] Reading Gitlab branches")

	project := d.Get("project").(string)

	options := &gitlab.ListBranchesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}

	var allBranches []*gitlab.Branch
	for options.Page != 0 {
		branches, resp, err := client.Branches.ListBranches(project, options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
		allBranches = append(allBranches, branches...)
		options.Page = resp.NextPage
	}

	d.SetId(project)
	if err := d.Set("branches", flattenBranches(allBranches)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func flattenBranches(branches []*gitlab.Branch) (values []map[string]interface{}) {
	for _, branch := range branches {
		values = append(values, map[string]interface{}{
			"name":                 branch.Name,
			"merged":               branch.Merged,
			"protected":            branch.Protected,
			"default":              branch.Default,
			"developers_can_push":  branch.DevelopersCanPush,
			"developers_can_merge": branch.DevelopersCanMerge,
			"can_push":             branch.CanPush,
			"web_url":              branch.WebURL,
			"commit":               flattenCommit(branch.Commit),
		})
	}
	return values
}
