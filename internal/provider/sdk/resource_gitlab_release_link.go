package sdk

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var _ = registerResource("gitlab_release_link", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_release_link`" + ` resource allows to manage the lifecycle of a release link.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/releases/links/)`,

		CreateContext: resourceGitlabReleaseLinkCreate,
		ReadContext:   resourceGitlabReleaseLinkRead,
		UpdateContext: resourceGitlabReleaseLinkUpdate,
		DeleteContext: resourceGitlabReleaseLinkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: gitlabReleaseLinkGetSchema(),
	}
})

func resourceGitlabReleaseLinkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	tagName := d.Get("tag_name").(string)
	name := d.Get("name").(string)
	url := d.Get("url").(string)

	options := &gitlab.CreateReleaseLinkOptions{
		Name: gitlab.Ptr(name),
		URL:  gitlab.Ptr(url),
	}
	if filePath, ok := d.GetOk("filepath"); ok {
		options.FilePath = gitlab.Ptr(filePath.(string))
	}
	if linkType, ok := d.GetOk("link_type"); ok {
		linkTypeValue := gitlab.LinkTypeValue(linkType.(string))
		options.LinkType = &linkTypeValue
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create release link project/tagName/name: %s/%s/%s", project, tagName, name))
	releaseLink, resp, err := client.ReleaseLinks.CreateReleaseLink(project, tagName, options, gitlab.WithContext(ctx))
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("[WARN] failed to create release link project/tagName/name: %s/%s/%s (response %v)", project, tagName, name, resp))
		return diag.FromErr(err)
	}
	d.SetId(resourceGitLabReleaseLinkBuildId(project, tagName, releaseLink.ID))

	return resourceGitlabReleaseLinkRead(ctx, d, meta)
}

func resourceGitlabReleaseLinkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, tagName, linkID, err := resourceGitLabReleaseLinkParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read release link project/tagName/linkID: %s/%s/%d", project, tagName, linkID))
	releaseLink, resp, err := client.ReleaseLinks.GetReleaseLink(project, tagName, linkID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, fmt.Sprintf("[WARN] recieved 404 for release link project/tagName/linkID: %s/%s/%d. Removing from state", project, tagName, linkID))
			d.SetId("")
			return nil
		}
		tflog.Warn(ctx, fmt.Sprintf("[WARN] failed to read release link project/tagName/linkID: %s/%s/%d. Response %v", project, tagName, linkID, resp))
		return diag.FromErr(err)
	}

	stateMap := gitlabReleaseLinkToStateMap(project, tagName, releaseLink)
	if err = setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabReleaseLinkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, tagName, linkID, err := resourceGitLabReleaseLinkParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.UpdateReleaseLinkOptions{}
	if d.HasChange("name") {
		options.Name = gitlab.Ptr(d.Get("name").(string))
	}
	if d.HasChange("url") {
		options.URL = gitlab.Ptr(d.Get("url").(string))
	}
	if d.HasChange("filepath") {
		options.FilePath = gitlab.Ptr(d.Get("filepath").(string))
	}
	if d.HasChange("link_type") {
		linkTypeValue := gitlab.LinkTypeValue(d.Get("link_type").(string))
		options.LinkType = &linkTypeValue
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update release link project/tagName/linkID: %s/%s/%d", project, tagName, linkID))
	_, _, err = client.ReleaseLinks.UpdateReleaseLink(project, tagName, linkID, options, gitlab.WithContext(ctx))
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("[WARN] failed to update release link project/tagName/linkID: %s/%s/%d", project, tagName, linkID))
		return diag.FromErr(err)
	}

	return resourceGitlabReleaseLinkRead(ctx, d, meta)
}

func resourceGitlabReleaseLinkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, tagName, linkID, err := resourceGitLabReleaseLinkParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] delete release link project/tagName/linkID: %s/%s/%d", project, tagName, linkID))
	_, resp, err := client.ReleaseLinks.DeleteReleaseLink(project, tagName, linkID, gitlab.WithContext(ctx))
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] failed to delete release link project/tagName/linkID: %s/%s/%d. Response %v", project, tagName, linkID, resp))
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitLabReleaseLinkParseId(id string) (string, string, int, error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("unexpected ID format (%q). Expected project:tagName:linkID", id)
	}

	linkID, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", "", 0, err
	}

	return parts[0], parts[1], linkID, nil
}

func resourceGitLabReleaseLinkBuildId(project string, tagName string, linkID int) string {
	id := fmt.Sprintf("%s:%s:%d", project, tagName, linkID)
	return id
}
