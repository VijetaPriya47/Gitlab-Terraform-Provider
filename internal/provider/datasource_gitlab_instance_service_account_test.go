//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitLabInstanceServiceAccount_DataSource_Basic(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create group and service account
	serviceAccount := testutil.CreateInstanceServiceAccounts(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_instance_service_account" "test" {
						service_account_id = %d
					}
					`,
					serviceAccount.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify id attribute
					resource.TestCheckResourceAttr("data.gitlab_instance_service_account.test", "service_account_id", strconv.Itoa(serviceAccount.ID)),
					resource.TestCheckResourceAttr("data.gitlab_instance_service_account.test", "name", serviceAccount.Name),
					resource.TestCheckResourceAttr("data.gitlab_instance_service_account.test", "username", serviceAccount.Username),
					resource.TestCheckResourceAttr("data.gitlab_instance_service_account.test", "email", serviceAccount.Email),
				),
			},
		},
	})
}
