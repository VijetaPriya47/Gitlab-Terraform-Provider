//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroupSAMLLinks_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	groupID, developerRoleID, reporterRoleID, err := createDatasourceGroupSAMLLinksTestData(t)
	if err != nil {
		t.Fatalf("could not create test data for datasource SAML group links: %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_group_saml_links" "test" {
					group = %s
				}`, groupID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "id", groupID),
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "group", groupID),
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "saml_links.#", "3"),
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "saml_links.0.name", "saml_group_a"),
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "saml_links.0.access_level", "developer"),
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "saml_links.0.member_role_id", developerRoleID),
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "saml_links.1.name", "saml_group_b"),
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "saml_links.1.access_level", "maintainer"),
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "saml_links.2.name", "saml_group_c"),
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "saml_links.2.access_level", "reporter"),
					resource.TestCheckResourceAttr("data.gitlab_group_saml_links.test", "saml_links.2.member_role_id", reporterRoleID),
				),
			},
		},
	})
}

func createDatasourceGroupSAMLLinksTestData(t *testing.T) (string, string, string, error) {
	// Create an instance role (since group roles don't work on self-hosted)
	rInt := acctest.RandInt()
	developerRole := testutil.CreateCustomInstanceRole(t, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr(fmt.Sprintf("test-role-%d", rInt)),
		BaseAccessLevel:   gitlab.Ptr(gitlab.DeveloperPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})

	reporterRole := testutil.CreateCustomInstanceRole(t, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr(fmt.Sprintf("test-role-two-%d", rInt)),
		BaseAccessLevel:   gitlab.Ptr(gitlab.ReporterPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})
	group := testutil.CreateGroups(t, 1)[0]
	_, _, err := testutil.TestGitlabClient.Groups.AddGroupSAMLLink(group.ID, &gitlab.AddGroupSAMLLinkOptions{
		SAMLGroupName: gitlab.Ptr("saml_group_a"),
		AccessLevel:   gitlab.Ptr(gitlab.DeveloperPermissions),
		MemberRoleID:  gitlab.Ptr(developerRole.ID),
	})
	if err != nil {
		return "", "", "", err
	}
	_, _, err = testutil.TestGitlabClient.Groups.AddGroupSAMLLink(group.ID, &gitlab.AddGroupSAMLLinkOptions{
		SAMLGroupName: gitlab.Ptr("saml_group_b"),
		AccessLevel:   gitlab.Ptr(gitlab.MaintainerPermissions),
	})
	if err != nil {
		return "", "", "", err
	}
	_, _, err = testutil.TestGitlabClient.Groups.AddGroupSAMLLink(group.ID, &gitlab.AddGroupSAMLLinkOptions{
		SAMLGroupName: gitlab.Ptr("saml_group_c"),
		AccessLevel:   gitlab.Ptr(gitlab.ReporterPermissions),
		MemberRoleID:  gitlab.Ptr(reporterRole.ID),
	})
	if err != nil {
		return "", "", "", err
	}
	return strconv.FormatInt(group.ID, 10), strconv.FormatInt(developerRole.ID, 10), strconv.FormatInt(reporterRole.ID, 10), nil
}
