package sdk

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	allowedToCreateElem = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"access_level": {
				Description: "Level of access.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"access_level_description": {
				Description: "Readable description of level of access.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"user_id": {
				Description: "The ID of a GitLab user allowed to perform the relevant action. Mutually exclusive with `group_id`.",
				Type:        schema.TypeInt,
				ForceNew:    true,
				Optional:    true,
			},
			"group_id": {
				Description: "The ID of a GitLab group allowed to perform the relevant action. Mutually exclusive with `user_id`.",
				Type:        schema.TypeInt,
				ForceNew:    true,
				Optional:    true,
			},
		},
	}
)

var _ = registerResource("gitlab_tag_protection", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_tag_protection`" + ` resource allows to manage the lifecycle of a tag protection.

~> As tag protections cannot be updated, they are deleted and recreated when a change is requested. This means that if the deletion succeeds but the creation fails, tags will be left unprotected.
If this is a potential issue for you, please use the ` + "`create_before_destroy`" + ` meta-argument: https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/protected_tags.html)`,

		CreateContext: resourceGitlabTagProtectionCreate,
		ReadContext:   resourceGitlabTagProtectionRead,
		DeleteContext: resourceGitlabTagProtectionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"project": {
				Description: "The id of the project.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"tag": {
				Description: "Name of the tag or wildcard.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"create_access_level": {
				Description:      fmt.Sprintf("Access levels which are allowed to create. Valid values are: %s.", utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(api.ValidProtectedBranchTagAccessLevelNames, false)),
				Required:         true,
				ForceNew:         true,
			},
			"allowed_to_create": schemaAllowedToCreate(),
		},
	}
})

func schemaAllowedToCreate() *schema.Schema {
	return &schema.Schema{
		Description: "User or group which are allowed to create.",
		Type:        schema.TypeSet,
		Elem:        allowedToCreateElem,
		Optional:    true,
		ForceNew:    true,
	}
}

func resourceGitlabTagProtectionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	tag := gitlab.Ptr(d.Get("tag").(string))
	createAccessLevel := tagProtectionAccessLevelID[d.Get("create_access_level").(string)]

	allowedToCreate, err := expandTagPermissionOptions(d.Get("allowed_to_create").(*schema.Set).List())

	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.ProtectRepositoryTagsOptions{
		Name:              tag,
		CreateAccessLevel: &createAccessLevel,
		AllowedToCreate:   &allowedToCreate,
	}

	log.Printf("[DEBUG] create gitlab tag protection on %v for project %s", options.Name, project)

	tp, _, err := client.ProtectedTags.ProtectRepositoryTags(project, options, gitlab.WithContext(ctx))
	if err != nil {
		// Remove existing tag protection
		_, err = client.ProtectedTags.UnprotectRepositoryTags(project, *tag, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
		// Reprotect tag with updated values
		tp, _, err = client.ProtectedTags.ProtectRepositoryTags(project, options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// If allowed_to_create has been set but didn't come back, it means it's not supported under this license
	if len(allowedToCreate) > 0 {
		// Fastest way to do that: sum all userId and groupIds, if the sum is still 0 at the end,
		// then it means no rights were added to individual users or groups
		sum := 0
		for _, cal := range tp.CreateAccessLevels {
			sum += cal.UserID + cal.GroupID
		}
		if sum == 0 {
			return diag.Errorf("feature unavailable: `allowed_to_create`, Premium or Ultimate license required.")
		}
	}

	d.SetId(utils.BuildTwoPartID(&project, &tp.Name))

	return resourceGitlabTagProtectionRead(ctx, d, meta)
}

func resourceGitlabTagProtectionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, tag, err := projectAndTagFromID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] read gitlab tag protection for project %s, tag %s", project, tag)

	pt, _, err := client.ProtectedTags.GetProtectedTag(project, tag, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			log.Printf("[DEBUG] gitlab tag protection not found %s/%s", project, tag)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	accessLevel, ok := tagProtectionAccessLevelNames[getProtectedTagAccessLevel(pt)]
	if !ok {
		return diag.Errorf("tag protection access level %d is not supported. Supported are: %v", pt.CreateAccessLevels[0].AccessLevel, tagProtectionAccessLevelNames)
	}

	d.Set("project", project)
	d.Set("tag", pt.Name)
	d.Set("create_access_level", accessLevel)

	if err := d.Set("allowed_to_create", flattenNonZeroTagAccessDescriptions(pt.CreateAccessLevels)); err != nil {
		return diag.Errorf("error setting allowed_to_create: %v", err)
	}

	d.SetId(utils.BuildTwoPartID(&project, &pt.Name))

	return nil
}

func resourceGitlabTagProtectionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	tag := d.Get("tag").(string)

	log.Printf("[DEBUG] Delete gitlab protected tag %s for project %s", tag, project)

	_, err := client.ProtectedTags.UnprotectRepositoryTags(project, tag, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func projectAndTagFromID(id string) (string, string, error) {
	project, tag, err := utils.ParseTwoPartID(id)

	if err != nil {
		log.Printf("[WARN] cannot get group member id from input: %v", id)
	}
	return project, tag, err
}

func expandTagPermissionOptions(allowedTo []interface{}) ([]*gitlab.TagsPermissionOptions, error) {
	result := make([]*gitlab.TagsPermissionOptions, 0)
	for _, v := range allowedTo {
		opt := &gitlab.TagsPermissionOptions{}
		if userID, ok := v.(map[string]interface{})["user_id"]; ok && userID != 0 {
			opt.UserID = gitlab.Ptr(userID.(int))
		}
		if groupID, ok := v.(map[string]interface{})["group_id"]; ok && groupID != 0 {
			opt.GroupID = gitlab.Ptr(groupID.(int))
		}
		if opt.UserID != nil && opt.GroupID != nil {
			return nil, fmt.Errorf("both user_id and group_id cannot be present in the same allowed_to_create")
		}
		result = append(result, opt)
	}
	return result, nil
}

func flattenNonZeroTagAccessDescriptions(descriptions []*gitlab.TagAccessDescription) (values []map[string]interface{}) {
	for _, description := range descriptions {
		if description.UserID == 0 && description.GroupID == 0 {
			continue
		}
		values = append(values, map[string]interface{}{
			"access_level":             api.AccessLevelValueToName[description.AccessLevel],
			"access_level_description": description.AccessLevelDescription,
			"user_id":                  description.UserID,
			"group_id":                 description.GroupID,
		})
	}

	return values
}

// The "base" access level is the access level that doesn't have a user or group assigned to it.
// all access levels specified within an "allowed_to_create" block will have one of those two attributes specified.
func getProtectedTagAccessLevel(pt *gitlab.ProtectedTag) gitlab.AccessLevelValue {
	for _, cal := range pt.CreateAccessLevels {
		if cal.UserID == 0 && cal.GroupID == 0 {
			return cal.AccessLevel
		}
	}
	return gitlab.NoPermissions
}
