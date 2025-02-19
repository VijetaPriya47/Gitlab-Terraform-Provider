package sdk

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_user_sshkey", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_user_sshkey` + "`" + ` resource allows to manage the lifecycle of an SSH key assigned to a user.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/users/#single-ssh-key)`,

		CreateContext: resourceGitlabUserSSHKeyCreate,
		ReadContext:   resourceGitlabUserSSHKeyRead,
		DeleteContext: resourceGitlabUserSSHKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: gitlabUserSSHKeySchema(),
	}
})

func resourceGitlabUserSSHKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	userID, userIDOk := d.GetOk("user_id")

	options := &gitlab.AddSSHKeyOptions{
		Title: gitlab.Ptr(d.Get("title").(string)),
		Key:   gitlab.Ptr(d.Get("key").(string)),
	}

	if expiresAt, ok := d.GetOk("expires_at"); ok {
		parsedExpiresAt, err := time.Parse(time.RFC3339, expiresAt.(string))
		if err != nil {
			return diag.Errorf("failed to parse created_at: %s. It must be in valid RFC3339 format.", err)
		}
		gitlabExpiresAt := gitlab.ISOTime(parsedExpiresAt)
		options.ExpiresAt = &gitlabExpiresAt
	}

	var isAdmin bool
	var key *gitlab.SSHKey
	var err error

	if userIDOk {
		isAdmin, err = api.IsCurrentUserAdmin(ctx, client)
		if err != nil {
			return diag.Errorf("failed to check if user is admin for configuring ssh keys for a user")
		}
		if !isAdmin {
			return diag.Errorf("current user needs to be admin for configuring ssh keys for a user")
		}

		key, _, err = client.Users.AddSSHKeyForUser(userID.(int), options, gitlab.WithContext(ctx))

	} else {
		key, _, err = client.Users.AddSSHKey(options, gitlab.WithContext(ctx))
	}

	if err != nil {
		return diag.FromErr(err)
	}

	var userIDForID string
	if userIDOk {
		userIDForID = fmt.Sprintf("%d", userID.(int))

	} else {
		user, _, err := client.Users.CurrentUser(gitlab.WithContext(ctx))
		if err != nil {
			return diag.Errorf("failed to get current user: %s", err)
		}

		userIDForID = fmt.Sprintf("%d", user.ID)
	}

	keyIDForID := fmt.Sprintf("%d", key.ID)
	d.SetId(utils.BuildTwoPartID(&userIDForID, &keyIDForID))
	return resourceGitlabUserSSHKeyRead(ctx, d, meta)
}

func resourceGitlabUserSSHKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	userID, keyID, err := resourceGitlabUserSSHKeyParseID(d.Id())
	if err != nil {
		return diag.Errorf("unable to parse user ssh key resource id: %s: %v", d.Id(), err)
	}

	options := &gitlab.ListSSHKeysForUserOptions{
		Page:    1,
		PerPage: 20,
	}

	var key *gitlab.SSHKey
	for options.Page != 0 && key == nil {
		keys, resp, err := client.Users.ListSSHKeysForUser(userID, options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		for _, k := range keys {
			if k.ID == keyID {
				key = k
				break
			}
		}

		options.Page = resp.NextPage
	}

	if key == nil {
		tflog.Warn(ctx, fmt.Sprintf("Could not find sshkey %d for user %d", keyID, userID))
		d.SetId("")
		return nil
	}

	d.Set("user_id", userID)
	d.Set("key_id", keyID)
	d.Set("title", key.Title)
	d.Set("key", key.Key)
	if key.ExpiresAt != nil {
		d.Set("expires_at", key.ExpiresAt.Format(time.RFC3339))
	}
	if key.CreatedAt != nil {
		d.Set("created_at", key.CreatedAt.Format(time.RFC3339))
	}
	return nil
}

func resourceGitlabUserSSHKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	_, keyID, err := resourceGitlabUserSSHKeyParseID(d.Id())
	if err != nil {
		return diag.Errorf("unable to parse user ssh key resource id: %s: %v", d.Id(), err)
	}

	var isAdmin bool

	isAdmin, err = api.IsCurrentUserAdmin(ctx, client)
	if err != nil {
		return diag.Errorf("failed to check if user is admin for configuring ssh keys for a user")
	}

	userID := d.Get("user_id").(int)

	if isAdmin {
		_, err = client.Users.DeleteSSHKeyForUser(userID, keyID, gitlab.WithContext(ctx))

	} else {
		_, err = client.Users.DeleteSSHKey(keyID, gitlab.WithContext(ctx))
	}

	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitlabUserSSHKeyParseID(id string) (int, int, error) {
	userIDFromID, keyIDFromID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return 0, 0, err
	}
	userID, err := strconv.Atoi(userIDFromID)
	if err != nil {
		return 0, 0, err
	}
	keyID, err := strconv.Atoi(keyIDFromID)
	if err != nil {
		return 0, 0, err
	}

	return userID, keyID, nil
}
