package sdk

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Values to be used for validation and documentation
var (
	defaultBranchProtectionValues         = []int{0, 1, 2, 3, 4}
	defaultBranchProtectionDefaultsValues = []string{api.AccessLevelValueToName[gitlab.DeveloperPermissions], api.AccessLevelValueToName[gitlab.MaintainerPermissions], api.AccessLevelValueToName[gitlab.NoPermissions]}
	visibilityLevelValues                 = []string{"private", "internal", "public"}
	projectCreationLevelValues            = []string{string(gitlab.NoOneProjectCreation), string(gitlab.OwnerProjectCreation), string(gitlab.MaintainerProjectCreation), string(gitlab.DeveloperProjectCreation)}
	subGroupCreationLevelValues           = []string{string(gitlab.OwnerSubGroupCreationLevelValue), string(gitlab.MaintainerSubGroupCreationLevelValue)}
	validSharedRunnersSettings            = []string{
		"enabled",
		"disabled_and_overridable",
		"disabled_and_unoverridable",

		// Deprecated
		"disabled_with_override",
	}
)

var _ = registerResource("gitlab_group", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_group`" + ` resource allows to manage the lifecycle of a group.

-> On GitLab.com, you cannot use the ` + "`gitlab_group`" + ` resource to create a [top-level group](https://docs.gitlab.com/user/group/#group-hierarchy). Instead, you must [create a group](https://docs.gitlab.com/user/group/#create-a-group) in the UI, then import the group into your Terraform configuration. From here, you can manage the group using the Terraform Provider.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/groups/)`,

		CreateContext: resourceGitlabGroupCreate,
		ReadContext:   resourceGitlabGroupRead,
		UpdateContext: resourceGitlabGroupUpdate,
		DeleteContext: resourceGitlabGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: constructSchema(map[string]*schema.Schema{
			"name": {
				Description: "The name of the group.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"path": {
				Description: "The path of the group.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"full_path": {
				Description: "The full path of the group.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"full_name": {
				Description: "The full name of the group.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"web_url": {
				Description: "Web URL of the group.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"description": {
				Description: "The group's description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"lfs_enabled": {
				Description: "Enable/disable Large File Storage (LFS) for the projects in this group.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"default_branch": {
				Description: "Initial default branch name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"default_branch_protection": {
				Description:  fmt.Sprintf("See https://docs.gitlab.com/api/groups/#options-for-default_branch_protection. Valid values are: %s.", utils.RenderIntValueListForDocs(defaultBranchProtectionValues)),
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntInSlice(defaultBranchProtectionValues),
				Deprecated:   "Deprecated in GitLab 17.0, due for removal in v5 of the API. Use default_branch_protection_defaults instead.",
				ConflictsWith: []string{
					"default_branch_protection_defaults",
				},
			},
			"default_branch_protection_defaults": {
				Description: "The default branch protection defaults",
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				ConflictsWith: []string{
					"default_branch_protection",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allowed_to_push": {
							Description: fmt.Sprintf("An array of access levels allowed to push. Valid values are: %s.", utils.RenderValueListForDocs(defaultBranchProtectionDefaultsValues)),
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringInSlice(defaultBranchProtectionDefaultsValues, false),
							},
						},
						"allow_force_push": {
							Description: "Allow force push for all users with push access.",
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
						},
						"allowed_to_merge": {
							Description: fmt.Sprintf("An array of access levels allowed to merge. Valid values are: %s.", utils.RenderValueListForDocs(defaultBranchProtectionDefaultsValues)),
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringInSlice(defaultBranchProtectionDefaultsValues, false),
							},
						},
						"developer_can_initial_push": {
							Description: "Allow developers to initial push.",
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"request_access_enabled": {
				Description: "Allow users to request member access.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"visibility_level": {
				Description:  fmt.Sprintf("The group's visibility. Can be `private`, `internal`, or `public`. Valid values are: %s.", utils.RenderValueListForDocs(visibilityLevelValues)),
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(visibilityLevelValues, true),
			},
			"share_with_group_lock": {
				Description: "Prevent sharing a project with another group within this group.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"project_creation_level": {
				Description:  fmt.Sprintf("Determine if developers can create projects in the group. Valid values are: %s", utils.RenderValueListForDocs(projectCreationLevelValues)),
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(projectCreationLevelValues, true),
			},
			"auto_devops_enabled": {
				Description: "Default to Auto DevOps pipeline for all projects within this group.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"emails_enabled": {
				Description: "Enable email notifications.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"mentions_disabled": {
				Description: "Disable the capability of a group from getting mentioned.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"subgroup_creation_level": {
				Description:  fmt.Sprintf("Allowed to create subgroups. Valid values are: %s.", utils.RenderValueListForDocs(subGroupCreationLevelValues)),
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(subGroupCreationLevelValues, true),
			},
			"require_two_factor_authentication": {
				Description: "Require all users in this group to setup Two-factor authentication.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"two_factor_grace_period": {
				Description: "Defaults to 48. Time before Two-factor authentication is enforced (in hours).",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"parent_id": {
				Description: "Id of the parent group (creates a nested group).",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"runners_token": {
				Description: "The group level registration token to use during runner setup.",
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
			"prevent_forking_outside_group": {
				Description: "Defaults to false. When enabled, users can not fork projects from this group to external namespaces.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"membership_lock": {
				Description: "Users cannot be added to projects in this group.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"extra_shared_runners_minutes_limit": {
				Description: "Can be set by administrators only. Additional CI/CD minutes for this group.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"shared_runners_minutes_limit": {
				Description: "Can be set by administrators only. Maximum number of monthly CI/CD minutes for this group. Can be nil (default; inherit system default), 0 (unlimited), or > 0.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"ip_restriction_ranges": {
				Description: "A list of IP addresses or subnet masks to restrict group access. Will be concatenated together into a comma separated string. Only allowed on top level groups.",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"allowed_email_domains_list": {
				Description: "A list of email address domains to allow group access. Will be concatenated together into a comma separated string.",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Computed:    true,
			},
			"wiki_access_level": {
				Description:  fmt.Sprintf("The group's wiki access level. Only available on Premium and Ultimate plans. Valid values are %s.", utils.RenderValueListForDocs(validWikiAccessLevels)),
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(validWikiAccessLevels, true),
			},
			"shared_runners_setting": {
				Description:  fmt.Sprintf("Enable or disable shared runners for a group’s subgroups and projects. Valid values are: %s.", utils.RenderValueListForDocs(validSharedRunnersSettings)),
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(validSharedRunnersSettings, false),
			},
			"permanently_remove_on_delete": {
				Description: "Whether the group should be permanently removed during a `delete` operation. This only works with subgroups. Must be configured via an `apply` before the `destroy` is run.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"push_rules": {
				Description: "Push rules for the group.",
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"author_email_regex": {
							Description: "All commit author emails must match this regex, e.g. `@my-company.com$`.",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						"branch_name_regex": {
							Description: "All branch names must match this regex, e.g. `(feature|hotfix)\\/*`.",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						"commit_message_regex": {
							Description: "All commit messages must match this regex, e.g. `Fixed \\d+\\..*`.",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						"commit_message_negative_regex": {
							Description: "No commit message is allowed to match this regex, for example `ssh\\:\\/\\/`.",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						"file_name_regex": {
							Description: "Filenames matching the regular expression provided in this attribute are not allowed, for example, `(jar|exe)$`.",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						"commit_committer_check": {
							Description: "Only commits pushed using verified emails are allowed.",
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
						},
						"commit_committer_name_check": {
							Description: "Users can only push commits to this repository if the commit author name is consistent with their GitLab account name.",
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
						},
						"deny_delete_tag": {
							Description: "Deny deleting a tag.",
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
						},
						"member_check": {
							Description: "Allows only GitLab users to author commits.",
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
						},
						"prevent_secrets": {
							Description: "GitLab will reject any files that are likely to contain secrets.",
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
						},
						"reject_unsigned_commits": {
							Description: "Only commits signed through GPG are allowed.",
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
						},
						"reject_non_dco_commits": {
							Description: "Reject commit when it’s not DCO certified.",
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
						},
						"max_file_size": {
							Description:  "Maximum file size (MB) allowed.",
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.IntAtLeast(0),
						},
					},
				},
			},
		}, avatarableSchema()),
		CustomizeDiff: avatarableDiff,
	}
})

func resourceGitlabGroupCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	options := &gitlab.CreateGroupOptions{
		Name: gitlab.Ptr(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("path"); ok {
		options.Path = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("default_branch"); ok {
		options.DefaultBranch = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("description"); ok {
		options.Description = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("visibility_level"); ok {
		options.Visibility = stringToVisibilityLevel(v.(string))
	}

	if v, ok := d.GetOk("share_with_group_lock"); ok {
		options.ShareWithGroupLock = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("lfs_enabled"); ok {
		options.LFSEnabled = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("request_access_enabled"); ok {
		options.RequestAccessEnabled = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("require_two_factor_authentication"); ok {
		options.RequireTwoFactorAuth = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("two_factor_grace_period"); ok {
		options.TwoFactorGracePeriod = gitlab.Ptr(v.(int))
	}

	if v, ok := d.GetOk("project_creation_level"); ok {
		options.ProjectCreationLevel = stringToProjectCreationLevel(v.(string))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("auto_devops_enabled"); ok {
		options.AutoDevopsEnabled = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("subgroup_creation_level"); ok {
		options.SubGroupCreationLevel = stringToSubGroupCreationLevel(v.(string))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("emails_enabled"); ok {
		options.EmailsEnabled = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("mentions_disabled"); ok {
		options.MentionsDisabled = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("parent_id"); ok {
		options.ParentID = gitlab.Ptr(v.(int))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("default_branch_protection"); ok {
		options.DefaultBranchProtection = gitlab.Ptr(v.(int))
	}

	if v, ok := d.GetOk("default_branch_protection_defaults.0"); ok {
		defaults := v.(map[string]any)
		options.DefaultBranchProtectionDefaults = &gitlab.DefaultBranchProtectionDefaultsOptions{
			AllowedToPush:           gitlab.Ptr(convertAccessLevelNamesToValues(defaults["allowed_to_push"].([]any))),
			AllowForcePush:          gitlab.Ptr(defaults["allow_force_push"].(bool)),
			AllowedToMerge:          gitlab.Ptr(convertAccessLevelNamesToValues(defaults["allowed_to_merge"].([]any))),
			DeveloperCanInitialPush: gitlab.Ptr(defaults["developer_can_initial_push"].(bool)),
		}
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("membership_lock"); ok {
		options.MembershipLock = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("extra_shared_runners_minutes_limit"); ok {
		options.ExtraSharedRunnersMinutesLimit = gitlab.Ptr(v.(int))
	}

	if v, ok := d.GetOk("shared_runners_minutes_limit"); ok {
		options.SharedRunnersMinutesLimit = gitlab.Ptr(v.(int))
	}

	avatar, err := handleAvatarOnCreate(d)
	if err != nil {
		return diag.FromErr(err)
	}
	if avatar != nil {
		options.Avatar = &gitlab.GroupAvatar{
			Filename: avatar.Filename,
			Image:    avatar.Image,
		}
	}

	if v, ok := d.GetOk("wiki_access_level"); ok {
		options.WikiAccessLevel = stringToAccessControlValue(v.(string))
	}

	tflog.Debug(ctx, "[DEBUG] create gitlab group", map[string]any{
		"name": *options.Name,
	})

	group, _, err := client.Groups.CreateGroup(options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	// Wait for the Group to return properly before we update it
	// Groups are created asynchronously, so we want to ensure the create operation
	// completely finishes before we act on the group, or we can get an error.
	// see: https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/692
	stateConf := &retry.StateChangeConf{
		Pending: []string{"Creating"},
		Target:  []string{"Created"},
		Refresh: func() (any, string, error) {
			out, _, err := client.Groups.GetGroup(group.ID, nil, gitlab.WithContext(ctx))
			if err != nil {
				if api.Is404(err) {
					return out, "Creating", nil
				}
				tflog.Error(ctx, "[ERROR] Received error retrieving group", map[string]any{
					"group_id": group.ID,
					"error":    err,
				})
				return out, "Error", err
			}
			return out, "Created", nil
		},

		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 3 * time.Second,
		Delay:      5 * time.Second,
	}

	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("error waiting for group (%s) to create: %s", d.Id(), err)
	}

	// Our group has been created, we can now update it.
	d.SetId(fmt.Sprintf("%d", group.ID))

	if _, ok := d.GetOk("push_rules"); ok {
		err := editOrAddGroupPushRules(ctx, client, d.Id(), d)
		if err != nil {
			if api.Is404(err) {
				tflog.Error(ctx, "[ERROR] Failed to edit push rules for group", map[string]any{
					"group_id": d.Id(),
					"error":    err,
				})
				return diag.Errorf("Group push rules are not supported in your version of GitLab")
			}
			return diag.Errorf("Failed to edit push rules for group %q: %s", d.Id(), err)
		}
	}

	var updateOptions gitlab.UpdateGroupOptions

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("prevent_forking_outside_group"); ok {
		updateOptions.PreventForkingOutsideGroup = gitlab.Ptr(v.(bool))
	}

	// IP Restriction can only be set on update.
	if v, ok := d.GetOk("ip_restriction_ranges"); ok {
		updateOptions.IPRestrictionRanges = stringListToCommaSeparatedString(v.([]any))
	}

	// Email domains can only be set on update.
	if v, ok := d.GetOk("allowed_email_domains_list"); ok {
		updateOptions.AllowedEmailDomainsList = stringListToCommaSeparatedString(v.([]any))
	}

	if v, ok := d.GetOk("shared_runners_setting"); ok {
		updateOptions.SharedRunnersSetting = stringToSharedRunnersSetting(v.(string))
	}

	if (updateOptions != gitlab.UpdateGroupOptions{}) {
		if _, _, err = client.Groups.UpdateGroup(d.Id(), &updateOptions, gitlab.WithContext(ctx)); err != nil {
			return diag.Errorf("could not update group after creation %q: %s", d.Id(), err)
		}
	}

	return resourceGitlabGroupRead(ctx, d, meta)
}

func convertAccessLevelNamesToValues(names []any) []*gitlab.GroupAccessLevel {
	valuesList := []*gitlab.GroupAccessLevel{}

	for _, name := range names {
		nameStr := name.(string)
		groupAccessLevel := gitlab.GroupAccessLevel{
			AccessLevel: gitlab.Ptr(api.AccessLevelNameToValue[nameStr]),
		}
		valuesList = append(valuesList, &groupAccessLevel)
	}

	return valuesList
}

func resourceGitlabGroupRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	tflog.Debug(ctx, "[DEBUG] read gitlab group", map[string]any{
		"group_id": d.Id(),
	})

	group, _, err := client.Groups.GetGroup(
		d.Id(),
		&gitlab.GetGroupOptions{WithProjects: gitlab.Ptr(false)},
		gitlab.WithContext(ctx),
	)
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "[DEBUG] gitlab group not found so removing", map[string]any{
				"id": d.Id(),
			})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if group.MarkedForDeletionOn != nil {
		tflog.Debug(ctx, "[DEBUG] gitlab group marked for deletion", map[string]any{
			"id": d.Id(),
		})
		d.SetId("")
		return nil
	}

	d.SetId(fmt.Sprintf("%d", group.ID))
	d.Set("name", group.Name)
	d.Set("path", group.Path)
	d.Set("full_path", group.FullPath)
	d.Set("full_name", group.FullName)
	d.Set("web_url", group.WebURL)
	d.Set("default_branch", group.DefaultBranch)
	d.Set("description", group.Description)
	d.Set("lfs_enabled", group.LFSEnabled)
	d.Set("request_access_enabled", group.RequestAccessEnabled)
	d.Set("visibility_level", group.Visibility)
	d.Set("project_creation_level", group.ProjectCreationLevel)
	d.Set("subgroup_creation_level", group.SubGroupCreationLevel)
	d.Set("require_two_factor_authentication", group.RequireTwoFactorAuth)
	d.Set("two_factor_grace_period", group.TwoFactorGracePeriod)
	d.Set("auto_devops_enabled", group.AutoDevopsEnabled)
	d.Set("mentions_disabled", group.MentionsDisabled)
	d.Set("parent_id", group.ParentID)
	d.Set("runners_token", group.RunnersToken)
	d.Set("share_with_group_lock", group.ShareWithGroupLock)
	d.Set("prevent_forking_outside_group", group.PreventForkingOutsideGroup)
	d.Set("membership_lock", group.MembershipLock)
	d.Set("extra_shared_runners_minutes_limit", group.ExtraSharedRunnersMinutesLimit)
	d.Set("shared_runners_minutes_limit", group.SharedRunnersMinutesLimit)
	d.Set("avatar_url", group.AvatarURL)
	d.Set("wiki_access_level", group.WikiAccessLevel)
	d.Set("shared_runners_setting", group.SharedRunnersSetting)
	d.Set("emails_enabled", group.EmailsEnabled)

	// nolint:staticcheck // SA1019 ignore deprecated DefaultBranchProtection
	d.Set("default_branch_protection", group.DefaultBranchProtection)

	if group.DefaultBranchProtectionDefaults != nil {
		err = d.Set("default_branch_protection_defaults", []map[string]any{
			{
				"allowed_to_push":            convertAccessLevelValuesToNames(group.DefaultBranchProtectionDefaults.AllowedToPush),
				"allow_force_push":           group.DefaultBranchProtectionDefaults.AllowForcePush,
				"allowed_to_merge":           convertAccessLevelValuesToNames(group.DefaultBranchProtectionDefaults.AllowedToMerge),
				"developer_can_initial_push": group.DefaultBranchProtectionDefaults.DeveloperCanInitialPush,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// The value comes back from the API as a comma separated string, and stores in TF as a set.
	// We need to set the value only if it's "", otherwise the split gives up [""] which will result
	// in a non-empty plan.
	IPValue := []string{}
	if group.IPRestrictionRanges != "" {
		IPValue = strings.Split(group.IPRestrictionRanges, ",")
	}

	if err := d.Set("ip_restriction_ranges", IPValue); err != nil {
		tflog.Error(ctx, "Error setting ip_restriction_ranges.")
		return diag.FromErr(err)
	}

	// The value comes back from the API as a comma separated string, and stores in TF as a set.
	// We need to set the value only if it's "", otherwise the split gives up [""] which will result
	// in a non-empty plan.
	emailDomains := []string{}
	if group.AllowedEmailDomainsList != "" {
		emailDomains = strings.Split(group.AllowedEmailDomainsList, ",")
	}

	if err := d.Set("allowed_email_domains_list", emailDomains); err != nil {
		tflog.Error(ctx, "Error setting allowed_email_domains_list.")
		return diag.FromErr(err)
	}

	isEE, err := utils.IsRunningInEEContext(client)
	if err != nil {
		return diag.FromErr(err)
	}

	if isEE {
		tflog.Debug(ctx, "[DEBUG] read gitlab group push rules", map[string]any{"id": d.Id()})

		pushRules, _, err := client.Groups.GetGroupPushRules(d.Id(), gitlab.WithContext(ctx))
		if api.Is404(err) {
			tflog.Error(ctx, "[ERROR] Failed to get push rules for group", map[string]any{
				"group_id": d.Id(),
				"error":    err,
			})
		} else if err != nil {
			return diag.Errorf("Failed to get push rules for group %q: %s", d.Id(), err)
		}
		pushRuleValues, err := flattenGroupPushRules(ctx, client, pushRules)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("push_rules", pushRuleValues); err != nil {
			return diag.FromErr(err)
		}
	} else {
		tflog.Debug(ctx, "[DEBUG] gitlab group push rule not read due to gitlab community edition")
	}

	return nil
}

func convertAccessLevelValuesToNames(values []*gitlab.GroupAccessLevel) []*string {
	namesList := []*string{}

	for _, value := range values {
		namesList = append(namesList, gitlab.Ptr(api.AccessLevelValueToName[*value.AccessLevel]))
	}

	return namesList
}

func resourceGitlabGroupUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	options := &gitlab.UpdateGroupOptions{}

	if d.HasChange("name") {
		options.Name = gitlab.Ptr(d.Get("name").(string))
	}

	if d.HasChange("path") {
		options.Path = gitlab.Ptr(d.Get("path").(string))
	}

	if d.HasChange("default_branch") {
		options.DefaultBranch = gitlab.Ptr(d.Get("default_branch").(string))
	}

	if d.HasChange("description") {
		options.Description = gitlab.Ptr(d.Get("description").(string))
	}

	if d.HasChange("lfs_enabled") {
		options.LFSEnabled = gitlab.Ptr(d.Get("lfs_enabled").(bool))
	}

	if d.HasChange("request_access_enabled") {
		options.RequestAccessEnabled = gitlab.Ptr(d.Get("request_access_enabled").(bool))
	}

	// Always set visibility ; workaround for
	// https://gitlab.com/gitlab-org/gitlab-foss/-/issues/38459
	if v, ok := d.GetOk("visibility_level"); ok {
		options.Visibility = stringToVisibilityLevel(v.(string))
	}

	if d.HasChange("project_creation_level") {
		options.ProjectCreationLevel = stringToProjectCreationLevel(d.Get("project_creation_level").(string))
	}

	if d.HasChange("subgroup_creation_level") {
		options.SubGroupCreationLevel = stringToSubGroupCreationLevel(d.Get("subgroup_creation_level").(string))
	}

	if d.HasChange("require_two_factor_authentication") {
		options.RequireTwoFactorAuth = gitlab.Ptr(d.Get("require_two_factor_authentication").(bool))
	}

	if d.HasChange("two_factor_grace_period") {
		options.TwoFactorGracePeriod = gitlab.Ptr(d.Get("two_factor_grace_period").(int))
	}

	if d.HasChange("auto_devops_enabled") {
		options.AutoDevopsEnabled = gitlab.Ptr(d.Get("auto_devops_enabled").(bool))
	}

	if d.HasChange("emails_enabled") {
		options.EmailsEnabled = gitlab.Ptr(d.Get("emails_enabled").(bool))
	}

	if d.HasChange("mentions_disabled") {
		options.MentionsDisabled = gitlab.Ptr(d.Get("mentions_disabled").(bool))
	}

	if d.HasChange("share_with_group_lock") {
		options.ShareWithGroupLock = gitlab.Ptr(d.Get("share_with_group_lock").(bool))
	}

	if d.HasChange("default_branch_protection") {
		// nolint:staticcheck // SA1019 ignore deprecated DefaultBranchProtection
		options.DefaultBranchProtection = gitlab.Ptr(d.Get("default_branch_protection").(int))
	}

	if d.HasChange("default_branch_protection_defaults.0") {
		options.DefaultBranchProtectionDefaults = gitlab.Ptr(expandDefaultBranchProtectionDefaults(d))
	}

	if d.HasChange("prevent_forking_outside_group") {
		options.PreventForkingOutsideGroup = gitlab.Ptr(d.Get("prevent_forking_outside_group").(bool))
	}

	if d.HasChange("membership_lock") {
		options.MembershipLock = gitlab.Ptr(d.Get("membership_lock").(bool))
	}

	if d.HasChange("extra_shared_runners_minutes_limit") {
		options.ExtraSharedRunnersMinutesLimit = gitlab.Ptr(d.Get("extra_shared_runners_minutes_limit").(int))
	}

	if d.HasChange("shared_runners_minutes_limit") {
		options.SharedRunnersMinutesLimit = gitlab.Ptr(d.Get("shared_runners_minutes_limit").(int))
	}

	if d.HasChange("ip_restriction_ranges") {
		options.IPRestrictionRanges = stringListToCommaSeparatedString(d.Get("ip_restriction_ranges").([]any))
	}

	if d.HasChange("allowed_email_domains_list") {
		options.AllowedEmailDomainsList = stringListToCommaSeparatedString(d.Get("allowed_email_domains_list").([]any))
	}

	avatar, err := handleAvatarOnUpdate(d)
	if err != nil {
		return diag.FromErr(err)
	}
	if avatar != nil {
		options.Avatar = &gitlab.GroupAvatar{
			Filename: avatar.Filename,
			Image:    avatar.Image,
		}
	}

	if d.HasChange("wiki_access_level") {
		options.WikiAccessLevel = stringToAccessControlValue(d.Get("wiki_access_level").(string))
	}

	if d.HasChange("shared_runners_setting") {
		options.SharedRunnersSetting = stringToSharedRunnersSetting(d.Get("shared_runners_setting").(string))
	}

	tflog.Debug(ctx, "update gitlab group", map[string]any{
		"group_id": d.Id(),
		"options":  fmt.Sprintf("%+v", options),
	})

	_, _, err = client.Groups.UpdateGroup(d.Id(), options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("parent_id") {
		err = transferSubGroup(ctx, d, client)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("push_rules") {
		err := editOrAddGroupPushRules(ctx, client, d.Id(), d)
		if err != nil {
			if api.Is404(err) {
				tflog.Error(ctx, "[ERROR] Failed to edit push rules for group", map[string]any{
					"group_id": d.Id(),
					"error":    err,
				})
				return diag.Errorf("Group push rules are not supported in your version of GitLab")
			}
			return diag.Errorf("Failed to edit push rules for group %q: %s", d.Id(), err)
		}
	}

	return resourceGitlabGroupRead(ctx, d, meta)
}

func transferSubGroup(ctx context.Context, d *schema.ResourceData, client *gitlab.Client) error {
	o, n := d.GetChange("parent_id")
	parentId, ok := n.(int)
	if !ok {
		return fmt.Errorf("error converting parent_id %v into an int", n)
	}

	opt := &gitlab.TransferSubGroupOptions{}
	if parentId != 0 {
		tflog.Debug(ctx, "transfer gitlab group", map[string]any{
			"group_id":  d.Id(),
			"old_group": o,
			"new_group": parentId,
		})

		opt.GroupID = gitlab.Ptr(parentId)
	} else {
		tflog.Debug(ctx, "turn gitlab group into a new top-level group", map[string]any{
			"group_id":  d.Id(),
			"old_group": o,
		})
	}

	_, _, err := client.Groups.TransferSubGroup(d.Id(), opt, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("error transfering group %s to new parent group %v: %s", d.Id(), parentId, err)
	}

	return nil
}

func resourceGitlabGroupDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	tflog.Debug(ctx, "delete gitlab group", map[string]any{
		"id": d.Id(),
	})

	_, err := client.Groups.DeleteGroup(d.Id(), nil, gitlab.WithContext(ctx))
	if err != nil && !strings.Contains(err.Error(), "Group has been already marked for deletion") {
		return diag.Errorf("error deleting group %s: %s", d.Id(), err)
	}

	// Wait for the group to be deleted.
	// Deleting a group in gitlab is async.
	stateConf := &retry.StateChangeConf{
		Pending: []string{"Deleting"},
		Target:  []string{"Deleted"},
		Refresh: func() (any, string, error) {
			out, response, err := client.Groups.GetGroup(d.Id(), nil, gitlab.WithContext(ctx))
			if err != nil {
				if response != nil && response.StatusCode == 404 {
					return out, "Deleted", nil
				}
				tflog.Error(ctx, "Received error", map[string]any{
					"error": err,
				})
				return out, "Error", err
			}
			if out.MarkedForDeletionOn != nil {
				// Represents a Gitlab EE soft-delete
				return out, "Deleted", nil
			}
			return out, "Deleting", nil
		},

		Timeout:    10 * time.Minute,
		MinTimeout: 3 * time.Second,
		Delay:      5 * time.Second,
	}

	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("error waiting for group (%s) to become deleted: %s", d.Id(), err)
	}

	// If permanent deletion is selected, issue a second "permanently delete" API call
	if d.Get("permanently_remove_on_delete").(bool) && d.Get("full_path").(string) != "" {
		tflog.Debug(ctx, "Attempting to permanently delete the group", map[string]any{
			"group": d.Get("full_path").(string),
		})

		opts := &gitlab.DeleteGroupOptions{}
		opts.PermanentlyRemove = gitlab.Ptr(d.Get("permanently_remove_on_delete").(bool))
		opts.FullPath = gitlab.Ptr(d.Get("full_path").(string))

		_, err = client.Groups.DeleteGroup(d.Id(), opts, gitlab.WithContext(ctx))
		if err != nil {
			return diag.Errorf("group (%s) was marked for deletion, but permanent deletion of the group failed.", d.Id())
		}

		// Group Deletion happens in the background, so we should wait until we get a 404 when reading the group:
		// Wait for the group to be deleted.
		stateConf := &retry.StateChangeConf{
			Pending: []string{"Deleting"},
			Target:  []string{"Deleted"},
			Refresh: func() (any, string, error) {
				out, response, err := client.Groups.GetGroup(d.Id(), nil, gitlab.WithContext(ctx))
				if err != nil {
					if response != nil && response.StatusCode == 404 {
						return out, "Deleted", nil
					}
					tflog.Error(ctx, "Received error", map[string]any{
						"error": err,
					})
					return out, "Error", err
				}
				return out, "Deleting", nil
			},

			Timeout:    10 * time.Minute,
			MinTimeout: 3 * time.Second,
			Delay:      5 * time.Second,
		}
		_, err = stateConf.WaitForStateContext(ctx)
		if err != nil {
			return diag.Errorf("error waiting for group (%s) to become permanently deleted: %s", d.Id(), err)
		}
	}

	return nil
}

func editOrAddGroupPushRules(ctx context.Context, client *gitlab.Client, groupID string, d *schema.ResourceData) error {
	tflog.Debug(ctx, "[DEBUG] Editing push rules for group", map[string]any{
		"group_id": groupID,
	})

	pushRules, _, err := client.Groups.GetGroupPushRules(d.Id(), gitlab.WithContext(ctx))
	// NOTE: push rules id `0` indicates that there haven't been any push rules set.
	if err != nil || pushRules.ID == 0 {
		addOptions, err := expandAddGroupPushRuleOptions(ctx, client, d)
		if err != nil {
			return err
		}
		if (gitlab.AddGroupPushRuleOptions{}) != addOptions {
			tflog.Debug(ctx, "[DEBUG] Creating new push rules for group", map[string]any{
				"group_id": groupID,
			})
			_, _, err = client.Groups.AddGroupPushRule(groupID, &addOptions, gitlab.WithContext(ctx))
			if err != nil {
				return err
			}
		} else {
			tflog.Debug(ctx, "[DEBUG] Don't create new push rules for defaults for group", map[string]any{
				"group_id": groupID,
			})
		}

		return nil
	}

	editOptions, err := expandEditGroupPushRuleOptions(ctx, client, d)
	if err != nil {
		return err
	}
	if (gitlab.EditGroupPushRuleOptions{}) != editOptions {
		tflog.Debug(ctx, "[DEBUG] Editing existing push rules for group", map[string]any{
			"group_id": groupID,
		})
		_, _, err = client.Groups.EditGroupPushRule(groupID, &editOptions, gitlab.WithContext(ctx))
		if err != nil {
			return err
		}
	} else {
		tflog.Debug(ctx, "[DEBUG] Don't edit existing push rules for defaults for group", map[string]any{
			"group_id": groupID,
		})
	}

	return nil
}

func expandDefaultBranchProtectionDefaults(d *schema.ResourceData) gitlab.DefaultBranchProtectionDefaultsOptions {
	options := gitlab.DefaultBranchProtectionDefaultsOptions{}

	options.AllowedToPush = gitlab.Ptr(convertAccessLevelNamesToValues(d.Get("default_branch_protection_defaults.0.allowed_to_push").([]any)))
	options.AllowForcePush = gitlab.Ptr(d.Get("default_branch_protection_defaults.0.allow_force_push").(bool))
	options.AllowedToMerge = gitlab.Ptr(convertAccessLevelNamesToValues(d.Get("default_branch_protection_defaults.0.allowed_to_merge").([]any)))
	options.DeveloperCanInitialPush = gitlab.Ptr(d.Get("default_branch_protection_defaults.0.developer_can_initial_push").(bool))

	return options
}

func expandEditGroupPushRuleOptions(ctx context.Context, client *gitlab.Client, d *schema.ResourceData) (gitlab.EditGroupPushRuleOptions, error) {
	options := gitlab.EditGroupPushRuleOptions{}

	if d.HasChange("push_rules.0.commit_committer_check") {
		options.CommitCommitterCheck = gitlab.Ptr(d.Get("push_rules.0.commit_committer_check").(bool))
	}

	if d.HasChange("push_rules.0.reject_unsigned_commits") {
		options.RejectUnsignedCommits = gitlab.Ptr(d.Get("push_rules.0.reject_unsigned_commits").(bool))
	}

	if d.HasChange("push_rules.0.reject_non_dco_commits") {
		options.RejectNonDCOCommits = gitlab.Ptr(d.Get("push_rules.0.reject_non_dco_commits").(bool))
	}

	if d.HasChange("push_rules.0.author_email_regex") {
		options.AuthorEmailRegex = gitlab.Ptr(d.Get("push_rules.0.author_email_regex").(string))
	}

	if d.HasChange("push_rules.0.branch_name_regex") {
		options.BranchNameRegex = gitlab.Ptr(d.Get("push_rules.0.branch_name_regex").(string))
	}

	if d.HasChange("push_rules.0.commit_message_regex") {
		options.CommitMessageRegex = gitlab.Ptr(d.Get("push_rules.0.commit_message_regex").(string))
	}

	if d.HasChange("push_rules.0.commit_message_negative_regex") {
		options.CommitMessageNegativeRegex = gitlab.Ptr(d.Get("push_rules.0.commit_message_negative_regex").(string))
	}

	if d.HasChange("push_rules.0.file_name_regex") {
		options.FileNameRegex = gitlab.Ptr(d.Get("push_rules.0.file_name_regex").(string))
	}

	if d.HasChange("push_rules.0.commit_committer_name_check") {
		options.CommitCommitterNameCheck = gitlab.Ptr(d.Get("push_rules.0.commit_committer_name_check").(bool))
	}

	if d.HasChange("push_rules.0.deny_delete_tag") {
		options.DenyDeleteTag = gitlab.Ptr(d.Get("push_rules.0.deny_delete_tag").(bool))
	}

	if d.HasChange("push_rules.0.member_check") {
		options.MemberCheck = gitlab.Ptr(d.Get("push_rules.0.member_check").(bool))
	}

	if d.HasChange("push_rules.0.prevent_secrets") {
		options.PreventSecrets = gitlab.Ptr(d.Get("push_rules.0.prevent_secrets").(bool))
	}

	if d.HasChange("push_rules.0.max_file_size") {
		options.MaxFileSize = gitlab.Ptr(d.Get("push_rules.0.max_file_size").(int))
	}

	return options, nil
}

func expandAddGroupPushRuleOptions(ctx context.Context, client *gitlab.Client, d *schema.ResourceData) (gitlab.AddGroupPushRuleOptions, error) {
	options := gitlab.AddGroupPushRuleOptions{}

	if v, ok := d.GetOk("push_rules.0.commit_committer_check"); ok {
		options.CommitCommitterCheck = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.reject_unsigned_commits"); ok {
		options.RejectUnsignedCommits = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.reject_non_dco_commits"); ok {
		options.RejectNonDCOCommits = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.author_email_regex"); ok {
		options.AuthorEmailRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("push_rules.0.branch_name_regex"); ok {
		options.BranchNameRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("push_rules.0.commit_message_regex"); ok {
		options.CommitMessageRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("push_rules.0.commit_message_negative_regex"); ok {
		options.CommitMessageNegativeRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("push_rules.0.file_name_regex"); ok {
		options.FileNameRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("push_rules.0.commit_committer_name_check"); ok {
		options.CommitCommitterNameCheck = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.deny_delete_tag"); ok {
		options.DenyDeleteTag = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.member_check"); ok {
		options.MemberCheck = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.prevent_secrets"); ok {
		options.PreventSecrets = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.max_file_size"); ok {
		options.MaxFileSize = gitlab.Ptr(v.(int))
	}

	return options, nil
}

func flattenGroupPushRules(ctx context.Context, client *gitlab.Client, pushRules *gitlab.GroupPushRules) (values []map[string]any, err error) {
	if pushRules == nil {
		return []map[string]any{}, nil
	}

	values = []map[string]any{
		{
			"author_email_regex":            pushRules.AuthorEmailRegex,
			"branch_name_regex":             pushRules.BranchNameRegex,
			"commit_message_regex":          pushRules.CommitMessageRegex,
			"commit_message_negative_regex": pushRules.CommitMessageNegativeRegex,
			"file_name_regex":               pushRules.FileNameRegex,
			"commit_committer_check":        pushRules.CommitCommitterCheck,
			"commit_committer_name_check":   pushRules.CommitCommitterNameCheck,
			"deny_delete_tag":               pushRules.DenyDeleteTag,
			"member_check":                  pushRules.MemberCheck,
			"prevent_secrets":               pushRules.PreventSecrets,
			"reject_unsigned_commits":       pushRules.RejectUnsignedCommits,
			"reject_non_dco_commits":        pushRules.RejectNonDCOCommits,
			"max_file_size":                 pushRules.MaxFileSize,
		},
	}

	return values, nil
}
