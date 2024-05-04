package sdk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// https://docs.gitlab.com/ee/api/groups.html#share-groups-with-groups

var _ = registerResource("gitlab_group_share_group", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_group_share_group` + "`" + ` resource allows to manage the lifecycle of group shared with another group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/groups.html#share-groups-with-groups)`,

		CreateContext: resourceGitlabGroupShareGroupCreate,
		ReadContext:   resourceGitlabGroupShareGroupRead,
		DeleteContext: resourceGitlabGroupShareGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"group_id": {
				Description: "The id of the main group to be shared.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"share_group_id": {
				Description: "The id of the additional group with which the main group will be shared.",
				Type:        schema.TypeInt,
				ForceNew:    true,
				Required:    true,
			},
			"group_access": {
				Description:      fmt.Sprintf("The access level to grant the group. Valid values are: %s", utils.RenderValueListForDocs(api.ValidGroupAccessLevelNames)),
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(api.ValidGroupAccessLevelNames, false)),
				ForceNew:         true,
				Required:         true,
			},
			"expires_at": {
				Description:  "Share expiration date. Format: `YYYY-MM-DD`",
				Type:         schema.TypeString,
				ValidateFunc: validateDateFunc,
				ForceNew:     true,
				Optional:     true,
			},
		},
	}
})

func resourceGitlabGroupShareGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	groupId := d.Get("group_id").(string)
	shareGroupId := d.Get("share_group_id").(int)
	groupAccess := api.AccessLevelNameToValue[d.Get("group_access").(string)]
	options := &gitlab.ShareWithGroupOptions{
		GroupID:     &shareGroupId,
		GroupAccess: &groupAccess,
		ExpiresAt:   gitlab.Ptr(d.Get("expires_at").(string)),
	}

	client := meta.(*gitlab.Client)
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab group share for %d in %s", shareGroupId, groupId))

	_, _, err := client.GroupMembers.ShareWithGroup(groupId, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	shareGroupIdString := strconv.Itoa(shareGroupId)
	d.SetId(utils.BuildTwoPartID(&groupId, &shareGroupIdString))

	return resourceGitlabGroupShareGroupRead(ctx, d, meta)
}

func resourceGitlabGroupShareGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	id := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab shared groups %s", id))

	groupId, sharedGroupId, err := groupIdsFromId(id)
	if err != nil {
		return diag.FromErr(err)
	}

	// Query main group
	group, _, err := client.Groups.GetGroup(groupId, nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab group %s not found so removing from state", groupId))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// Find shared group data from queried group
	for _, sharedGroup := range group.SharedWithGroups {
		if sharedGroupId == sharedGroup.GroupID {
			convertedAccessLevel := gitlab.AccessLevelValue(sharedGroup.GroupAccessLevel)

			d.Set("group_id", groupId)
			d.Set("share_group_id", sharedGroup.GroupID)
			d.Set("group_access", api.AccessLevelValueToName[convertedAccessLevel])

			if sharedGroup.ExpiresAt == nil {
				d.Set("expires_at", "")
			} else {
				d.Set("expires_at", sharedGroup.ExpiresAt.String())
			}

			return nil
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab shared group %s not found so removing from state", id))
	d.SetId("")
	return nil
}

func resourceGitlabGroupShareGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	id := d.Id()

	groupId, sharedGroupId, err := groupIdsFromId(id)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab share group %d for %s", sharedGroupId, groupId))

	_, err = client.GroupMembers.DeleteShareWithGroup(groupId, sharedGroupId, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func groupIdsFromId(id string) (string, int, error) {
	groupId, sharedGroupIdString, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, fmt.Errorf("error parsing ID: %s", id)
	}

	sharedGroupId, err := strconv.Atoi(sharedGroupIdString)
	if err != nil {
		return "", 0, fmt.Errorf("can not determine shared group id: %s", sharedGroupIdString)
	}

	return groupId, sharedGroupId, nil
}
