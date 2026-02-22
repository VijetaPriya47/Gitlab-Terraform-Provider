//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
				}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "https://example.com/mirror-test.git",
						Enabled:               true,
						OnlyProtectedBranches: true,
						KeepDivergentRefs:     true,
						AuthMethod:            "password",
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
						AuthMethod:            "password",
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

		mirrorID, err := strconv.ParseInt(rawMirrorId, 10, 64)
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

		mirrorID, err := strconv.ParseInt(mirrorId, 10, 64)
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
	MirrorBranchRegex     string
	KeepDivergentRefs     bool
	AuthMethod            string
}

func testAccCheckGitlabProjectMirrorAttributes(mirror *gitlab.ProjectMirror, want *testAccGitlabProjectMirrorExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if want.URL != "" && mirror.URL != want.URL {
			return fmt.Errorf("got url %q; want %q", mirror.URL, want.URL)
		}

		if mirror.Enabled != want.Enabled {
			return fmt.Errorf("got enabled %t; want %t", mirror.Enabled, want.Enabled)
		}

		if mirror.OnlyProtectedBranches != want.OnlyProtectedBranches {
			return fmt.Errorf("got only_protected_branches %t; want %t", mirror.OnlyProtectedBranches, want.OnlyProtectedBranches)
		}

		if mirror.MirrorBranchRegex != want.MirrorBranchRegex {
			return fmt.Errorf("got mirror_branch_regex %s; want %s", mirror.MirrorBranchRegex, want.MirrorBranchRegex)
		}

		if mirror.KeepDivergentRefs != want.KeepDivergentRefs {
			return fmt.Errorf("got keep_divergent_refs %t; want %t", mirror.KeepDivergentRefs, want.KeepDivergentRefs)
		}

		if want.AuthMethod != "" && mirror.AuthMethod != want.AuthMethod {
			return fmt.Errorf("got auth_method %s; want %s", mirror.AuthMethod, want.AuthMethod)
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
			// Note: URL is ignored because GitLab API returns redacted URLs without credentials
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_mirror.foo",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore:  []string{"url"},
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
					project                 = "%d"
					url                     = "ssh://git@example.com/mirror-test.git"
					enabled                 = true
					only_protected_branches = true
					keep_divergent_refs     = true
					auth_method             = "ssh_public_key"
				}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "ssh://git@example.com/mirror-test.git",
						Enabled:               true,
						OnlyProtectedBranches: true,
						KeepDivergentRefs:     true,
						AuthMethod:            "ssh_public_key",
					}),
				),
			},
		},
	})
}

func TestAccGitlabProjectMirror_AuthMethod(t *testing.T) {
	var mirror gitlab.ProjectMirror
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "foo" {
					project                 = "%d"
					url                     = "ssh://git@example.com/mirror-test.git"
					enabled                 = true
					only_protected_branches = true
					keep_divergent_refs     = true
					auth_method             = "ssh_public_key"
				}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "ssh://git@example.com/mirror-test.git",
						Enabled:               true,
						OnlyProtectedBranches: true,
						KeepDivergentRefs:     true,
						AuthMethod:            "ssh_public_key",
					}),
				),
			},
			// Verify upstream attributes with an import
			// Note: URL is ignored because GitLab API returns redacted URLs without credentials
			{
				ResourceName:            "gitlab_project_mirror.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"url"},
			},
			{
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "bar" {
					project                 = "%d"
					url                     = "https://git:*****@example.com/mirror-test.git"
					enabled                 = true
					only_protected_branches = true
					keep_divergent_refs     = true
					auth_method             = "password"
				}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.bar", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "https://git:*****@example.com/mirror-test.git",
						Enabled:               true,
						OnlyProtectedBranches: true,
						KeepDivergentRefs:     true,
						AuthMethod:            "password",
					}),
				),
			},
			// Verify upstream attributes with an import
			// Note: URL is ignored because GitLab API returns redacted URLs without credentials
			{
				ResourceName:            "gitlab_project_mirror.bar",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"url"},
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
			// Note: URL is ignored because GitLab API returns redacted URLs without credentials
			{
				ResourceName:            "gitlab_project_mirror.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"url"},
			},
		},
	})
}

func TestAccGitlabProjectMirror_AuthMethodValidation(t *testing.T) {
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			{
				// Invalid auth method
				Config: fmt.Sprintf(`resource "gitlab_project_mirror" "foo" {
					project     = "%d"
					url         = "https://example.com/test.git"
					auth_method = "basic"
				}`, project.ID),
				ExpectError: regexp.MustCompile("Attribute auth_method value must be one of"),
			},
		},
	})
}

