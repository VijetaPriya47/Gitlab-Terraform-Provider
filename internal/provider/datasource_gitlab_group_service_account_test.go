//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitLabGroupServiceAccount_DataSource_Basic(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create group and service account
	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.FormatInt(group.ID, 10)
	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	// lintignore:AT001
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_group_service_account" "test" {
						service_account_id = %s
						group = %s
					}
					`,
					strconv.FormatInt(serviceAccount.ID, 10),
					groupID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify id attribute
					resource.TestCheckResourceAttr("data.gitlab_group_service_account.test", "service_account_id", strconv.FormatInt(serviceAccount.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_group_service_account.test", "group", groupID),
					resource.TestCheckResourceAttr("data.gitlab_group_service_account.test", "name", serviceAccount.Name),
					resource.TestCheckResourceAttr("data.gitlab_group_service_account.test", "username", serviceAccount.UserName),
				),
			},
			// Error not found
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_group_service_account" "test" {
						service_account_id = 1
						group = %s
					}
					`,
					groupID,
				),
				ExpectError: regexp.MustCompile("Service account not found"),
			},
		},
	})
}
