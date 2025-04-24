package sdk

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

const applicationSettingsID = "gitlab"

var _ = registerResource("gitlab_application_settings", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_application_settings` + "`" + ` resource allows to manage the GitLab application settings.

~> This is an **experimental resource**. By nature it doesn't properly fit into how Terraform resources are meant to work.
   Feel free to join the [discussion](https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/957) if you have any
   ideas or questions regarding this resource.

~> All ` + "`" + `gitlab_application_settings` + "`" + ` use the same ID ` + "`" + `gitlab` + "`" + `.

!> This resource does not implement any destroy logic, it's a no-op at this point.
   It's also not possible to revert to the previous settings.

-> Requires at administrative privileges on GitLab.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/settings/)`,

		CreateContext: resourceGitlabApplicationSettingsSet,
		ReadContext:   resourceGitlabApplicationSettingsRead,
		UpdateContext: resourceGitlabApplicationSettingsSet,
		DeleteContext: resourceGitlabApplicationSettingsDelete,

		Schema: gitlabApplicationSettingsSchema(),
	}
})

func resourceGitlabApplicationSettingsSet(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	tflog.Debug(ctx, "[DEBUG] update GitLab Application Settings")
	options := gitlabApplicationSettingsToUpdateOptions(d)

	// Since there is logic included in passing "nil" as value to the `enabled_git_access_protocol` and we
	// want to support that, we have to override that request value and make a separate call with a new struct
	if options.EnabledGitAccessProtocol != nil && strings.ToLower(*options.EnabledGitAccessProtocol) == "nil" {
		err := updateNilGitAccessSetting(client)
		if err != nil {
			return diag.FromErr(err)
		}

		// We need to set the value to explicit `nil` or the UpdateSettings call will error because it has `omitempty` on the struct
		options.EnabledGitAccessProtocol = nil
	}

	if (gitlab.UpdateSettingsOptions{}) != *options {
		_, _, err := api.UpdateSettings(client, options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(applicationSettingsID)
	return resourceGitlabApplicationSettingsRead(ctx, d, meta)
}

func resourceGitlabApplicationSettingsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	if d.Id() != applicationSettingsID {
		return diag.Errorf("The `gitlab_application_settings` resource can only exist once and requires the id to be `gitlab`")
	}

	client := meta.(*gitlab.Client)
	tflog.Debug(ctx, "[DEBUG] read GitLab Application settings")
	settings, _, err := api.GetSettings(client, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	stateMap := gitlabApplicationSettingsToStateMap(settings)
	if err = setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabApplicationSettingsDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	tflog.Debug(ctx, "[DEBUG] destroying the application settings does not yet do anything.")
	return nil
}

// Overrides the `omitempty` on the client-go struct and sets the `enabled_git_access_protocol` to nil
func updateNilGitAccessSetting(client *gitlab.Client) error {
	// Empty struct required for the method call.
	options := &gitlab.UpdateSettingsOptions{}

	// Call with an overwritten http body.
	_, _, err := api.UpdateSettings(client, options, func(request *retryablehttp.Request) error {
		optionsStruct := struct {
			EnabledGitAccessProtocol *string `url:"enabled_git_access_protocol" json:"enabled_git_access_protocol"`
		}{
			EnabledGitAccessProtocol: nil,
		}

		body, err := json.Marshal(optionsStruct)
		if err != nil {
			return err
		}

		err = request.SetBody(body)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}
