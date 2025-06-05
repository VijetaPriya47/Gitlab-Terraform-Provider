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

var _ = registerDataSource("gitlab_user", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_user`" + ` data source allows details of a user to be retrieved by either the user ID, username or email address.

-> Some attributes might not be returned depending on if you're an admin or not.

~> When using the ` + "`email`" + ` attribute, an exact match is not guaranteed. The most related match will be returned. The most related match will prioritize an exact match if one is available.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/users/#get-a-single-user)`,

		ReadContext: dataSourceGitlabUserRead,
		Schema: map[string]*schema.Schema{
			"user_id": {
				Description: "The ID of the user.",
				Type:        schema.TypeInt,
				Computed:    true,
				Optional:    true,
				ConflictsWith: []string{
					"username",
					"email",
				},
			},
			"username": {
				Description: "The username of the user.",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				ConflictsWith: []string{
					"user_id",
					"email",
				},
			},
			"email": {
				Description: "The public email address of the user.",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				ConflictsWith: []string{
					"user_id",
					"username",
				},
			},
			"name": {
				Description: "The name of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"is_admin": {
				Description: "Whether the user is an admin.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"can_create_group": {
				Description: "Whether the user can create groups.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"can_create_project": {
				Description: "Whether the user can create projects.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"projects_limit": {
				Description: "Number of projects the user can create.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"created_at": {
				Description: "Date the user was created at.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"state": {
				Description: "Whether the user is active or blocked.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"external": {
				Description: "Whether the user is external.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"extern_uid": {
				Description: "The external UID of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"organization": {
				Description: "The organization of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"two_factor_enabled": {
				Description: "Whether user's two-factor auth is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"note": {
				Description: "Admin notes for this user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"user_provider": {
				Description: "The UID provider of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"avatar_url": {
				Description: "The avatar URL of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"bio": {
				Description: "The bio of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"is_bot": {
				Description: "Whether the user is a bot.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"location": {
				Description: "The location of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"skype": {
				Description: "Skype username of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"linkedin": {
				Description: "LinkedIn profile of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"twitter": {
				Description: "Twitter username of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"website_url": {
				Description: "User's website URL.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"theme_id": {
				Description: "User's theme ID.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"color_scheme_id": {
				Description: "User's color scheme ID.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"last_sign_in_at": {
				Description: "Last user's sign-in date.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"current_sign_in_at": {
				Description: "Current user's sign-in date.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"namespace_id": {
				Description: "The ID of the user's namespace. Requires admin token to access this field.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"email_exact_match": {
				Description: "(Experimental) If true, returns only an exact match. Otherwise, fuzzy matching might return the closest result. If no exact match is available, the data source returns an error.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ConflictsWith: []string{
					"user_id",
					"username",
				},
			},
		},
	}
})

func dataSourceGitlabUserRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	var user *gitlab.User
	var err error

	tflog.Info(ctx, "[INFO] Reading Gitlab user")

	userIDData, userIDOk := d.GetOk("user_id")
	usernameData, usernameOk := d.GetOk("username")
	emailData, emailOk := d.GetOk("email")
	emailExactMatch, emailExactMatchOk := d.GetOk("email_exact_match")

	if userIDOk {
		// Get user by id
		user, _, err = client.Users.GetUser(userIDData.(int), gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	} else if usernameOk || emailOk {
		username := strings.ToLower(usernameData.(string))
		email := strings.ToLower(emailData.(string))

		listUsersOptions := &gitlab.ListUsersOptions{}
		if usernameOk {
			// Get user by username
			listUsersOptions.Username = gitlab.Ptr(username)
		} else {
			// Get user by email
			// Note: Search can return multiple users potentially, but as of GitLab 16.6,
			// useing Search without "sort" will prioritize an exact match at the top
			// of the list.
			listUsersOptions.Search = gitlab.Ptr(email)
		}

		var users []*gitlab.User
		users, _, err = client.Users.ListUsers(listUsersOptions, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		if len(users) > 0 {
			if emailOk && emailExactMatchOk && emailExactMatch.(bool) {
				for _, v := range users {
					if v.Email == email {
						user = v
						break
					}
				}
			} else {
				if len(users) > 1 {
					tflog.Info(ctx, "more than one user found matching defined email. Will return the first user, since this can only happen when using `search`", map[string]interface{}{
						"email": email,
					})
				}
				user = users[0]
			}
		}
		if user == nil {
			return diag.Errorf("couldn't find a user matching: %s%s", username, email)
		}
	} else {
		return diag.Errorf("one and only one of user_id, username or email must be set")
	}

	d.SetId(fmt.Sprintf("%d", user.ID))
	d.Set("user_id", user.ID)
	d.Set("username", user.Username)
	d.Set("email", user.Email)
	d.Set("name", user.Name)
	d.Set("is_admin", user.IsAdmin)
	d.Set("can_create_group", user.CanCreateGroup)
	d.Set("can_create_project", user.CanCreateProject)
	d.Set("projects_limit", user.ProjectsLimit)
	d.Set("state", user.State)
	d.Set("external", user.External)
	d.Set("extern_uid", user.ExternUID)
	d.Set("is_bot", user.Bot)

	if user.CreatedAt != nil {
		d.Set("created_at", user.CreatedAt.String())
	} else {
		d.Set("created_at", "")
	}

	d.Set("organization", user.Organization)
	d.Set("two_factor_enabled", user.TwoFactorEnabled)
	d.Set("note", user.Note)
	d.Set("user_provider", user.Provider)
	d.Set("avatar_url", user.AvatarURL)
	d.Set("bio", user.Bio)
	d.Set("location", user.Location)
	d.Set("skype", user.Skype)
	d.Set("linkedin", user.Linkedin)
	d.Set("twitter", user.Twitter)
	d.Set("website_url", user.WebsiteURL)
	d.Set("theme_id", user.ThemeID)
	d.Set("color_scheme_id", user.ColorSchemeID)
	d.Set("namespace_id", user.NamespaceID)

	return nil
}
