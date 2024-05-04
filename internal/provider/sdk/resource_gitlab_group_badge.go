package sdk

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_group_badge", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_group_badge`" + ` resource allows to manage the lifecycle of group badges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/user/project/badges.html#group-badges)`,

		CreateContext: resourceGitlabGroupBadgeCreate,
		ReadContext:   resourceGitlabGroupBadgeRead,
		UpdateContext: resourceGitlabGroupBadgeUpdate,
		DeleteContext: resourceGitlabGroupBadgeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"group": {
				Description: "The id of the group to add the badge to.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"link_url": {
				Description: "The url linked with the badge.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"image_url": {
				Description: "The image url which will be presented on group overview.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "The name of the badge.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"rendered_link_url": {
				Description: "The link_url argument rendered (in case of use of placeholders).",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"rendered_image_url": {
				Description: "The image_url argument rendered (in case of use of placeholders).",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
})

func resourceGitlabGroupBadgeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	groupID := d.Get("group").(string)
	options := &gitlab.AddGroupBadgeOptions{
		LinkURL:  gitlab.Ptr(d.Get("link_url").(string)),
		ImageURL: gitlab.Ptr(d.Get("image_url").(string)),
	}

	if v, ok := d.GetOk("name"); ok {
		options.Name = gitlab.Ptr(v.(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab group variable %s/%s", *options.LinkURL, *options.ImageURL))

	badge, _, err := client.GroupBadges.AddGroupBadge(groupID, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	badgeID := strconv.Itoa(badge.ID)

	d.SetId(utils.BuildTwoPartID(&groupID, &badgeID))

	return resourceGitlabGroupBadgeRead(ctx, d, meta)
}

func resourceGitlabGroupBadgeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	ids := strings.Split(d.Id(), ":")
	groupID := ids[0]
	badgeID, err := strconv.Atoi(ids[1])
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab group badge %s/%d", groupID, badgeID))

	badge, _, err := client.GroupBadges.GetGroupBadge(groupID, badgeID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] group badge %d in group %s doesn't exist anymore, removing from state", badgeID, groupID))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	resourceGitlabGroupBadgeSetToState(d, badge, &groupID)
	return nil
}

func resourceGitlabGroupBadgeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	ids := strings.Split(d.Id(), ":")
	groupID := ids[0]
	badgeID, err := strconv.Atoi(ids[1])
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.EditGroupBadgeOptions{
		LinkURL:  gitlab.Ptr(d.Get("link_url").(string)),
		ImageURL: gitlab.Ptr(d.Get("image_url").(string)),
	}

	if d.HasChange("name") {
		options.Name = gitlab.Ptr(d.Get("name").(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab group badge %s/%d", groupID, badgeID))

	_, _, err = client.GroupBadges.EditGroupBadge(groupID, badgeID, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabGroupBadgeRead(ctx, d, meta)
}

func resourceGitlabGroupBadgeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	ids := strings.Split(d.Id(), ":")
	groupID := ids[0]
	badgeID, err := strconv.Atoi(ids[1])
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab group badge %s/%d", groupID, badgeID))

	_, err = client.GroupBadges.DeleteGroupBadge(groupID, badgeID, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitlabGroupBadgeSetToState(d *schema.ResourceData, badge *gitlab.GroupBadge, groupID *string) {
	d.Set("link_url", badge.LinkURL)
	d.Set("image_url", badge.ImageURL)
	d.Set("name", badge.Name)
	d.Set("rendered_link_url", badge.RenderedLinkURL)
	d.Set("rendered_image_url", badge.RenderedImageURL)
	d.Set("group", groupID)
}