func TestAccGitlabProjectMirror_branchRegex(t *testing.T) {
	// Branch regex only available in premium and ultimate
	testutil.SkipIfCE(t)

	var mirror gitlab.ProjectMirror
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			// Check conflicts with only_protected_branches
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "foo" {
						project                 = "%d"
						url                     = "ssh://git@example.com/mirror-test.git"
						only_protected_branches = true
						mirror_branch_regex     = "release/*"
					}
				`, project.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
			// Check only_protected_branches not set when regex used
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "foo" {
						project             = "%d"
						url                 = "ssh://git@example.com/mirror-test.git"
						mirror_branch_regex = "release/*"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:               "ssh://git@example.com/mirror-test.git",
						Enabled:           true,
						MirrorBranchRegex: "release/*",
						KeepDivergentRefs: true,
						AuthMethod:        "password",
					}),
				),
			},
			// Verify upstream attributes with an import
			// Note: URL is ignored because GitLab API returns redacted URLs without credentials
			{
				ResourceName:            "gitlab_project_mirror.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"url"},
			},
			// Update regex
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "foo" {
						project             = "%d"
						url                 = "ssh://git@example.com/mirror-test.git"
						mirror_branch_regex = "develop/*"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:               "ssh://git@example.com/mirror-test.git",
						Enabled:           true,
						MirrorBranchRegex: "develop/*",
						KeepDivergentRefs: true,
						AuthMethod:        "password",
					}),
				),
			},
			// Verify upstream attributes with an import
			// Note: URL is ignored because GitLab API returns redacted URLs without credentials
			{
				ResourceName:            "gitlab_project_mirror.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"url"},
			},
			// test to verify mirror_branch_regex doesn't always show as unknown in plan when not provided
			// for https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/6473
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "bar" {
						project  = "%d"
						url      = "https://user:password@git.example.org/path/to/repo.git"
						enabled  = true
			  
						keep_divergent_refs     = false
						only_protected_branches = true
			  		}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.bar", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						Enabled:               true,
						KeepDivergentRefs:     false,
						OnlyProtectedBranches: true,
					}),
				),
			},
			// Verify upstream attributes with an import
			{
				ResourceName:            "gitlab_project_mirror.bar",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"url"},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "bar" {
						project  = "%d"
						url      = "https://user:password@git.example.org/path/to/repo.git"
						enabled  = true
			  
						keep_divergent_refs     = false
						only_protected_branches = true
			  		}
				`, project.ID),
				PlanOnly: true,
			},
		},
	})
}

func TestAccGitlabProjectMirror_credentialChange(t *testing.T) {
	var mirror gitlab.ProjectMirror
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			// Create a mirror with URL containing credentials
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "foo" {
						project = "%d"
						url     = "https://user1:pass1@example.com/test.git"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
				),
			},
			// Update to different credentials (same host/path, different credentials)
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "foo" {
						project = "%d"
						url     = "https://user2:pass2@example.com/test.git"
					}
				`, project.ID),
				// Check that an "Replace" is being performed since the username and password have changed
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("gitlab_project_mirror.foo", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_mirror.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"url"},
			},
		},
	})
}

func TestAccGitlabProjectMirror_credentialChangeWarning(t *testing.T) {
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			// Create a mirror with URL containing credentials
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "foo" {
						project = "%d"
						url     = "https://user:pass@example.com/test.git"
					}
				`, project.ID),
				ExpectNonEmptyPlan: false,
			},
			// Verify that a plan with credentials shows a warning
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "foo" {
						project = "%d"
						url     = "https://user:pass@example.com/test.git"
					}
				`, project.ID),
				PlanOnly: true,
			},
		},
	})
}

func TestAccGitlabProjectMirror_urlChangeWithoutCredentials(t *testing.T) {
	var mirror gitlab.ProjectMirror
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			// Create a mirror with URL without credentials
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "foo" {
						project = "%d"
						url     = "https://example.com/test1.git"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "https://example.com/test1.git",
						Enabled:               true,
						OnlyProtectedBranches: true,
						KeepDivergentRefs:     true,
						AuthMethod:            "password",
					}),
				),
			},
			// Update to different path (no credentials)
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_mirror" "foo" {
						project = "%d"
						url     = "https://example.com/test2.git"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMirrorExists("gitlab_project_mirror.foo", &mirror),
					testAccCheckGitlabProjectMirrorAttributes(&mirror, &testAccGitlabProjectMirrorExpectedAttributes{
						URL:                   "https://example.com/test2.git",
						Enabled:               true,
						OnlyProtectedBranches: true,
						KeepDivergentRefs:     true,
						AuthMethod:            "password",
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
		},
	})
}
