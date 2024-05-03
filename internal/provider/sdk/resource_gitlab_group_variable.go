package sdk

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_group_variable", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_group_variable` + "`" + ` resource allows to manage the lifecycle of a CI/CD variable for a group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/group_level_variables.html)`,

		CreateContext: resourceGitlabGroupVariableCreate,
		ReadContext:   resourceGitlabGroupVariableRead,
		UpdateContext: resourceGitlabGroupVariableUpdate,
		DeleteContext: resourceGitlabGroupVariableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: gitlabGroupVariableGetSchema(),
	}
})

func resourceGitlabGroupVariableCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	group := d.Get("group").(string)
	key := d.Get("key").(string)
	value := d.Get("value").(string)
	variableType := stringToVariableType(d.Get("variable_type").(string))
	protected := d.Get("protected").(bool)
	masked := d.Get("masked").(bool)
	environmentScope := d.Get("environment_scope").(string)
	raw := d.Get("raw").(bool)
	description := d.Get("description").(string)

	options := gitlab.CreateGroupVariableOptions{
		Key:              &key,
		Value:            &value,
		VariableType:     variableType,
		Protected:        &protected,
		Masked:           &masked,
		EnvironmentScope: &environmentScope,
		Raw:              &raw,
		Description:      &description,
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab group variable %s/%s", group, key))

	_, _, err := client.GroupVariables.CreateVariable(group, &options, gitlab.WithContext(ctx))
	if err != nil {
		return augmentVariableClientError(d, err)
	}

	keyScope := fmt.Sprintf("%s:%s", key, environmentScope)

	d.SetId(utils.BuildTwoPartID(&group, &keyScope))
	return resourceGitlabGroupVariableRead(ctx, d, meta)
}

func resourceGitlabGroupVariableRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	group, key, err := utils.ParseTwoPartID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	keyScope := strings.SplitN(key, ":", 2)
	scope := "*"
	if len(keyScope) == 2 {
		key = keyScope[0]
		scope = keyScope[1]
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab group variable %s/%s/%s", group, key, scope))

	v, _, err := client.GroupVariables.GetVariable(
		group,
		key,
		gitlab.WithContext(ctx),
		withEnvironmentScopeFilter(ctx, scope),
	)
	if err != nil {
		if api.Is404(err) {
			log.Printf("[DEBUG] gitlab group variable not found %s/%s", group, key)
			d.SetId("")
			return nil
		}
		return augmentVariableClientError(d, err)
	}

	stateMap := gitlabGroupVariableToStateMap(group, v)
	if err = setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabGroupVariableUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	group := d.Get("group").(string)
	key := d.Get("key").(string)
	value := d.Get("value").(string)
	variableType := stringToVariableType(d.Get("variable_type").(string))
	protected := d.Get("protected").(bool)
	masked := d.Get("masked").(bool)
	environmentScope := d.Get("environment_scope").(string)
	raw := d.Get("raw").(bool)
	description := d.Get("description").(string)

	options := &gitlab.UpdateGroupVariableOptions{
		Value:            &value,
		Protected:        &protected,
		VariableType:     variableType,
		Masked:           &masked,
		EnvironmentScope: &environmentScope,
		Raw:              &raw,
		Description:      &description,
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab group variable %s/%s/%s", group, key, environmentScope))

	_, _, err := client.GroupVariables.UpdateVariable(
		group,
		key,
		options,
		gitlab.WithContext(ctx),
		withEnvironmentScopeFilter(ctx, environmentScope),
	)
	if err != nil {
		return augmentVariableClientError(d, err)
	}
	return resourceGitlabGroupVariableRead(ctx, d, meta)
}

func resourceGitlabGroupVariableDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	group := d.Get("group").(string)
	key := d.Get("key").(string)
	environmentScope := d.Get("environment_scope").(string)
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab group variable %s/%s/%s", group, key, environmentScope))

	_, err := client.GroupVariables.RemoveVariable(
		group,
		key,
		gitlab.WithContext(ctx),
		withEnvironmentScopeFilter(ctx, environmentScope),
	)
	if err != nil {
		return augmentVariableClientError(d, err)
	}

	return nil
}
