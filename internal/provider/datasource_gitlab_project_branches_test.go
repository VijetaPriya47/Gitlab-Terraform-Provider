//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectBranches_search(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testBranches := testutil.CreateBranches(t, testProject, 25)
	expectedBranches := len(testBranches) + 1 // main branch already exists

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_branches" "this" {
						project = "%d"
					}
				`, testProject.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.#", fmt.Sprintf("%d", expectedBranches)),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.merged"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.protected"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.default"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.developers_can_push"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.developers_can_merge"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.can_push"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.web_url"),
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.0.commit.#", "1"),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectBranches_filters(t *testing.T) {
	testProject := testutil.CreateProject(t)
	branchPrefix := acctest.RandomWithPrefix("acctest-branch")
	searchToken := acctest.RandomWithPrefix("search")
	regexToken := acctest.RandomWithPrefix("regex")
	searchBranchName := fmt.Sprintf("%s-%s", branchPrefix, searchToken)
	regexBranchName := fmt.Sprintf("%s-%s", branchPrefix, regexToken)

	testutil.CreateBranch(t, testProject, searchBranchName)
	testutil.CreateBranch(t, testProject, regexBranchName)

	regexPattern := fmt.Sprintf("^%s$", regexBranchName)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_branches" "this" {
						project = "%[1]d"
						search  = "%[2]s"
					}
				`, testProject.ID, searchToken),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.0.name", searchBranchName),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_branches" "this" {
						project = "%[1]d"
						regex   = "%[2]s"
					}
				`, testProject.ID, regexPattern),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.0.name", regexBranchName),
				),
			},
		},
	})
}

// This tests is testing that the update from https://github.com/mitchellh/hashstructure -> V2 maintains the ID properly
// and doesn't result in a breaking change.
func TestAccDataGitlabProjectBranches_UpdateHashStruct(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// We want to use the same config on old and new version
	commonConfig := fmt.Sprintf(`
	data "gitlab_project_branches" "this" {
		project = "%d"
	}
	`, testProject.ID)

	oldID := ""

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				// The version before this change was made.
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "3.20.0",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: commonConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "id"),

					// Store the generated ID for the next step in the text.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.gitlab_project_branches.this"]
						if !ok {
							return fmt.Errorf("data.gitlab_project_branches.this not found")
						}

						// Set the ID so we can check it later
						oldID = rs.Primary.ID
						return nil
					},
				),
			},
			{
				// The version before this change was made.
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "16.0",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: commonConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "id"),

					// Verify that the old and new hashes match.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.gitlab_project_branches.this"]
						if !ok {
							return fmt.Errorf("data.gitlab_project_branches.this not found")
						}

						// get the new ID to check it
						newID := rs.Primary.ID
						if newID != oldID {
							return fmt.Errorf("old and new IDs do not match! There is an error in the hash generation, likely in github.com/mitchellh/hashstructure/v2")
						}

						// hashes match.
						return nil
					},
				),
			},
			{
				// The data source now simply uses the project as the ID, but we still want
				// to ensure that the upgrade path doesn't cause issues, so we simply run the
				// config one more time to ensure no terraform errors happen.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   commonConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "id"),

					// Verify the ID os the project ID
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.gitlab_project_branches.this"]
						if !ok {
							return fmt.Errorf("data.gitlab_project_branches.this not found")
						}

						// get the new ID to check it against the project ID
						newID := rs.Primary.ID
						if newID != strconv.FormatInt(testProject.ID, 10) {
							return fmt.Errorf("project ID and data source ID do not match!")
						}

						return nil
					},
				),
			},
		},
	})
}
