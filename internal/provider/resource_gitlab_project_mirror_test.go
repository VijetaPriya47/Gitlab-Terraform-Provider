//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAccGitlabProjectMirror_basic(t *testing.T) {
	var mirror gitlab.ProjectMirror
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,

		Steps: []resource.TestStep{
			// Create a project and mirror with default options
			{
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "foo" {
					project = "%d"
					url = "https://example.com/mirror-test.git"
					enabled = true
					only_protected_branches = true
					keep_divergent_refs = true
					}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "https://example.com/mirror-test.git",
						Enabled:               true,
						OnlyProtectedBranches: true,
						KeepDivergentRefs:     true,
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_mirror.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"url"},
			},
			// Update the mirror settings
			{
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "foo" {
					project = "%d"
					url = "https://example.com/mirror-test.git"
					enabled = false
					only_protected_branches = false
					keep_divergent_refs = false
					}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "https://example.com/mirror-test.git",
						Enabled:               false,
						OnlyProtectedBranches: false,
						KeepDivergentRefs:     false,
					}),
				),
			},
		},
	})
}

func testAccCheckGitlabProjectMirrorExists(n string, mirror *gitlab.ProjectMirror) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		project, rawMirrorId, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return err
		}

		mirrorID, err := strconv.Atoi(rawMirrorId)
		if err != nil {
			return err
		}

		gotMirror, _, err := testutil.TestGitlabClient.ProjectMirrors.GetProjectMirror(project, mirrorID)
		if err != nil {
			return err
		}
		*mirror = *gotMirror
		return nil
	}
}

func testAccCheckGitlabProjectMirrorDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_mirror" {
			continue
		}

		project, mirrorId, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return err
		}

		mirrorID, err := strconv.Atoi(mirrorId)
		if err != nil {
			return err
		}

		_, _, err = testutil.TestGitlabClient.ProjectMirrors.GetProjectMirror(project, mirrorID)
		if err == nil {
			return fmt.Errorf("Project Mirror %d in project %s still exists", mirrorID, project)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}

type testAccGitlabProjectMirrorExpectedAttributes struct {
	URL                   string
	Enabled               bool
	OnlyProtectedBranches bool
	KeepDivergentRefs     bool
}

func testAccCheckGitlabProjectMirrorAttributes(mirror *gitlab.ProjectMirror, want *testAccGitlabProjectMirrorExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if want.URL != "" {
			if mirror.URL != want.URL {
				return fmt.Errorf("got url %q; want %q", mirror.URL, want.URL)
			}
		}

		if mirror.Enabled != want.Enabled {
			return fmt.Errorf("got enabled %t; want %t", mirror.Enabled, want.Enabled)
		}

		if mirror.OnlyProtectedBranches != want.OnlyProtectedBranches {
			return fmt.Errorf("got only_protected_branches %t; want %t", mirror.OnlyProtectedBranches, want.OnlyProtectedBranches)
		}

		if mirror.KeepDivergentRefs != want.KeepDivergentRefs {
			return fmt.Errorf("got keep_divergent_refs %t; want %t", mirror.KeepDivergentRefs, want.KeepDivergentRefs)
		}

		return nil
	}
}

func TestAccGitlabProjectMirror_migrateFromSDKToFramework(t *testing.T) {
	var mirror gitlab.ProjectMirror
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "= 17.8",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "foo" {
					project = "%d"
					url = "https://example.com/test/test.git"
					enabled = true
					only_protected_branches = true
					keep_divergent_refs = true
				}`, project.ID),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "foo" {
					project = "%d"
					url = "https://example.com/test/test.git"
					enabled = true
					only_protected_branches = true
					keep_divergent_refs = true
				}`, project.ID),
				Check: testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_mirror.foo",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func TestAccGitlabProjectMirror_ssh(t *testing.T) {
	var mirror gitlab.ProjectMirror
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "foo" {
					project = "%d"
					url = "ssh://git@example.com/mirror-test.git"
					enabled = true
					only_protected_branches = true
					keep_divergent_refs = true
				}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "ssh://git@example.com/mirror-test.git",
						Enabled:               true,
						OnlyProtectedBranches: true,
						KeepDivergentRefs:     true,
					}),
				),
			},
		},
	})
}

func TestAccGitlabProjectMirror_urlValidations(t *testing.T) {
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			{
				// Invalid scheme
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "foo" {
					project = "%d"
					url = "ftp://example.com/test.git"
					enabled = true
				}`, project.ID),
				ExpectError: regexp.MustCompile("Only allowed schemes are http, https, ssh, git"),
			},
			{
				// Empty URL
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "foo" {
					project = "%d"
					url = ""
					enabled = true
				}`, project.ID),
				ExpectError: regexp.MustCompile("can't be blank"),
			},
			{
				// Valid HTTPS URL with credentials
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "foo" {
					project = "%d"
					url = "https://example.com/test.git"
					enabled = true
				}`, project.ID),
			},
			{
				ResourceName:      "gitlab_project_mirror.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
