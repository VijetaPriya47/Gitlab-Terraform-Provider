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

var _ = registerResource("gitlab_label", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_label` + "`" + ` resource allows to manage the lifecycle of a project label.

~> This resource is deprecated. use ` + "`gitlab_project_label`" + `instead!

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/labels.html#project-labels)`,
		DeprecationMessage: "Ths resource is deprecated. Use `gitlab_project_label` instead!",
		CreateContext:      resourceGitlabProjectLabelCreate,
		ReadContext:        resourceGitlabProjectLabelRead,
		UpdateContext:      resourceGitlabProjectLabelUpdate,
		DeleteContext:      resourceGitlabProjectLabelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema:        gitlabProjectLabelSchema(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabProjectLabelResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabProjectLabelStateUpgradeV0,
				Version: 0,
			},
		},
	}
})

var _ = registerResource("gitlab_project_label", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_project_label` + "`" + ` resource allows to manage the lifecycle of a project label.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/labels.html#project-labels)`,

		CreateContext: resourceGitlabProjectLabelCreate,
		ReadContext:   resourceGitlabProjectLabelRead,
		UpdateContext: resourceGitlabProjectLabelUpdate,
		DeleteContext: resourceGitlabProjectLabelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema:        gitlabProjectLabelSchema(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabProjectLabelResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabProjectLabelStateUpgradeV0,
				Version: 0,
			},
		},
	}
})

func gitlabProjectLabelSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"label_id": {
			Description: "The id of the project label.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		"project": {
			Description: "The name or id of the project to add the label to.",
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

// resourceGitlabProjectLabelResourceV0 returns the V0 schema definition.
// From V0-V1 the `id` attribute value format changed from `<project-label-name>` to `<project-id>:<project-label-name>`,
// which means that the actual schema definition was not impacted and we can just return the
// V1 schema as V0 schema.
func resourceGitlabProjectLabelResourceV0() *schema.Resource {
	return &schema.Resource{Schema: gitlabProjectLabelSchema()}
}

// resourceGitlabProjectLabelStateUpgradeV0 performs the state migration from V0 to V1.
func resourceGitlabProjectLabelStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	project := rawState["project"].(string)
	oldId := rawState["id"].(string)
	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]interface{}{"project": project, "v0-id": oldId})
	rawState["id"] = utils.BuildTwoPartID(&project, &oldId)
	tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]interface{}{"v0-id": oldId, "v1-id": rawState["id"]})
	return rawState, nil
}

func resourceGitlabProjectLabelBuildId(project string, labelName string) string {
	return utils.BuildTwoPartID(&project, &labelName)
}

func resourceGitlabProjectLabelParseId(id string) (string, string, error) {
	project, labelName, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", "", err
	}
	return project, labelName, nil
}

func resourceGitlabProjectLabelCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	options := &gitlab.CreateLabelOptions{
		Name:  gitlab.Ptr(d.Get("name").(string)),
		Color: gitlab.Ptr(d.Get("color").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		options.Description = gitlab.Ptr(v.(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab label %s", *options.Name))

	label, _, err := client.Labels.CreateLabel(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceGitlabProjectLabelBuildId(project, label.Name))
	return resourceGitlabProjectLabelRead(ctx, d, meta)
}

func resourceGitlabProjectLabelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, labelName, err := resourceGitlabProjectLabelParseId(d.Id())
	if err != nil {
		return diag.Errorf("Failed to parse project label id %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab label %s/%s", project, labelName))

	label, _, err := client.Labels.GetLabel(project, labelName, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] failed to read gitlab label %s/%s", project, labelName))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("label_id", label.ID)
	d.Set("project", project)
	d.Set("description", label.Description)
	d.Set("color", label.Color)
	d.Set("name", label.Name)
	return nil
}

func resourceGitlabProjectLabelUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, _, err := resourceGitlabProjectLabelParseId(d.Id())
	if err != nil {
		return diag.Errorf("Failed to parse project label id %q: %s", d.Id(), err)
	}
	options := &gitlab.UpdateLabelOptions{
		Name:  gitlab.Ptr(d.Get("name").(string)),
		Color: gitlab.Ptr(d.Get("color").(string)),
	}

	if d.HasChange("description") {
		options.Description = gitlab.Ptr(d.Get("description").(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab label %s", d.Id()))

	_, _, err = client.Labels.UpdateLabel(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabProjectLabelRead(ctx, d, meta)
}

func resourceGitlabProjectLabelDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, labelName, err := resourceGitlabProjectLabelParseId(d.Id())
	if err != nil {
		return diag.Errorf("Failed to parse project label id %q: %s", d.Id(), err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab label %s", d.Id()))
	_, err = client.Labels.DeleteLabel(project, labelName, nil, gitlab.WithContext(ctx))
	return diag.FromErr(err)
}
