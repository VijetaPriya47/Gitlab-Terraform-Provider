package sdk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_project_share_group", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_project_share_group` + "`" + ` resource allows to manage the lifecycle of project shared with a group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/projects.html#share-project-with-group)`,

		CreateContext: resourceGitlabProjectShareGroupCreate,
		ReadContext:   resourceGitlabProjectShareGroupRead,
		DeleteContext: resourceGitlabProjectShareGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema:        gitlabProjectShareGroupSchema(),
		SchemaVersion: 2,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceGitlabProjectShareGroupResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabProjectShareGroupStateUpgradeV0,
				Version: 0,
			},
			{
				Type:    resourceGitlabProjectShareGroupResourceV1().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceGitlabProjectShareGroupStateUpgradeV1,
				Version: 1,
			},
		},
	}
})

func gitlabProjectShareGroupSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"project": {
			Description: "The ID or URL-encoded path of the project.",
			Type:        schema.TypeString,
			ForceNew:    true,
			Required:    true,
		},
		"group_id": {
			Description: "The id of the group.",
			Type:        schema.TypeInt,
			ForceNew:    true,
			Required:    true,
		},
		"group_access": {
			Description:      fmt.Sprintf("The access level to grant the group for the project. Valid values are: %s", utils.RenderValueListForDocs(api.ValidProjectAccessLevelNames)),
			Type:             schema.TypeString,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(api.ValidProjectAccessLevelNames, false)),
			ForceNew:         true,
			Optional:         true,
			ExactlyOneOf:     []string{"access_level", "group_access"},
		},
		"access_level": {
			Description:      fmt.Sprintf("The access level to grant the group for the project. Valid values are: %s", utils.RenderValueListForDocs(api.ValidProjectAccessLevelNames)),
			Type:             schema.TypeString,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(api.ValidProjectAccessLevelNames, false)),
			ForceNew:         true,
			Optional:         true,
			Deprecated:       "Use `group_access` instead of the `access_level` attribute.",
			ExactlyOneOf:     []string{"access_level", "group_access"},
		},
	}
}

func resourceGitlabProjectShareGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	groupId := d.Get("group_id").(int)
	project := d.Get("project").(string)

	var groupAccess gitlab.AccessLevelValue
	if v, ok := d.GetOk("group_access"); ok {
		groupAccess = gitlab.AccessLevelValue(api.AccessLevelNameToValue[v.(string)])
	} else if v, ok := d.GetOk("access_level"); ok {
		groupAccess = gitlab.AccessLevelValue(api.AccessLevelNameToValue[v.(string)])
	} else {
		return diag.Errorf("Neither `group_access` nor `access_level` (deprecated) is set")
	}

	options := &gitlab.ShareWithGroupOptions{
		GroupID:     &groupId,
		GroupAccess: &groupAccess,
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab project membership for %d in %s", options.GroupID, project))

	_, err := client.Projects.ShareProjectWithGroup(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}
	groupIdString := strconv.Itoa(groupId)
	d.SetId(utils.BuildTwoPartID(&project, &groupIdString))
	return resourceGitlabProjectShareGroupRead(ctx, d, meta)
}

func resourceGitlabProjectShareGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	id := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab project projectMember %s", id))

	project, groupId, err := projectAndGroupIdFromId(id)
	if err != nil {
		return diag.FromErr(err)
	}

	projectInformation, _, err := client.Projects.GetProject(project, nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] failed to read gitlab project %s: %s", id, err))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	foundGroup := false
	for _, v := range projectInformation.SharedWithGroups {
		if groupId == v.GroupID {
			resourceGitlabProjectShareGroupSetToState(d, v, &project)
			foundGroup = true
			break
		}
	}
	// If we didn't find our group, we need to remove it from state
	if !foundGroup {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Gitlab project group share not found for group %v and project %s; removing from state.", groupId, project))
		d.SetId("")
	}

	return nil
}

