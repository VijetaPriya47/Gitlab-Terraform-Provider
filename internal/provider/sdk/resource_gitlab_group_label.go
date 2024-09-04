package sdk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_group_label", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_group_label`" + ` resource allows to manage the lifecycle of labels within a group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/user/project/labels.html#group-labels)`,

		CreateContext: resourceGitlabGroupLabelCreate,
		ReadContext:   resourceGitlabGroupLabelRead,
		UpdateContext: resourceGitlabGroupLabelUpdate,
		DeleteContext: resourceGitlabGroupLabelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema:        gitlabGroupLabelSchema(),
		SchemaVersion: 2,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabGroupLabelResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabGroupLabelStateUpgradeV0,
				Version: 0,
			},
			{
				Type:    resourceGitlabGroupLabelResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabGroupLabelStateUpgradeV1,
				Version: 1,
			},
		},
	}
})

func gitlabGroupLabelSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"label_id": {
			Description: "The id of the group label.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		"group": {
			Description: "The name or id of the group to add the label to.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"name": {
			Description: "The name of the label.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"color": {
			Description: "The color of the label given in 6-digit hex notation with leading '#' sign (e.g. #FFAABB) or one of the [CSS color names](https://developer.mozilla.org/en-US/docs/Web/CSS/color_value#Color_keywords).",
			Type:        schema.TypeString,
			Required:    true,
		},
		"description": {
			Description: "The description of the label.",
			Type:        schema.TypeString,
			Optional:    true,
		},
	}
}

// resourceGitlabGroupLabelResourceV0 returns the V0 schema definition.
// From V0-V1 the `id` attribute value format changed from `<group-label-name>` to `<group-id>:<group-label-name>`,
// which means that the actual schema definition was not impacted and we can just return the
// V1 schema as V0 schema.
func resourceGitlabGroupLabelResourceV0() *schema.Resource {
	return &schema.Resource{Schema: gitlabGroupLabelSchema()}
}

// resourceGitlabGroupLabelStateUpgradeV0 performs the state migration from V0 to V1.
func resourceGitlabGroupLabelStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	group := rawState["group"].(string)
	oldId := rawState["id"].(string)
	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]interface{}{"group": group, "v0-id": oldId})
	rawState["id"] = utils.BuildTwoPartID(&group, &oldId)
	tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]interface{}{"v0-id": oldId, "v1-id": rawState["id"]})
	return rawState, nil
}

func resourceGitlabGroupLabelStateUpgradeV1(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	group := rawState["group"].(string)
	oldId := rawState["id"].(string)
	// Check if label_id is present in rawState
	labelId, ok := rawState["label_id"].(string)

	tflog.Debug(ctx, "attempting state migration from V1 to V2 - changing the `id` attribute format", map[string]interface{}{"group": group, "label_id": labelId, "v1-id": oldId})

	if !ok {
		tflog.Debug(ctx, "label_id is not present in the raw state, retrieving the information via API to build the old label_id for migration", map[string]interface{}{"group": group, "v1-id": oldId})

		client := meta.(*gitlab.Client)

		// Derive or fetch label_id if not present
		var err error
		group, labelName, err := resourceGitlabGroupLabelParseId(oldId)
		if err != nil {
			return nil, fmt.Errorf("failed to parse group label id %q: %s", oldId, err)
		}

		label, _, err := client.GroupLabels.GetGroupLabel(group, labelName, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch label: %v", err)
		}
		labelId = strconv.Itoa(label.ID)
	}

	rawState["id"] = utils.BuildTwoPartID(&group, &labelId)
	tflog.Debug(ctx, "migrated `id` attribute for V1 to V2", map[string]interface{}{"v1-id": oldId, "v2-id": rawState["id"]})
	return rawState, nil
}

func resourceGitlabGroupLabelBuildId(group string, labelId string) string {
	return utils.BuildTwoPartID(&group, &labelId)
}

func resourceGitlabGroupLabelParseId(id string) (string, string, error) {
	group, labelId, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", "", err
	}
	return group, labelId, nil
}

func resourceGitlabGroupLabelCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	group := d.Get("group").(string)
	options := &gitlab.CreateGroupLabelOptions{
		Name:  gitlab.Ptr(d.Get("name").(string)),
		Color: gitlab.Ptr(d.Get("color").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		options.Description = gitlab.Ptr(v.(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab group label %s", *options.Name))

	label, _, err := client.GroupLabels.CreateGroupLabel(group, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceGitlabGroupLabelBuildId(group, strconv.Itoa(label.ID)))
	return resourceGitlabGroupLabelRead(ctx, d, meta)
}

func resourceGitlabGroupLabelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	group, labelId, err := resourceGitlabGroupLabelParseId(d.Id())
	if err != nil {
		return diag.Errorf("Failed to parse group label id %q: %s", d.Id(), err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab group label %s/%s", group, labelId))

	label, _, err := client.GroupLabels.GetGroupLabel(group, labelId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] failed to read gitlab label %s/%s, removing from state", group, labelId))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	d.Set("group", group)
	d.Set("label_id", label.ID)
	d.Set("description", label.Description)
	d.Set("color", label.Color)
	d.Set("name", label.Name)
	return nil
}

func resourceGitlabGroupLabelUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	group, _, err := resourceGitlabGroupLabelParseId(d.Id())
	if err != nil {
		return diag.Errorf("Failed to parse group label id %q: %s", d.Id(), err)
	}

	options := &gitlab.UpdateGroupLabelOptions{
		Name:  gitlab.Ptr(d.Get("name").(string)),
		Color: gitlab.Ptr(d.Get("color").(string)),
	}

	if d.HasChange("description") {
		options.Description = gitlab.Ptr(d.Get("description").(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab group label %s", d.Id()))

	_, _, err = client.GroupLabels.UpdateGroupLabel(group, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabGroupLabelRead(ctx, d, meta)
}

func resourceGitlabGroupLabelDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	group, labelName, err := resourceGitlabGroupLabelParseId(d.Id())
	if err != nil {
		return diag.Errorf("Failed to parse group label id %q: %s", d.Id(), err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab group label %s", d.Id()))
	_, err = client.GroupLabels.DeleteGroupLabel(group, labelName, nil, gitlab.WithContext(ctx))
	return diag.FromErr(err)
}
