package sdk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_runner", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_runner`" + ` resource manages the lifecycle of a runner.

~> This resource has been deprecated in favor of the ` + "`gitlab_user_runner`" + ` resource. Please use that resource, and the new registration flow, instead. To be removed in 19.0.

A runner can either be registered at an instance level or group level.
The runner will be registered at a group level if the token used is from a group, or at an instance level if the token used is for the instance.

~> Using this resource will register a runner using the deprecated ` + "`registration_token`" + ` flow. To use the new ` + "`authentication_token`" + ` flow instead,
use the ` + "`gitlab_user_runner`" + ` resource!

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/runners/)`,

		DeprecationMessage: `This resource has been deprecated in favor of the ` + "`gitlab_user_runner`" + ` resource. Please use that resource, and the new registration flow, instead.`,
		CreateContext:      resourceGitLabRunnerCreate,
		UpdateContext:      resourceGitLabRunnerUpdate,
		ReadContext:        resourceGitLabRunnerRead,
		DeleteContext:      resourceGitLabRunnerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"registration_token": {
				Description: `The registration token used to register the runner.`,
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Sensitive:   true,
			},
			"description": {
				Description: `The runner's description.`,
				Type:        schema.TypeString,
				Optional:    true,
			},
			"paused": {
				Description: `Whether the runner should ignore new jobs.`,
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"locked": {
				Description: `Whether the runner should be locked for current project.`,
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"run_untagged": {
				Description: `Whether the runner should handle untagged jobs.`,
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"tag_list": {
				Description: `List of runner’s tags.`,
				Type:        schema.TypeSet,
				Set:         schema.HashString,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"access_level": {
				Description:  fmt.Sprintf(`The access_level of the runner. Valid values are: %s.`, utils.RenderValueListForDocs(runnerAccessLevelAllowedValues)),
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(runnerAccessLevelAllowedValues, true),
				Computed:     true,
			},
			"maximum_timeout": {
				Description: `Maximum timeout set when this runner handles the job.`,
				Type:        schema.TypeInt,
				Optional:    true,
			},
			// While Maintenance Note is available during "create", it is not available during "update", so
			// excluding it here for now.

			// Even though the output variable is just called 'token', we need to differentiate between the registration
			// and authentication token in terraform.
			"authentication_token": {
				Description: `The authentication token used for building a config.toml file. This value is not present when imported.`,
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
			"status": {
				Description: `The status of runners to show, one of: online and offline. active and paused are also possible values
				              which were deprecated in GitLab 14.8 and will be removed in GitLab 16.0.`,
				Type:     schema.TypeString,
				Computed: true,
			},
			"maintenance_note": {
				Description: `Free-form maintenance notes for the runner (1024 characters).`,
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}
})

func resourceGitLabRunnerCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	options := &gitlab.RegisterNewRunnerOptions{
		Token: gitlab.Ptr(d.Get("registration_token").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		options.Description = gitlab.Ptr(v.(string))
	}

	// GetOK skips the block if the value is "false", so need to use GetOkExists even though it's deprecated.
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("paused"); ok {
		options.Paused = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExist
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("locked"); ok {
		options.Locked = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("run_untagged"); ok {
		options.RunUntagged = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("tag_list"); ok {
		options.TagList = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("access_level"); ok {
		options.AccessLevel = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("maximum_timeout"); ok {
		options.MaximumTimeout = gitlab.Ptr(int64(v.(int)))
	}

	if v, ok := d.GetOk("maintenance_note"); ok {
		options.MaintenanceNote = gitlab.Ptr(v.(string))
	}

	// Explicitly not printing the registration token here, even though it may make debugging a bit trickier, since it's a secret
	tflog.Debug(ctx, "[DEBUG] Update GitLab Runner using registration token in configuration")
	runner, _, err := client.Runners.RegisterNewRunner(options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(runner.ID, 10))

	// The authentication_token will ONLY exist during creation, and will not return during "read", so we need to set it here.
	d.Set("authentication_token", runner.Token)

	return resourceGitLabRunnerRead(ctx, d, meta)
}

func resourceGitLabRunnerRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	runnerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	runner, _, err := client.Runners.GetRunnerDetails(runnerID, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(runner.ID, 10))
	d.Set("description", runner.Description)
	d.Set("paused", runner.Paused)
	d.Set("locked", runner.Locked)
	d.Set("run_untagged", runner.RunUntagged)
	d.Set("access_level", runner.AccessLevel)
	d.Set("maximum_timeout", runner.MaximumTimeout)
	d.Set("status", runner.Status)
	d.Set("maintenance_note", runner.MaintenanceNote)

	if err := d.Set("tag_list", runner.TagList); err != nil {
		return diag.FromErr(fmt.Errorf("[DEBUG] error setting tag list for runner: %s", err))
	}

	return nil
}

func resourceGitLabRunnerUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	runnerID := d.Id()

	options := &gitlab.UpdateRunnerDetailsOptions{}
	if v, ok := d.GetOk("description"); ok {
		options.Description = gitlab.Ptr(v.(string))
	}

	// GetOK skips the block if the value is "false", so need to use GetOkExists even though it's deprecated.
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("paused"); ok {
		options.Paused = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("locked"); ok {
		options.Locked = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("run_untagged"); ok {
		options.RunUntagged = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("tag_list"); ok {
		options.TagList = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("access_level"); ok {
		options.AccessLevel = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("maximum_timeout"); ok {
		options.MaximumTimeout = gitlab.Ptr(int64(v.(int)))
	}
	if v, ok := d.GetOk("maintenance_note"); ok {
		options.MaintenanceNote = gitlab.Ptr(v.(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Update GitLab Runner %s", d.Id()))
	_, _, err := client.Runners.UpdateRunnerDetails(runnerID, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitLabRunnerRead(ctx, d, meta)
}

func resourceGitLabRunnerDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	runnerID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete GitLab Runner %s", d.Id()))
	_, err = client.Runners.DeleteRegisteredRunnerByID(runnerID, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

var runnerAccessLevelAllowedValues = []string{
	"not_protected",
	"ref_protected",
}
