package sdk

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var _ = registerDataSource("gitlab_instance_variable", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_instance_variable`" + ` data source allows to retrieve details about an instance-level CI/CD variable.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/instance_level_ci_variables/)`,

		ReadContext: dataSourceGitlabInstanceVariableRead,
		Schema:      datasourceSchemaFromResourceSchema(gitlabInstanceVariableGetSchema(), []string{"key"}, nil),
	}
})

func dataSourceGitlabInstanceVariableRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	key := d.Get("key").(string)

	variable, _, err := client.InstanceVariables.GetVariable(key, nil, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(key)
	stateMap := gitlabInstanceVariableToStateMap(variable)
	if err := setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
