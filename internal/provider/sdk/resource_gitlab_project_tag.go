package sdk

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_project_tag", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_project_tag`" + ` resource allows to manage the lifecycle of a tag in a project.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/ee/api/tags.html)`,

		CreateContext: resourceGitlabProjectTagCreate,
		ReadContext:   resourceGitlabProjectTagRead,
		DeleteContext: resourceGitlabProjectTagDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: constructSchema(
			gitlabProjectTagGetSchema(),
			map[string]*schema.Schema{
				"project": {
					Description: "The ID or URL-encoded path of the project owned by the authenticated user.",
					Type:        schema.TypeString,
					ForceNew:    true,
					Required:    true,
				},
				"ref": {
					Description: "Create tag using commit SHA, another tag name, or branch name. This attribute is not available for imported resources.",
					Type:        schema.TypeString,
					ForceNew:    true,
					Required:    true,
				},
			},
		),
	}
})

func resourceGitlabProjectTagCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	name := d.Get("name").(string)
	project := d.Get("project").(string)
	ref := d.Get("ref").(string)
	message := d.Get("message").(string)
	tagOptions := &gitlab.CreateTagOptions{
		TagName: &name, Ref: &ref, Message: &message,
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab tag %s/%s with ref %s", project, name, ref))
	_, resp, err := client.Tags.CreateTag(project, tagOptions, gitlab.WithContext(ctx))
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] failed to create gitlab tag %s/%s response %v", project, name, resp))
		return diag.FromErr(err)
	}
	d.SetId(utils.BuildTwoPartID(&project, &name))
	d.Set("ref", ref)
	return resourceGitlabProjectTagRead(ctx, d, meta)
}

func resourceGitlabProjectTagRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, name, err := utils.ParseTwoPartID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab tag %s/%s", project, name))
	tag, resp, err := client.Tags.GetTag(project, name, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] recieved 404 for gitlab tag %s/%s, removing from state", project, name))
			d.SetId("")
			return diag.FromErr(err)
		}
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] failed to read gitlab tag %s/%s response %v", project, name, resp))
		return diag.FromErr(err)
	}
	d.Set("name", tag.Name)
	d.Set("project", project)
	d.Set("message", tag.Message)
	d.Set("protected", tag.Protected)
	d.Set("target", tag.Target)
	releaseNote := flattenReleaseNote(tag.Release)
	if err := d.Set("release", releaseNote); err != nil {
		return diag.FromErr(err)
	}
	commit := flattenCommit(tag.Commit)
	if err := d.Set("commit", commit); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabProjectTagDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, name, err := utils.ParseTwoPartID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] delete gitlab tag %s/%s", project, name))
	resp, err := client.Tags.DeleteTag(project, name, gitlab.WithContext(ctx))
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] failed to delete gitlab tag %s/%s response %v", project, name, resp))
		return diag.FromErr(err)
	}
	return nil
}

func flattenReleaseNote(releaseNote *gitlab.ReleaseNote) (values []map[string]interface{}) {
	if releaseNote == nil {
		return []map[string]interface{}{}
	}
	return []map[string]interface{}{
		{
			"tag_name":    releaseNote.TagName,
			"description": releaseNote.Description,
		},
	}
}
