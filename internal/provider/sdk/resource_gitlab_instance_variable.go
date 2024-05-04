package sdk

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var _ = registerResource("gitlab_instance_variable", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_instance_variable` + "`" + ` resource allows to manage the lifecycle of an instance-level CI/CD variable.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/instance_level_ci_variables.html)`,

		CreateContext: resourceGitlabInstanceVariableCreate,
		ReadContext:   resourceGitlabInstanceVariableRead,
		UpdateContext: resourceGitlabInstanceVariableUpdate,
		DeleteContext: resourceGitlabInstanceVariableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: gitlabInstanceVariableGetSchema(),
	}
})

func resourceGitlabInstanceVariableCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	key := d.Get("key").(string)
	value := d.Get("value").(string)
	variableType := stringToVariableType(d.Get("variable_type").(string))
	protected := d.Get("protected").(bool)
	masked := d.Get("masked").(bool)
	raw := d.Get("raw").(bool)

	options := gitlab.CreateInstanceVariableOptions{
		Key:          &key,
		Value:        &value,
		VariableType: variableType,
		Protected:    &protected,
		Masked:       &masked,
		Raw:          &raw,
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab instance level CI variable %s", key))

	_, _, err := client.InstanceVariables.CreateVariable(&options, gitlab.WithContext(ctx))
	if err != nil {
		return augmentVariableClientError(ctx, d, err)
	}

	d.SetId(key)
	return resourceGitlabInstanceVariableRead(ctx, d, meta)
}

func resourceGitlabInstanceVariableRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	key := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab instance level CI variable %s", key))

	v, _, err := client.InstanceVariables.GetVariable(key, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab instance level CI variable for %s not found so removing from state", d.Id()))
			d.SetId("")
			return nil
		}
		return augmentVariableClientError(ctx, d, err)
	}

	d.Set("key", v.Key)
	d.Set("value", v.Value)
	d.Set("variable_type", v.VariableType)
	d.Set("protected", v.Protected)
	d.Set("masked", v.Masked)
	d.Set("raw", v.Raw)

	stateMap := gitlabInstanceVariableToStateMap(v)
	if err = setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabInstanceVariableUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	key := d.Get("key").(string)
	value := d.Get("value").(string)
	variableType := stringToVariableType(d.Get("variable_type").(string))
	protected := d.Get("protected").(bool)
	masked := d.Get("masked").(bool)
	raw := d.Get("raw").(bool)

	options := &gitlab.UpdateInstanceVariableOptions{
		Value:        &value,
		Protected:    &protected,
		VariableType: variableType,
		Masked:       &masked,
		Raw:          &raw,
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab instance level CI variable %s", key))

	_, _, err := client.InstanceVariables.UpdateVariable(key, options, gitlab.WithContext(ctx))
	if err != nil {
		return augmentVariableClientError(ctx, d, err)
	}
	return resourceGitlabInstanceVariableRead(ctx, d, meta)
}

func resourceGitlabInstanceVariableDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	key := d.Get("key").(string)
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab instance level CI variable %s", key))

	_, err := client.InstanceVariables.RemoveVariable(key, gitlab.WithContext(ctx))
	if err != nil {
		return augmentVariableClientError(ctx, d, err)
	}

	return nil
}
