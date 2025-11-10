//go:build acceptance

package sdk

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectMembership_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	user := testutil.CreateUsers(t, 1)[0]
	var membership gitlab.ProjectMember

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectMembershipDestroy,
		Steps: []resource.TestStep{
			// Assign member to the project as a developer
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_membership" "foo" {
						project      = "%d"
						user_id      = %d
						access_level = "developer"
					}
				`, project.ID, user.ID),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabProjectMembershipExists("gitlab_project_membership.foo", &membership), testAccCheckGitlabProjectMembershipAttributes(&membership, &testAccGitlabProjectMembershipExpectedAttributes{
					access_level: "developer",
				})),
			},

			// Update the project member to change the access level (use testAccGitlabProjectMembershipUpdateConfig for Config)
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_membership" "foo" {
						project      = "%d"
						user_id      = %d
						expires_at   = "2099-01-01"
						access_level = "guest"
					}
				`, project.ID, user.ID),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabProjectMembershipExists("gitlab_project_membership.foo", &membership), testAccCheckGitlabProjectMembershipAttributes(&membership, &testAccGitlabProjectMembershipExpectedAttributes{
					access_level: "guest",
					expiresAt:    "2099-01-01",
				})),
			},

			// Update the project member to change the access level back
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_membership" "foo" {
						project      = "%d"
						user_id      = %d
						access_level = "developer"
					}
				`, project.ID, user.ID),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabProjectMembershipExists("gitlab_project_membership.foo", &membership), testAccCheckGitlabProjectMembershipAttributes(&membership, &testAccGitlabProjectMembershipExpectedAttributes{
					access_level: "developer",
				})),
			},
		},
	})
}

func TestAccGitlabProjectMembership_UseCustomRole(t *testing.T) {
	// custom roles only available to EE ultimate
	testutil.SkipIfCE(t)

	// create a user
	user := testutil.CreateUsers(t, 1)[0]
	// create a group where we will define the custom role
	group := testutil.CreateGroups(t, 1)[0]

	// Create a custom role on that group which will be deleted when the test finishes
	roleOne := testutil.CreateCustomInstanceRole(t, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr("test-role"),
		BaseAccessLevel:   gitlab.Ptr(gitlab.ReporterPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})

	// Create a second custom role on that group (for testing update; will be deleted when the test finishes)
	roleTwo := testutil.CreateCustomInstanceRole(t, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr("test-role-update"),
		BaseAccessLevel:   gitlab.Ptr(gitlab.DeveloperPermissions),
		ReadVulnerability: gitlab.Ptr(true),
		ReadCode:          gitlab.Ptr(false),
	})

	// create a project in the group so we can grant the member access via the custom role
	project := testutil.CreateProjectWithNamespace(t, group.ID)

	checkProjectMembershipViaAPI := func(s *terraform.State) error {
		member, _, err := testutil.TestGitlabClient.ProjectMembers.GetProjectMember(project.ID, user.ID)
		if err != nil {
			return fmt.Errorf("Error getting project member via API: %v", err)
		}

		if member.MemberRole != nil {
			return fmt.Errorf("API CHECK FAILED: MemberRole should be nil: got %v", member.MemberRole)
		}

		// Return nil to indicate the check passed.
		return nil
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectMembershipDestroy,
		Steps: []resource.TestStep{
			// Assign member to the project as a custom reporter-based role
			{
				Config: fmt.Sprintf(
					`
					resource "gitlab_project_membership" "foo" {
						project         = "%d"
						user_id         = "%d"
						access_level 	= "reporter"
						member_role_id  = %d
					}
					`, project.ID, user.ID, roleOne.ID,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_membership.foo", "project", strconv.Itoa(project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_membership.foo", "member_role_id", strconv.Itoa(roleOne.ID)),
				),
			},
			{
				ResourceName:      "gitlab_project_membership.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"skip_subresources_on_destroy",
					"unassign_issuables_on_destroy",
				},
			},
			// Assign member to the project as a separate custom developer-based role
			{
				Config: fmt.Sprintf(
					`
					resource "gitlab_project_membership" "foo" {
						project         = "%d"
						user_id         = "%d"
						access_level    = "developer"
						member_role_id  = %d
					}
					`, project.ID, user.ID, roleTwo.ID,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_membership.foo", "project", strconv.Itoa(project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_membership.foo", "member_role_id", strconv.Itoa(roleTwo.ID)),
				),
			},
			{
				Config: fmt.Sprintf(
					`
					resource "gitlab_project_membership" "foo" {
						project      = "%d"
						user_id      = "%d"
						access_level = "developer"
					}
					`, project.ID, user.ID,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_membership.foo", "project", strconv.Itoa(project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_membership.foo", "member_role_id", "0"),
					checkProjectMembershipViaAPI,
				),
			},
		},
	})
}

func testAccCheckGitlabProjectMembershipExists(n string, membership *gitlab.ProjectMember) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		projectID := rs.Primary.Attributes["project"]
		if projectID == "" {
			return fmt.Errorf("No project ID is set")
		}

		userID := rs.Primary.Attributes["user_id"]
		id, _ := strconv.Atoi(userID)
		if userID == "" {
			return fmt.Errorf("No user id is set")
		}

		gotProjectMembership, _, err := testutil.TestGitlabClient.ProjectMembers.GetProjectMember(projectID, id)
		if err != nil {
			return err
		}

		*membership = *gotProjectMembership
		return nil
	}
}

type testAccGitlabProjectMembershipExpectedAttributes struct {
	access_level string
	expiresAt    string
}

func testAccCheckGitlabProjectMembershipAttributes(membership *gitlab.ProjectMember, want *testAccGitlabProjectMembershipExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		access_level_id, ok := api.AccessLevelValueToName[membership.AccessLevel]
		if !ok {
			return fmt.Errorf("Invalid access level '%s'", access_level_id)
		}
		if access_level_id != want.access_level {
			return fmt.Errorf("got access level %s; want %s", access_level_id, want.access_level)
		}
		return nil
	}
}

func testAccCheckGitlabProjectMembershipDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_membership" {
			continue
		}

		projectID := rs.Primary.Attributes["project"]
		userID := rs.Primary.Attributes["user_id"]

		// GetProjectMember needs int type for userID
		userIDI, err := strconv.Atoi(userID)
		if err != nil {
			return err
		}
		gotMembership, _, err := testutil.TestGitlabClient.ProjectMembers.GetProjectMember(projectID, userIDI)
		if err != nil {
			if api.Is404(err) {
				return nil
			}
			if gotMembership != nil && fmt.Sprintf("%d", gotMembership.AccessLevel) == rs.Primary.Attributes["access_level"] {
				return fmt.Errorf("Project still has member.")
			}
			return err
		}

		return nil
	}
	return nil
}

func TestAccGitlabProjectMembership_failsWithPastExpiryDate(t *testing.T) {
	project := testutil.CreateProject(t)
	user := testutil.CreateUsers(t, 1)[0]

	pastDateForConfig := api.CurrentTime().Add(-24 * time.Hour).Format("2006-01-02")

	parsedDate, err := time.Parse("2006-01-02", pastDateForConfig)
	if err != nil {
		t.Fatalf("Failed to parse date for test setup: %v", err)
	}

	pastDateForError := parsedDate.Format(time.RFC3339)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectMembershipDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_membership" "test_fail" {
						project                        = "%d"
						user_id                        = %d
						access_level                   = "developer"
						expires_at                     = "%s"
					}
				`, project.ID, user.ID, pastDateForConfig),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`(?s)Expiry date %s must be in the future\. Current time is\s*.*`, pastDateForError)),
			},
		},
	})
}

func TestAccGitlabProjectMembership_failsToUpdateWithPastExpiryDate(t *testing.T) {
	project := testutil.CreateProject(t)
	user := testutil.CreateUsers(t, 1)[0]

	futureDate := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	pastDateForConfig := time.Now().Add(-24 * time.Hour).Format("2006-01-02")

	parsedDate, err := time.Parse("2006-01-02", pastDateForConfig)
	if err != nil {
		t.Fatalf("Failed to parse date for test setup: %v", err)
	}

	pastDateForError := parsedDate.Format(time.RFC3339)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectMembershipDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_membership" "test_update_fail" {
						project                        = "%d"
						user_id                        = %d
						access_level                   = "developer"
						expires_at                     = "%s"
					}
				`, project.ID, user.ID, futureDate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_membership.test_update_fail", "expires_at", futureDate),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_membership" "test_update_fail" {
						project                        = "%d"
						user_id                        = %d
						access_level                   = "developer"
						expires_at                     = "%s"
					}
				`, project.ID, user.ID, pastDateForConfig),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`(?s)Expiry date %s must be in the future\. Current time is\s*.*`, pastDateForError)),
			},
		},
	})
}
