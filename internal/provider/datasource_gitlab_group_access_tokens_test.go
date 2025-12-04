//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroupAccessTokens_basic(t *testing.T) {
	testGroup := testutil.CreateGroups(t, 1)[0]
	testAccessTokens := make([]*gitlab.GroupAccessToken, 0)
	for range 25 {
		testAccessTokens = append(testAccessTokens, testutil.CreateGroupAccessToken(t, testGroup.ID))
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_access_tokens" "this" {
						group = %d
					}
				`, testGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_access_tokens.this", "access_tokens.#", fmt.Sprintf("%d", len(testAccessTokens))),
					resource.TestCheckResourceAttr("data.gitlab_group_access_tokens.this", "access_tokens.0.id", strconv.FormatInt(testAccessTokens[0].ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_group_access_tokens.this", "access_tokens.0.name", testAccessTokens[0].Name),
					resource.TestCheckResourceAttr("data.gitlab_group_access_tokens.this", "access_tokens.24.id", strconv.FormatInt(testAccessTokens[24].ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_group_access_tokens.this", "access_tokens.24.name", testAccessTokens[24].Name),
				),
			},
		},
	})
}
