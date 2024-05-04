package sdk

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var validPersonalAccessTokenScopes = []string{
	"api",
	"read_user",
	"read_api",
	"read_repository",
	"write_repository",
	"read_registry",
	"write_registry",
	"sudo",
	"admin_mode",
	"create_runner",
}

var _ = registerResource("gitlab_personal_access_token", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_personal_access_token`" + ` resource allows to manage the lifecycle of a personal access token for a specified user.

-> This resource requires administration privileges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/personal_access_tokens.html)`,

		CreateContext: resourceGitlabPersonalAccessTokenCreate,
		ReadContext:   resourceGitlabPersonalAccessTokenRead,
		DeleteContext: resourceGitlabPersonalAccessTokenDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"user_id": {
				Description: "The id of the user.",
				Type:        schema.TypeInt,
				ForceNew:    true,
				Required:    true,
			},
			"name": {
				Description: "The name of the personal access token.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"scopes": {
				Description: fmt.Sprintf("The scope for the personal access token. It determines the actions which can be performed when authenticating with this token. Valid values are: %s.", utils.RenderValueListForDocs(validPersonalAccessTokenScopes)),
				Type:        schema.TypeSet,
				Required:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(validPersonalAccessTokenScopes, false),
				},
			},
			"active": {
				Description: "True if the token is active.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"revoked": {
				Description: "True if the token is revoked.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"created_at": {
				Description: "Time the token has been created, RFC3339 format.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"expires_at": {
				Description:      "The token expires at midnight UTC on that date. The date must be in the format YYYY-MM-DD.",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: isISO6801Date,
			},
			"token": {
				Description: "The personal access token. This is only populated when creating a new personal access token. This attribute is not available for imported resources.",
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
})

func resourceGitlabPersonalAccessTokenCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	currentUserAdmin, err := isCurrentUserAdmin(ctx, client)
	if err != nil {
		return diag.Errorf("[ERROR] cannot query the user API for current user: %v", err)
	}

	if !currentUserAdmin {
		return diag.Errorf("current user needs to be admin when creating a personal access token")
	}

	options := &gitlab.CreatePersonalAccessTokenOptions{
		Name:   gitlab.Ptr(d.Get("name").(string)),
		Scopes: stringSetToStringSlice(d.Get("scopes").(*schema.Set)),
	}

	userID := d.Get("user_id").(int)
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab PersonalAccessToken %s (scopes: %s) for user ID %d", *options.Name, options.Scopes, userID))

	if v, ok := d.GetOk("expires_at"); ok {
		parsedExpiresAt, err := parseISO8601Date(v.(string))
		if err != nil {
			return diag.Errorf("failed to parse expires_at '%s' as ISO8601 formatted date: %v", v.(string), err)
		}

		options.ExpiresAt = parsedExpiresAt
	}

	personalAccessToken, _, err := client.Users.CreatePersonalAccessToken(userID, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d:%d", userID, personalAccessToken.ID))
	// NOTE: the token can only be read once after creating it
	d.Set("token", personalAccessToken.Token)

	return resourceGitlabPersonalAccessTokenRead(ctx, d, meta)
}

func resourceGitlabPersonalAccessTokenRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	userID, tokenID, err := resourceGitLabPersonalAccessTokenParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab PersonalAccessToken %d, user ID %d", tokenID, userID))

	personalAccessToken, _, err := client.PersonalAccessTokens.GetSinglePersonalAccessTokenByID(tokenID)
	if errors.Is(err, errResourceGitlabPersonalAccessTokenNotFound) {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] failed to read gitlab PersonalAccessToken %d, user ID %d", tokenID, userID))
		d.SetId("")

		return nil
	}

	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("user_id", userID)
	d.Set("name", personalAccessToken.Name)
	if personalAccessToken.ExpiresAt != nil {
		d.Set("expires_at", personalAccessToken.ExpiresAt.String())
	}
	d.Set("active", personalAccessToken.Active)
	d.Set("created_at", personalAccessToken.CreatedAt.Format(time.RFC3339))
	d.Set("revoked", personalAccessToken.Revoked)

	if err = d.Set("scopes", personalAccessToken.Scopes); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitlabPersonalAccessTokenDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	_, tokenID, err := resourceGitLabPersonalAccessTokenParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab PersonalAccessToken %s", d.Id()))
	_, err = client.PersonalAccessTokens.RevokePersonalAccessToken(tokenID, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

var errResourceGitlabPersonalAccessTokenNotFound = errors.New("personal access token not found")

func resourceGitLabPersonalAccessTokenParseId(id string) (int, int, error) {
	userID, tokenID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return 0, 0, err
	}

	userIID, err := strconv.Atoi(userID)
	if err != nil {
		return 0, 0, err
	}

	tokenIID, err := strconv.Atoi(tokenID)
	if err != nil {
		return 0, 0, err
	}

	return userIID, tokenIID, nil
}
