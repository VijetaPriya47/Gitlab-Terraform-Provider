//go:build acceptance
// +build acceptance

package sdk

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectMirror_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	var mirror gitlab.ProjectMirror

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			// Create with default options
			{
				Config: testAccGitlabProjectMirrorConfig(testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "https://example.com/mirror",
						Enabled:               true,
						OnlyProtectedBranches: true,
						KeepDivergentRefs:     true,
					}),
				),
			},
			// Update to toggle all the values to their inverse
			{
				Config: testAccGitlabProjectMirrorUpdateConfig(testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "https://example.com/mirror",
						Enabled:               false,
						OnlyProtectedBranches: false,
						KeepDivergentRefs:     false,
					}),
				),
			},
			// Update to toggle the options back
			{
				Config: testAccGitlabProjectMirrorConfig(testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "https://example.com/mirror",
						Enabled:               true,
						OnlyProtectedBranches: true,
						KeepDivergentRefs:     true,
					}),
				),
			},
			// Import
			{
				ResourceName:      "gitlab_project_mirror.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectMirror_withPassword(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			// Create a project and mirror with a username / password.
			{
				Config: testAccGitlabProjectMirrorConfigWithPassword(testProject.PathWithNamespace),
			},
		},
	})
}

func testAccCheckGitlabProjectMirrorExists(n string, mirror *gitlab.ProjectMirror) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		splitID := strings.Split(rs.Primary.ID, ":")

		mirrorID, err := strconv.Atoi(splitID[len(splitID)-1])
		if err != nil {
			return err
		}
		repoName := rs.Primary.Attributes["project"]
		if repoName == "" {
			return fmt.Errorf("No project ID is set")
		}

		mirrors, _, err := testutil.TestGitlabClient.ProjectMirrors.ListProjectMirror(repoName, nil)
		if err != nil {
			return err
		}

		for _, m := range mirrors {
			if m.ID == mirrorID {
				*mirror = *m
				return nil
			}
		}
		return errors.New("unable to find mirror")
	}
}

type testAccGitlabProjectMirrorExpectedAttributes struct {
	URL                   string
	Enabled               bool
	OnlyProtectedBranches bool
	KeepDivergentRefs     bool
}

func testAccCheckGitlabProjectMirrorAttributes(mirror *gitlab.ProjectMirror, want *testAccGitlabProjectMirrorExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mirror.URL != want.URL {
			return fmt.Errorf("got url %q; want %q", mirror.URL, want.URL)
		}

		if mirror.Enabled != want.Enabled {
			return fmt.Errorf("got enabled %t; want %t", mirror.Enabled, want.Enabled)
		}
		if mirror.OnlyProtectedBranches != want.OnlyProtectedBranches {
			return fmt.Errorf("got only protected branches %t; want %t", mirror.OnlyProtectedBranches, want.OnlyProtectedBranches)
		}
		if mirror.KeepDivergentRefs != want.KeepDivergentRefs {
			return fmt.Errorf("got keep divergent refs %t; want %t", mirror.KeepDivergentRefs, want.KeepDivergentRefs)
		}
		return nil
	}
}

func testAccCheckGitlabProjectMirrorDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_mirror" {
			continue
		}

		projectID, rawMirrorID, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return err
		}
		mirrorID, err := strconv.Atoi(rawMirrorID)
		if err != nil {
			return err
		}

		mirror, _, err := testutil.TestGitlabClient.ProjectMirrors.GetProjectMirror(projectID, mirrorID)
		if err == nil && mirror != nil && mirror.ID == mirrorID {
			return fmt.Errorf("Project Mirror still exists")
		}
		if err != nil && !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}

func testAccGitlabProjectMirrorConfig(project string) string {
	return fmt.Sprintf(`
resource "gitlab_project_mirror" "foo" {
  project = %q
  url = "https://example.com/mirror"
}
	`, project)
}

func testAccGitlabProjectMirrorConfigWithPassword(project string) string {
	return fmt.Sprintf(`
resource "gitlab_project_mirror" "foo" {
  project = %q
  url = "https://foo:bar@example.com/mirror"
}
	`, project)
}

func testAccGitlabProjectMirrorUpdateConfig(project string) string {
	return fmt.Sprintf(`
resource "gitlab_project_mirror" "foo" {
  project = %q
  url = "https://example.com/mirror"
  enabled = false
  only_protected_branches = false
  keep_divergent_refs = false
}
	`, project)
}
