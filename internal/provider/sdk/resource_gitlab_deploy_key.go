package sdk

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_deploy_key", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_deploy_key`" + ` resource manages the lifecycle of a project deploy key.

-> To enable an already existing deploy key for another project use the ` + "`gitlab_deploy_key_enable`" + ` resource.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/deploy_keys/)`,

		CreateContext: resourceGitlabDeployKeyCreate,
		ReadContext:   resourceGitlabDeployKeyRead,
		DeleteContext: resourceGitlabDeployKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema:        gitlabProjectDeployKeySchema(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabProjectDeployKeyResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabProjectDeployKeyStateUpgradeV0,
				Version: 0,
			},
		},
	}
})

func gitlabProjectDeployKeySchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"deploy_key_id": {
			Description: "The id of the project deploy key.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		"project": {
			Description: "The name or id of the project to add the deploy key to.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"title": {
			Description: "A title to describe the deploy key with.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"key": {
			Description: "The public ssh key body.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				return old == strings.TrimSpace(new)
			},
		},
		"can_push": {
			Description: "Allow this deploy key to be used to push changes to the project. Defaults to `false`.",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			ForceNew:    true,
		},
		"expires_at": {
			Description:      "Expiration date for the deploy key. Does not expire if no value is provided. Expected in RFC3339 format `(2019-03-15T08:00:00Z)`",
			Type:             schema.TypeString,
			Optional:         true,
			ForceNew:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.IsRFC3339Time),
		},
	}
}

// resourceGitlabProjectDeployKeyResourceV0 returns the V0 schema definition.
// From V0-V1 the `id` attribute value format changed from `<deploy-key-id>` to `<project-id>:<deploy-key-id>`,
// which means that the actual schema definition was not impacted and we can just return the
// V1 schema as V0 schema.
func resourceGitlabProjectDeployKeyResourceV0() *schema.Resource {
	return &schema.Resource{Schema: gitlabProjectDeployKeySchema()}
}

// resourceGitlabProjectDeployKeyStateUpgradeV0 performs the state migration from V0 to V1.
func resourceGitlabProjectDeployKeyStateUpgradeV0(ctx context.Context, rawState map[string]any, meta any) (map[string]any, error) {
	project := rawState["project"].(string)
	oldId := rawState["id"].(string)
	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]any{"project": project, "v0-id": oldId})
	rawState["id"] = utils.BuildTwoPartID(&project, &oldId)
	tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]any{"v0-id": oldId, "v1-id": rawState["id"]})
	return rawState, nil
}

func resourceGitlabProjectDeployKeyBuildId(project string, deployKeyId int64) string {
	h := strconv.FormatInt(deployKeyId, 10)
	return utils.BuildTwoPartID(&project, &h)
}

func resourceGitlabProjectDeployKeyParseId(id string) (string, int64, error) {
	project, rawDeployKeyId, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	deployKeyId, err := strconv.ParseInt(rawDeployKeyId, 10, 64)
	if err != nil {
		return "", 0, err
	}

	return project, deployKeyId, nil
}

func resourceGitlabDeployKeyCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	options := &gitlab.AddDeployKeyOptions{
		Title:   gitlab.Ptr(d.Get("title").(string)),
		Key:     gitlab.Ptr(strings.TrimSpace(d.Get("key").(string))),
		CanPush: gitlab.Ptr(d.Get("can_push").(bool)),
	}

	if v, ok := d.GetOk("expires_at"); ok {
		parsedExpiresAt, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return diag.Errorf("failed to parse expires_at: %s. It must be in valid RFC3339 format.", err)
		}
		options.ExpiresAt = gitlab.Ptr(parsedExpiresAt)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab deployment key %s", *options.Title))

	deployKey, _, err := client.DeployKeys.AddDeployKey(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceGitlabProjectDeployKeyBuildId(project, deployKey.ID))

	return resourceGitlabDeployKeyRead(ctx, d, meta)
}

func resourceGitlabDeployKeyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	project, deployKeyID, err := resourceGitlabProjectDeployKeyParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab deploy key %s/%d", project, deployKeyID))

	deployKey, _, err := client.DeployKeys.GetDeployKey(project, deployKeyID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab deploy key not found %s/%d", project, deployKeyID))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("deploy_key_id", deployKey.ID)
	d.Set("project", project)
	d.Set("title", deployKey.Title)
	d.Set("key", deployKey.Key)
	d.Set("can_push", deployKey.CanPush)

	if deployKey.ExpiresAt != nil {
		d.Set("expires_at", deployKey.ExpiresAt.Format(time.RFC3339))
	}

	return nil
}

func resourceGitlabDeployKeyDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	project, deployKeyID, err := resourceGitlabProjectDeployKeyParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab deploy key %s", d.Id()))

	_, err = client.DeployKeys.DeleteDeployKey(project, deployKeyID, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
