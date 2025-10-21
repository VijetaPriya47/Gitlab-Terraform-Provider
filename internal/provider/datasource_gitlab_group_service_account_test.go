//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitLabGroupServiceAccount_DataSource_Basic(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create group and service account
	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)
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
					strconv.Itoa(serviceAccount.ID),
					groupID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify id attribute
					resource.TestCheckResourceAttr("data.gitlab_group_service_account.test", "service_account_id", strconv.Itoa(serviceAccount.ID)),
					resource.TestCheckResourceAttr("data.gitlab_group_service_account.test", "group", groupID),
					resource.TestCheckResourceAttr("data.gitlab_group_service_account.test", "name", serviceAccount.Name),
					resource.TestCheckResourceAttr("data.gitlab_group_service_account.test", "username", serviceAccount.UserName),
				),
			},
		},
	})
}
