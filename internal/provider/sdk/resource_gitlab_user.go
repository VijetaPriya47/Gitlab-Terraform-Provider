package sdk

import (
	"context"
	"fmt"
	"strconv"
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

var validUserStateValues = []string{
	"active",
	"deactivated",
	"blocked",
}

var _ = registerResource("gitlab_user", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_user`" + ` resource allows to manage the lifecycle of a user.

-> the provider needs to be configured with admin-level access for this resource to work.

-> You must specify either password or reset_password.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/users/)`,

		CreateContext: resourceGitlabUserCreate,
		ReadContext:   resourceGitlabUserRead,
		UpdateContext: resourceGitlabUserUpdate,
		DeleteContext: resourceGitlabUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"username": {
				Description: "The username of the user.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"password": {
				Description: "The password of the user.",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				ForceNew:    true,
				ConflictsWith: []string{
					"force_random_password",
				},
			},
			"force_random_password": {
				Description: "Set user password to a random value",
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				ConflictsWith: []string{
					"password",
				},
			},
			"email": {
				Description: "The e-mail address of the user.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "The name of the user.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"is_admin": {
				Description: "Boolean, defaults to false.  Whether to enable administrative privileges",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"can_create_group": {
				Description: "Boolean, defaults to false. Whether to allow the user to create groups.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"skip_confirmation": {
				Description: "Boolean, defaults to true. Whether to skip confirmation.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Skip Confirmation doesn't come back from the API, so this will always return a diff unless this skips.

					// Should only return true on non-create actions (I.e., after the ID is set). Otherwise the schema always
					// gets "false" instead of whatever is set.
					return d.Id() != ""
				},
			},
			"projects_limit": {
				Description: "Integer, defaults to 0.  Number of projects user can create.",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
			},
			"is_external": {
				Description: "Boolean, defaults to false. Whether a user has access only to some internal or private projects. External users can only access projects to which they are explicitly granted access.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"reset_password": {
				Description: "Boolean, defaults to false. Send user password reset link.",
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
			},
			"note": {
				Description: "The note associated to the user.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"state": {
				Description:      fmt.Sprintf("String, defaults to 'active'. The state of the user account. Valid values are %s.", utils.RenderValueListForDocs(validUserStateValues)),
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "active",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validUserStateValues, false)),
			},
			"namespace_id": {
				Description: "The ID of the user's namespace.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
		},
	}
})

func resourceGitlabUserSetToState(d *schema.ResourceData, user *gitlab.User) {
	d.Set("username", user.Username)
	d.Set("name", user.Name)
	d.Set("can_create_group", user.CanCreateGroup)
	d.Set("projects_limit", user.ProjectsLimit)
	d.Set("email", user.Email)
	d.Set("is_admin", user.IsAdmin)
	d.Set("is_external", user.External)
	d.Set("note", user.Note)
	d.Set("state", user.State)
	d.Set("namespace_id", user.NamespaceID)
}

func resourceGitlabUserCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	options := &gitlab.CreateUserOptions{
		Email:               gitlab.Ptr(d.Get("email").(string)),
		Username:            gitlab.Ptr(d.Get("username").(string)),
		Name:                gitlab.Ptr(d.Get("name").(string)),
		ProjectsLimit:       gitlab.Ptr(int64(d.Get("projects_limit").(int))),
		Admin:               gitlab.Ptr(d.Get("is_admin").(bool)),
		CanCreateGroup:      gitlab.Ptr(d.Get("can_create_group").(bool)),
		SkipConfirmation:    gitlab.Ptr(d.Get("skip_confirmation").(bool)),
		External:            gitlab.Ptr(d.Get("is_external").(bool)),
		ResetPassword:       gitlab.Ptr(d.Get("reset_password").(bool)),
		ForceRandomPassword: gitlab.Ptr(d.Get("force_random_password").(bool)),
		Note:                gitlab.Ptr(d.Get("note").(string)),
	}

	if len(d.Get("password").(string)) != 0 {
		options.Password = gitlab.Ptr(d.Get("password").(string))
	}

	// Validate the options set
	if (options.Password == nil || *options.Password == "") &&
		(options.ResetPassword == nil || !*options.ResetPassword) &&
		(options.ForceRandomPassword == nil || !*options.ForceRandomPassword) {
		return diag.Errorf(`At least one of "password", "reset_password", or "force_random_password" must be set`)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab user %q", *options.Username))

	user, _, err := client.Users.CreateUser(options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", user.ID))

	if d.Get("state") == "blocked" {
		err := client.Users.BlockUser(user.ID, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	} else if d.Get("state") == "deactivated" {
		err := client.Users.DeactivateUser(user.ID, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceGitlabUserRead(ctx, d, meta)
}

func resourceGitlabUserRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] import -- read gitlab user %s", d.Id()))

	id, _ := strconv.ParseInt(d.Id(), 10, 64)

	user, _, err := client.Users.GetUser(id, gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab user not found %d", id))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	resourceGitlabUserSetToState(d, user)
	return nil
}

func resourceGitlabUserUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	options := &gitlab.ModifyUserOptions{}

	if d.HasChange("name") {
		options.Name = gitlab.Ptr(d.Get("name").(string))
	}

	if d.HasChange("username") {
		options.Username = gitlab.Ptr(d.Get("username").(string))
	}

	if d.HasChange("email") {
		options.Email = gitlab.Ptr(d.Get("email").(string))
		options.SkipReconfirmation = gitlab.Ptr(true)
	}

	if d.HasChange("is_admin") {
		options.Admin = gitlab.Ptr(d.Get("is_admin").(bool))
	}

	if d.HasChange("can_create_group") {
		options.CanCreateGroup = gitlab.Ptr(d.Get("can_create_group").(bool))
	}

	if d.HasChange("projects_limit") {
		options.ProjectsLimit = gitlab.Ptr(int64(d.Get("projects_limit").(int)))
	}

	if d.HasChange("is_external") {
		options.External = gitlab.Ptr(d.Get("is_external").(bool))
	}

	if d.HasChange("note") {
		options.Note = gitlab.Ptr(d.Get("note").(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab user %s", d.Id()))

	id, _ := strconv.ParseInt(d.Id(), 10, 64)

	_, _, err := client.Users.ModifyUser(id, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("state") {
		oldState, newState := d.GetChange("state")
		var err error
		// NOTE: yes, this can be written much more consice, however, for the sake of understanding the behavior,
		//       of the API and the allowed state transitions of GitLab, let's keep it as-is and enjoy the readability.
		if newState == "active" && oldState == "blocked" {
			err = client.Users.UnblockUser(id, gitlab.WithContext(ctx))
		} else if newState == "active" && oldState == "deactivated" {
			err = client.Users.ActivateUser(id, gitlab.WithContext(ctx))
		} else if newState == "blocked" && oldState == "active" {
			err = client.Users.BlockUser(id, gitlab.WithContext(ctx))
		} else if newState == "blocked" && oldState == "deactivated" {
			err = client.Users.BlockUser(id, gitlab.WithContext(ctx))
		} else if newState == "deactivated" && oldState == "active" {
			err = client.Users.DeactivateUser(id, gitlab.WithContext(ctx))
		} else if newState == "deactivated" && oldState == "blocked" {
			// a blocked user cannot be deactivated, GitLab will return an error, like:
			// `403 Forbidden - A blocked user cannot be deactivated by the API`
			// we have to unblock the user first
			err = client.Users.UnblockUser(id, gitlab.WithContext(ctx))
			if err != nil {
				return diag.FromErr(err)
			}
			err = client.Users.DeactivateUser(id, gitlab.WithContext(ctx))
		}

		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceGitlabUserRead(ctx, d, meta)
}

func resourceGitlabUserDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab user %s", d.Id()))

	id, _ := strconv.ParseInt(d.Id(), 10, 64)

	if _, err := client.Users.DeleteUser(id, gitlab.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	stateConf := &retry.StateChangeConf{
		Timeout: 10 * time.Minute,
		Target:  []string{"Deleted"},
		Refresh: func() (any, string, error) {
			user, resp, err := client.Users.GetUser(id, gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
			if resp != nil && resp.StatusCode == 404 {
				return user, "Deleted", nil
			}
			if err != nil {
				return user, "Error", err
			}
			return user, "Deleting", nil
		},
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return diag.Errorf("Could not finish deleting user %d: %s", id, err)
	}

	return nil
}
