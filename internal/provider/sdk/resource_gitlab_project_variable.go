package sdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var _ = registerResource("gitlab_project_variable", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_project_variable` + "`" + ` resource allows to manage the lifecycle of a CI/CD variable for a project.

~> **Important:** If your GitLab version is older than 13.4, you may see nondeterministic behavior when updating or deleting gitlab_project_variable resources with non-unique keys, for example if there is another variable with the same key and different environment scope. See [this GitLab issue](https://gitlab.com/gitlab-org/gitlab/-/issues/9912).

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/project_level_variables.html)`,

		CreateContext: resourceGitlabProjectVariableCreate,
		ReadContext:   resourceGitlabProjectVariableRead,
		UpdateContext: resourceGitlabProjectVariableUpdate,
		DeleteContext: resourceGitlabProjectVariableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema:        gitlabProjectVariableGetSchema(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabProjectVariableResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabProjectVariableStateUpgradeV0,
				Version: 0,
			},
		},
	}
})

func resourceGitlabProjectVariableResourceV0() *schema.Resource {
	return &schema.Resource{Schema: gitlabProjectVariableGetSchema()}
}

// Perform a state migration from v0 -> v1. In v0, the ID accepted either a 2 digit or 3 digit ID, but we removed the backwards compatibility
// logic in 16.0. However, this meant some IDs still only had 2 values, since we previously didn't enforce 3. This migration will ensure all
// IDs conform to a 3-part ID by building out the ID from the raw components once if the state is v0 and a length of 2. A length of 3
// will still perform the migration, but leave the ID unchanged.
func resourceGitlabProjectVariableStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {

	// Parse the current ID to figure out the length
	id := rawState["id"].(string)
	parts := strings.Split(id, ":")

	// We only need to migrate the ID if the length is 2. If the length is 3, the current logic handles it appropriately.
	if len(parts) == 2 {
		// Retrieve information needed to generate a new key
		project := rawState["project"].(string)
		key := rawState["key"].(string)
		environmentScope := rawState["environment_scope"].(string)

		rawState["id"] = strings.Join([]string{project, key, environmentScope}, ":")
	}

	return rawState, nil
}

func resourceGitlabProjectVariableCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	project := d.Get("project").(string)
	key := d.Get("key").(string)
	value := d.Get("value").(string)
	variableType := stringToVariableType(d.Get("variable_type").(string))
	protected := d.Get("protected").(bool)
	masked := d.Get("masked").(bool)
	environmentScope := d.Get("environment_scope").(string)
	raw := d.Get("raw").(bool)
	description := d.Get("description").(string)

	options := gitlab.CreateProjectVariableOptions{
		Key:              &key,
		Value:            &value,
		VariableType:     variableType,
		Protected:        &protected,
		Masked:           &masked,
		EnvironmentScope: &environmentScope,
		Raw:              &raw,
		Description:      &description,
	}

	id := strings.Join([]string{project, key, environmentScope}, ":")

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab project variable %q", id))

	_, _, err := client.ProjectVariables.CreateVariable(project, &options, gitlab.WithContext(ctx))
	if err != nil {
		return augmentVariableClientError(ctx, d, err)
	}

	d.SetId(id)

	return resourceGitlabProjectVariableRead(ctx, d, meta)
}

func resourceGitlabProjectVariableRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	var (
		project          string
		key              string
		environmentScope string
	)

	parts := strings.SplitN(d.Id(), ":", 3)
	project = parts[0]
	key = parts[1]
	environmentScope = parts[2]

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab project variable %q", d.Id()))

	variable, _, err := client.ProjectVariables.GetVariable(project, key, nil, gitlab.WithContext(ctx), withEnvironmentScopeFilter(ctx, environmentScope))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab project variable %q was not found, removing from state", d.Id()))
			d.SetId("")
			return nil
		}
		return augmentVariableClientError(ctx, d, err)
	}

	stateMap := gitlabProjectVariableToStateMap(project, variable)
	if err = setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabProjectVariableUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	project := d.Get("project").(string)
	key := d.Get("key").(string)
	value := d.Get("value").(string)
	variableType := stringToVariableType(d.Get("variable_type").(string))
	protected := d.Get("protected").(bool)
	masked := d.Get("masked").(bool)
	environmentScope := d.Get("environment_scope").(string)
	raw := d.Get("raw").(bool)
	description := d.Get("description").(string)

	options := &gitlab.UpdateProjectVariableOptions{
		Value:            &value,
		VariableType:     variableType,
		Protected:        &protected,
		Masked:           &masked,
		EnvironmentScope: &environmentScope,
		Raw:              &raw,
		Description:      &description,
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab project variable %q", d.Id()))

	_, _, err := client.ProjectVariables.UpdateVariable(project, key, options, withEnvironmentScopeFilter(ctx, environmentScope))
	if err != nil {
		return augmentVariableClientError(ctx, d, err)
	}

	return resourceGitlabProjectVariableRead(ctx, d, meta)
}

func resourceGitlabProjectVariableDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	key := d.Get("key").(string)
	environmentScope := d.Get("environment_scope").(string)
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab project variable %q", d.Id()))

	// Note that the environment_scope filter is added here to support GitLab versions >= 13.4,
	// but it will be ignored in prior versions, causing nondeterministic destroy behavior when
	// destroying or updating scoped variables.
	// ref: https://gitlab.com/gitlab-org/gitlab/-/merge_requests/39209
	_, err := client.ProjectVariables.RemoveVariable(project, key, nil, withEnvironmentScopeFilter(ctx, environmentScope))
	return augmentVariableClientError(ctx, d, err)
}
