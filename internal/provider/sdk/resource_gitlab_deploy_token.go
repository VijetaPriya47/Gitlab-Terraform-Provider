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
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var _ = registerResource("gitlab_deploy_token", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_deploy_token`" + ` resource allows to manage the lifecycle of group and project deploy tokens.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/deploy_tokens.html)`,

		CreateContext: resourceGitlabDeployTokenCreate,
		ReadContext:   resourceGitlabDeployTokenRead,
		DeleteContext: resourceGitlabDeployTokenDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema:        gitlabDeployTokenSchema(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabDeployTokenResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabDeployTokenStateUpgradeV0,
				Version: 0,
			},
		},
	}
})

func gitlabDeployTokenSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"deploy_token_id": {
			Description: "The id of the deploy token.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		"project": {
			Description:  "The name or id of the project to add the deploy token to.",
			Type:         schema.TypeString,
			Optional:     true,
			ExactlyOneOf: []string{"project", "group"},
			ForceNew:     true,
		},
		"group": {
			Description:  "The name or id of the group to add the deploy token to.",
			Type:         schema.TypeString,
			Optional:     true,
			ExactlyOneOf: []string{"project", "group"},
			ForceNew:     true,
		},
		"name": {
			Description: "A name to describe the deploy token with.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"username": {
			Description: "A username for the deploy token. Default is `gitlab+deploy-token-{n}`.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
		},
		"expires_at": {
			Description:      "Time the token will expire it, RFC3339 format. Will not expire per default.",
			Type:             schema.TypeString,
			Optional:         true,
			ValidateFunc:     validation.IsRFC3339Time,
			DiffSuppressFunc: expiresAtSuppressFunc,
			ForceNew:         true,
		},
		"scopes": {
			Description: "Valid values: `read_repository`, `read_registry`, `read_package_registry`, `write_registry`, `write_package_registry`.",
			Type:        schema.TypeSet,
			Required:    true,
			ForceNew:    true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validation.StringInSlice(
					[]string{
						"read_registry",
						"read_repository",
						"read_package_registry",
						"write_registry",
						"write_package_registry",
					}, false),
			},
		},

		"token": {
			Description: "The secret token. This is only populated when creating a new deploy token. **Note**: The token is not available for imported resources.",
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
		},
	}
}

func expiresAtSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	oldDate, oldDateErr := time.Parse(time.RFC3339, old)
	newDate, newDateErr := time.Parse(time.RFC3339, new)
	if oldDateErr != nil || newDateErr != nil {
		return false
	}
	return oldDate == newDate
}

// resourceGitlabDeployTokenResourceV0 returns the V0 schema definition.
// From V0-V1 the `id` attribute value format changed from `<deploy-token-id>` to `<project-id>:<deploy-token-id>`,
// which means that the actual schema definition was not impacted and we can just return the
// V1 schema as V0 schema.
func resourceGitlabDeployTokenResourceV0() *schema.Resource {
	return &schema.Resource{Schema: gitlabDeployTokenSchema()}
}

// resourceGitlabDeployTokenStateUpgradeV0 performs the state migration from V0 to V1.
func resourceGitlabDeployTokenStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	var deployTokenType string
	var typeId string
	if project, isProject := rawState["project"]; isProject && project != nil && project != "" {
		deployTokenType = "project"
		typeId = project.(string)
	} else if group, isGroup := rawState["group"]; isGroup && group != nil && group != "" {
		deployTokenType = "group"
		typeId = group.(string)
	} else {
		return nil, fmt.Errorf("cannot migrate state from V0 to V1 because neither `project` nor `group` attribute is in state, so cannot determine the deploy token type")
	}

	oldId := rawState["id"].(string)

	deployTokenId, err := strconv.Atoi(oldId)
	if err != nil {
		return nil, fmt.Errorf("cannot migrate state from V0 to V1 because id %q cannot be converted into an integer: %w", oldId, err)
	}

	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]interface{}{"deployTokenType": deployTokenType, "typeId": typeId, "v0-id": oldId})
	rawState["id"] = resourceGitlabDeployTokenBuildId(deployTokenType, typeId, deployTokenId)
	tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]interface{}{"v0-id": oldId, "v1-id": rawState["id"]})
	return rawState, nil
}

func resourceGitlabDeployTokenBuildId(deployTokenType string, typeId string, deployTokenId int) string {
	return fmt.Sprintf("%s:%s:%d", deployTokenType, typeId, deployTokenId)
}

func resourceGitlabDeployTokenParseId(id string) (string, string, int, error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("unexpected ID format (%q). Expected deployKeyType:typeId:key", id)
	}

	deployTokenId, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", "", 0, err
	}

	return parts[0], parts[1], deployTokenId, nil
}

func resourceGitlabDeployTokenCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, isProject := d.GetOk("project")
	group, isGroup := d.GetOk("group")

	var expiresAt *time.Time
	var err error

	if exp, ok := d.GetOk("expires_at"); ok {
		parsedExpiresAt, err := time.Parse(time.RFC3339, exp.(string))
		expiresAt = &parsedExpiresAt
		if err != nil {
			return diag.Errorf("Invalid expires_at date: %v", err)
		}
	}

	scopes := stringSetToStringSlice(d.Get("scopes").(*schema.Set))

	var deployToken *gitlab.DeployToken

	var deployTokenType string
	var typeId string
	if isProject {
		deployTokenType = "project"
		typeId = project.(string)
		options := &gitlab.CreateProjectDeployTokenOptions{
			Name:      gitlab.Ptr(d.Get("name").(string)),
			Username:  gitlab.Ptr(d.Get("username").(string)),
			ExpiresAt: expiresAt,
			Scopes:    scopes,
		}

		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Create GitLab deploy token %s in project %s", *options.Name, project.(string)))

		deployToken, _, err = client.DeployTokens.CreateProjectDeployToken(project, options, gitlab.WithContext(ctx))
	} else if isGroup {
		deployTokenType = "group"
		typeId = group.(string)
		options := &gitlab.CreateGroupDeployTokenOptions{
			Name:      gitlab.Ptr(d.Get("name").(string)),
			Username:  gitlab.Ptr(d.Get("username").(string)),
			ExpiresAt: expiresAt,
			Scopes:    scopes,
		}

		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Create GitLab deploy token %s in group %s", *options.Name, group.(string)))

		deployToken, _, err = client.DeployTokens.CreateGroupDeployToken(group, options, gitlab.WithContext(ctx))
	}

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceGitlabDeployTokenBuildId(deployTokenType, typeId, deployToken.ID))

	// Token is only available on creation
	d.Set("token", deployToken.Token)

	return resourceGitlabDeployTokenRead(ctx, d, meta)
}

func resourceGitlabDeployTokenRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	deployTokenType, typeId, deployTokenId, err := resourceGitlabDeployTokenParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var deployToken *gitlab.DeployToken
	switch deployTokenType {
	case "project":
		d.Set("project", typeId)
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Read GitLab deploy token %d in project %s", deployTokenId, typeId))
		deployToken, _, err = client.DeployTokens.GetProjectDeployToken(typeId, deployTokenId, gitlab.WithContext(ctx))
	case "group":
		d.Set("group", typeId)
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Read GitLab deploy token %d in group %s", deployTokenId, typeId))
		deployToken, _, err = client.DeployTokens.GetGroupDeployToken(typeId, deployTokenId, gitlab.WithContext(ctx))
	}

	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] GitLab deploy token %d in was not found, removing from state", deployTokenId))
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	d.Set("deploy_token_id", deployToken.ID)
	d.Set("name", deployToken.Name)
	d.Set("username", deployToken.Username)

	if deployToken.ExpiresAt != nil {
		d.Set("expires_at", deployToken.ExpiresAt.Format(time.RFC3339))
	}

	if err := d.Set("scopes", deployToken.Scopes); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitlabDeployTokenDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	deployTokenType, typeId, deployTokenId, err := resourceGitlabDeployTokenParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	switch deployTokenType {
	case "project":
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete GitLab deploy token %d in project %s", deployTokenId, typeId))
		_, err = client.DeployTokens.DeleteProjectDeployToken(typeId, deployTokenId, gitlab.WithContext(ctx))
	case "group":
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete GitLab deploy token %d in group %s", deployTokenId, typeId))
		_, err = client.DeployTokens.DeleteGroupDeployToken(typeId, deployTokenId, gitlab.WithContext(ctx))
	}
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceGitlabDeployTokenRead(ctx, d, meta)
}
