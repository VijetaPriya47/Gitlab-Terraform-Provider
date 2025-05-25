package sdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var _ = registerDataSource("gitlab_user_sshkeys", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_user_sshkeys`" + ` data source retrieves a list of SSH keys for a user.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/user_keys/#list-all-ssh-keys-for-a-user)`,

		ReadContext: dataSourceGitlabUserKeysRead,
		Schema: map[string]*schema.Schema{
			"user_id": {
				Description: "ID of the user to get the SSH keys for.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				ConflictsWith: []string{
					"username",
				},
			},
			"username": {
				Description: "Username of the user to get the SSH keys for.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ConflictsWith: []string{
					"user_id",
				},
			},
			"keys": {
				Description: "The user's keys.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: datasourceSchemaFromResourceSchema(gitlabUserSSHKeySchema(), nil, nil),
				},
			},
		},
	}
})

func dataSourceGitlabUserKeysRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	tflog.Info(ctx, "[INFO] Reading Gitlab user")

	options := gitlab.ListSSHKeysForUserOptions{
		PerPage: 2,
		Page:    1,
	}
	var keys []*gitlab.SSHKey

	userIDData, userIDOk := d.GetOk("user_id")
	usernameData, usernameOk := d.GetOk("username")
	var uid any
	if userIDOk {
		uid = userIDData.(int)
	} else if usernameOk {
		uid = strings.ToLower(usernameData.(string))
	} else {
		return diag.Errorf("one and only one of user_id or username must be set")
	}

	for options.Page != 0 {
		paginatedKeys, resp, err := client.Users.ListSSHKeysForUser(uid, &options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
		keys = append(keys, paginatedKeys...)
		options.Page = resp.NextPage
	}

	d.SetId(fmt.Sprintf("%d", userIDData))
	if err := d.Set("keys", flattenSSHKeysForState(keys)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func flattenSSHKeysForState(keys []*gitlab.SSHKey) (values []map[string]any) {
	for _, key := range keys {
		values = append(values, gitlabUserKeyToStateMap(key))
	}
	return values
}