func projectAndGroupIdFromId(id string) (string, int, error) {
	project, groupIdString, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, fmt.Errorf("error parsing ID: %s", id)
	}

	groupId, err := strconv.Atoi(groupIdString)
	if err != nil {
		return "", 0, fmt.Errorf("can not determine group id: %v", id)
	}

	return project, groupId, nil
}

func resourceGitlabProjectShareGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	id := d.Id()
	projectId, groupId, err := projectAndGroupIdFromId(id)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab project membership %v for %s", groupId, projectId))

	_, err = client.Projects.DeleteSharedProjectFromGroup(projectId, groupId, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitlabProjectShareGroupSetToState(d *schema.ResourceData, group struct {
	GroupID          int    `json:"group_id"`
	GroupName        string `json:"group_name"`
	GroupFullPath    string `json:"group_full_path"`
	GroupAccessLevel int    `json:"group_access_level"`
}, projectId *string) {

	//This cast is needed due to an inconsistency in the upstream API
	//GroupAccessLevel is returned as an int but the map we lookup is sorted by the int alias AccessLevelValue
	convertedAccessLevel := gitlab.AccessLevelValue(group.GroupAccessLevel)

	d.Set("project", projectId)
	d.Set("group_id", group.GroupID)
	d.Set("group_access", api.AccessLevelValueToName[convertedAccessLevel])

	groupId := strconv.Itoa(group.GroupID)
	d.SetId(utils.BuildTwoPartID(projectId, &groupId))
}

func resourceGitlabProjectShareGroupResourceV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			// This should explicitly be left as "project_id" instead of "project",
			// as it's meant to be a point-in-time of the schema at the time.
			"project_id": {
				Description: "The id of the project.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"group_id": {
				Description: "The id of the group.",
				Type:        schema.TypeInt,
				ForceNew:    true,
				Required:    true,
			},
			"access_level": {
				Description:      fmt.Sprintf("The access level to grant the group for the project. Valid values are: %s", utils.RenderValueListForDocs(api.ValidProjectAccessLevelNames)),
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(api.ValidProjectAccessLevelNames, false)),
				ForceNew:         true,
				Required:         true,
			},
		},
	}
}

func resourceGitlabProjectShareGroupStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	rawState["group_access"] = rawState["access_level"]
	delete(rawState, "access_level")
	return rawState, nil
}

func resourceGitlabProjectShareGroupResourceV1() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"project_id": {
				Description: "The ID of the project.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"group_id": {
				Description: "The id of the group.",
				Type:        schema.TypeInt,
				ForceNew:    true,
				Required:    true,
			},
			"group_access": {
				Description:      fmt.Sprintf("The access level to grant the group for the project. Valid values are: %s", utils.RenderValueListForDocs(api.ValidProjectAccessLevelNames)),
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(api.ValidProjectAccessLevelNames, false)),
				ForceNew:         true,
				Optional:         true,
				ExactlyOneOf:     []string{"access_level", "group_access"},
			},
			"access_level": {
				Description:      fmt.Sprintf("The access level to grant the group for the project. Valid values are: %s", utils.RenderValueListForDocs(api.ValidProjectAccessLevelNames)),
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(api.ValidProjectAccessLevelNames, false)),
				ForceNew:         true,
				Optional:         true,
				Deprecated:       "Use `group_access` instead of the `access_level` attribute.",
				ExactlyOneOf:     []string{"access_level", "group_access"},
			},
		},
	}
}

// resourceGitlabProjectShareGroupStateUpgradeV1 performs the state migration from V1 to V2.
func resourceGitlabProjectShareGroupStateUpgradeV1(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	if rawState["project_id"] != nil {
		projectId, ok := rawState["project_id"].(string)
		if !ok {
			projectId = strconv.FormatInt(int64(rawState["project_id"].(float64)), 10)
		}
		rawState["project"] = projectId
		delete(rawState, "project_id")
		tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `project_id` attribute to `project`", map[string]interface{}{"project_id": projectId})
	}
	return rawState, nil
}
