//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitLabGroupServiceAccountAccessTokens_DataSource_Basic(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create group and service account
	group := testutil.CreateGroups(t, 1)[0]
	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, strconv.FormatInt(group.ID, 10))[0]

	// Create a token via API
	testutil.CreateGroupServiceAccountAccessToken(t, group.ID, serviceAccount.ID, "test-token", []string{"api"})

	// lintignore:AT001
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read via the data source
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_service_account_access_tokens" "test" {
						group              = %d
						service_account_id = %d
					}
				`, group.ID, serviceAccount.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes
					resource.TestCheckResourceAttr("data.gitlab_group_service_account_access_tokens.test", "group", strconv.FormatInt(group.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_group_service_account_access_tokens.test", "service_account_id", strconv.FormatInt(serviceAccount.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_group_service_account_access_tokens.test", "id", fmt.Sprintf("%d:%d", group.ID, serviceAccount.ID)),
					// Verify at least one token is returned
					resource.TestCheckResourceAttr("data.gitlab_group_service_account_access_tokens.test", "access_tokens.#", "1"),
					// Verify the token attributes
					resource.TestCheckResourceAttr("data.gitlab_group_service_account_access_tokens.test", "access_tokens.0.name", "test-token"),
					resource.TestCheckResourceAttr("data.gitlab_group_service_account_access_tokens.test", "access_tokens.0.active", "true"),
					resource.TestCheckResourceAttr("data.gitlab_group_service_account_access_tokens.test", "access_tokens.0.revoked", "false"),
					resource.TestCheckResourceAttrSet("data.gitlab_group_service_account_access_tokens.test", "access_tokens.0.id"),
					resource.TestCheckResourceAttrSet("data.gitlab_group_service_account_access_tokens.test", "access_tokens.0.created_at"),
				),
			},
		},
	})
}

func TestAcc_GitLabGroupServiceAccountAccessTokens_DataSource_MultipleTokens(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create group and service account
	group := testutil.CreateGroups(t, 1)[0]
	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, strconv.FormatInt(group.ID, 10))[0]

	// Create tokens via API
	testutil.CreateGroupServiceAccountAccessToken(t, group.ID, serviceAccount.ID, "test-token-1", []string{"api"})
	testutil.CreateGroupServiceAccountAccessToken(t, group.ID, serviceAccount.ID, "test-token-2", []string{"read_api"})

	// lintignore:AT001
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple tokens, then read them via the data source
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_service_account_access_tokens" "test" {
						group              = %d
						service_account_id = %d
					}
				`, group.ID, serviceAccount.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify at least two tokens are returned
					resource.TestCheckResourceAttr("data.gitlab_group_service_account_access_tokens.test", "access_tokens.#", "2"),
				),
			},
		},
	})
}
