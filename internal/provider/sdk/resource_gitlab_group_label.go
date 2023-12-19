package sdk

import (
	"context"
	"log"

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
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabGroupLabelResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabGroupLabelStateUpgradeV0,
				Version: 0,
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

func resourceGitlabGroupLabelBuildId(group string, labelName string) string {
	return utils.BuildTwoPartID(&group, &labelName)
}

func resourceGitlabGroupLabelParseId(id string) (string, string, error) {
	group, labelName, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", "", err
	}
	return group, labelName, nil
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

	log.Printf("[DEBUG] create gitlab group label %s", *options.Name)

	label, _, err := client.GroupLabels.CreateGroupLabel(group, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceGitlabGroupLabelBuildId(group, label.Name))
	return resourceGitlabGroupLabelRead(ctx, d, meta)
}

func resourceGitlabGroupLabelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	group, labelName, err := resourceGitlabGroupLabelParseId(d.Id())
	if err != nil {
		return diag.Errorf("Failed to parse group label id %q: %s", d.Id(), err)
	}

	log.Printf("[DEBUG] read gitlab group label %s/%s", group, labelName)

	label, _, err := client.GroupLabels.GetGroupLabel(group, labelName, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			log.Printf("[DEBUG] failed to read gitlab label %s/%s, removing from state", group, labelName)
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

	log.Printf("[DEBUG] update gitlab group label %s", d.Id())

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

	log.Printf("[DEBUG] Delete gitlab group label %s", d.Id())
	_, err = client.GroupLabels.DeleteGroupLabel(group, labelName, nil, gitlab.WithContext(ctx))
	return diag.FromErr(err)
}
