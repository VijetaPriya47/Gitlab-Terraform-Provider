package sdk

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_group_saml_link", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_group_saml_link`" + ` resource manages the lifecycle of an SAML integration with a group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/saml/#saml-group-links)`,

		CreateContext: resourceGitlabGroupSamlLinkCreate,
		ReadContext:   resourceGitlabGroupSamlLinkRead,
		DeleteContext: resourceGitlabGroupSamlLinkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"group": {
				Description: "The ID or path of the group to add the SAML Group Link to.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"saml_group_name": {
				Description: "The name of the SAML group.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"access_level": {
				Description:      fmt.Sprintf("Access level for members of the SAML group. Valid values are: %s.", utils.RenderValueListForDocs(api.ValidGroupSAMLLinkAccessLevelNames)),
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(api.ValidGroupSAMLLinkAccessLevelNames, false)),
				Required:         true,
				ForceNew:         true,
			},
			"member_role_id": {
				Description: "The ID of a custom member role. Only available for Ultimate instances. When using a custom role, the `access_level` must match the base role used to create the custom role.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
})

func resourceGitlabGroupSamlLinkCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	group := d.Get("group").(string)
	samlGroupName := d.Get("saml_group_name").(string)
	accessLevel := api.AccessLevelNameToValue[d.Get("access_level").(string)]

	options := &gitlab.AddGroupSAMLLinkOptions{
		SAMLGroupName: gitlab.Ptr(samlGroupName),
		AccessLevel:   gitlab.Ptr(accessLevel),
	}

	if v, ok := d.GetOk("member_role_id"); v != nil && ok {
		options.MemberRoleID = gitlab.Ptr(int64(v.(int)))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Create GitLab Group SAML Link for group %q with name %q", group, samlGroupName))
	SamlLink, _, err := client.Groups.AddGroupSAMLLink(group, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(utils.BuildTwoPartID(&group, &SamlLink.Name))
	return resourceGitlabGroupSamlLinkRead(ctx, d, meta)
}

func resourceGitlabGroupSamlLinkRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	group, samlGroupName, parse_err := utils.ParseTwoPartID(d.Id())
	if parse_err != nil {
		return diag.FromErr(parse_err)
	}

	// Try to fetch all group links from GitLab
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Read GitLab Group SAML Link for group %q", group))
	samlLink, _, err := client.Groups.GetGroupSAMLLink(group, samlGroupName, nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] GitLab SAML Group Link %s for group ID %s not found, removing from state", samlGroupName, group))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("group", group)
	d.Set("access_level", api.AccessLevelValueToName[samlLink.AccessLevel])
	d.Set("saml_group_name", samlLink.Name)
	d.Set("member_role_id", samlLink.MemberRoleID)

	return nil
}

func resourceGitlabGroupSamlLinkDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	group, samlGroupName, parse_err := utils.ParseTwoPartID(d.Id())
	if parse_err != nil {
		return diag.FromErr(parse_err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete GitLab Group SAML Link for group %q with name %q", group, samlGroupName))
	_, err := client.Groups.DeleteGroupSAMLLink(group, samlGroupName, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, fmt.Sprintf("[WARN] %s", err))
		} else {
			return diag.FromErr(err)
		}
	}

	return nil
}
