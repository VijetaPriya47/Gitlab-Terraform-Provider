//go:build acceptance

package sdk

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupLdapLink_SchemaMigration0_1(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabGroupLdapLinkDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 15.7.0", // Earliest 15.X deployment
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "foo" {
					group_id	    = "%d"
					cn				= "default"
					group_access 	= "developer"
					ldap_provider   = "default"
				}`, testGroup.ID),
			},
			{
				// "group_id" changed to "group" in 16.0, but apply should still work properly.
				ProtoV6ProviderFactories: providerFactoriesV6,
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "foo" {
					group 		    = "%d"
					cn				= "default"
					group_access 	= "developer"
					ldap_provider   = "default"
				}`, testGroup.ID),
				PlanOnly: true,
			},
		},
	})
}

func TestAccGitlabGroupLdapLink_basicCN(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	resourceName := "gitlab_group_ldap_link.foo"

	// PreCheck runs after Config so load test data here
	var ldapLink gitlab.LDAPGroupLink
	testLdapLink := gitlab.LDAPGroupLink{
		CN:       "default",
		Provider: "default",
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupLdapLinkDestroy,
		Steps: []resource.TestStep{
			// Create a group LDAP link as a developer (uses testAccGitlabGroupLdapLinkCreateConfig for Config)
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_group_ldap_link" "foo" {
					group 		    = "%d"
					cn				    = "%s"
					group_access 	= "developer"
					ldap_provider = "%s"
					force         = true
				
				}`, group.ID, testLdapLink.CN, testLdapLink.Provider),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupLdapLinkExists(resourceName, &ldapLink),
					testAccCheckGitlabGroupLdapLinkAttributes(&ldapLink, &testAccGitlabGroupLdapLinkExpectedAttributes{
						accessLevel: "developer",
					})),
			},

			// Import the group LDAP link (re-uses testAccGitlabGroupLdapLinkCreateConfig for Config)
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},

			// Update the group LDAP link to change the access level (uses testAccGitlabGroupLdapLinkUpdateConfig for Config)
			{
				Config: fmt.Sprintf(`			
				resource "gitlab_group_ldap_link" "foo" {
					group 		    = "%d"
					cn				    = "%s"
					group_access 	= "maintainer"
					ldap_provider = "%s"
				}`, group.ID, testLdapLink.CN, testLdapLink.Provider),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupLdapLinkExists(resourceName, &ldapLink),
					testAccCheckGitlabGroupLdapLinkAttributes(&ldapLink, &testAccGitlabGroupLdapLinkExpectedAttributes{
						accessLevel: "maintainer",
					})),
			},
		},
	})
}

func TestAccGitlabGroupLdapLink_basicFilter(t *testing.T) {
	testutil.SkipIfCE(t)

	resourceName := "gitlab_group_ldap_link.foo"

	group := testutil.CreateGroups(t, 1)[0]

	// PreCheck runs after Config so load test data here
	var ldapLink gitlab.LDAPGroupLink

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupLdapLinkDestroy,
		Steps: []resource.TestStep{
			// Create a group LDAP link using a valid filter
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "foo" {
					group 	      = "%d"
					filter        = "(&(objectClass=person)(objectClass=user))"
					group_access  = "developer"
					ldap_provider = "default"
				
				}`, group.ID),
				Check: testAccCheckGitlabGroupLdapLinkExists(resourceName, &ldapLink),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
		},
	})
}

func TestAccGitlabGroupLdapLink_customRole(t *testing.T) {
	testutil.SkipIfCE(t)
	var ldapLink gitlab.LDAPGroupLink

	// Create a custom instance role to use for testing
	rInt := acctest.RandInt()
	role := testutil.CreateCustomInstanceRole(t, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr(fmt.Sprintf("test-role-%d", rInt)),
		BaseAccessLevel:   gitlab.Ptr(gitlab.MaintainerPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})
	roleTwo := testutil.CreateCustomInstanceRole(t, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr(fmt.Sprintf("test-role-two-%d", rInt)),
		BaseAccessLevel:   gitlab.Ptr(gitlab.MaintainerPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupLdapLinkDestroy,
		Steps: []resource.TestStep{
			// Create a group LDAP link using a valid filter
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "foo" {
					group 	      = "%d"
					member_role_id = %d

					// needs to match maintainer permissions in the role
					group_access  = "maintainer" 
					ldap_provider = "default"
					filter        = "(&(objectClass=person)(objectClass=user))"
				}`, group.ID, role.ID),
				Check: testAccCheckGitlabGroupLdapLinkExists("gitlab_group_ldap_link.foo", &ldapLink),
			},
			{
				ResourceName:      "gitlab_group_ldap_link.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "foo" {
					group 	      = "%d"
					member_role_id = %d

					// needs to match maintainer permissions in the role
					group_access  = "maintainer" 
					ldap_provider = "default"
					filter        = "(&(objectClass=person)(objectClass=user))"
				}`, group.ID, roleTwo.ID),
				Check: testAccCheckGitlabGroupLdapLinkExists("gitlab_group_ldap_link.foo", &ldapLink),
			},
			{
				ResourceName:      "gitlab_group_ldap_link.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
			// Remove the custom role to revert to a base role.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "foo" {
					group 	      = "%d"
					member_role_id = 0

					// needs to match maintainer permissions in the role
					group_access  = "maintainer" 
					ldap_provider = "default"
					filter        = "(&(objectClass=person)(objectClass=user))"
				}`, group.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGitlabGroupLdapLinkExists("gitlab_group_ldap_link.foo", &ldapLink),
					// check that member_role_id has been removed from the API object returned from GitLab.
					func(s *terraform.State) error {
						if ldapLink.MemberRoleID != 0 {
							return fmt.Errorf("expected member_role_id to be removed, but got %d", ldapLink.MemberRoleID)
						}
						return nil
					},
				),
			},
		},
	})
}

// Since LDAP links are destroyed when a group is, test that LDAP links clean up
// properly when the group is removed.
func TestAccGitlabGroupLdapLink_removeOutsideTf(t *testing.T) {
	testutil.SkipIfCE(t)

	var ldapLink gitlab.LDAPGroupLink

	// permanently_delete only works on subgroups, so we need to create a TLG to use for the test
	parentGroup := testutil.CreateGroups(t, 1)[0]

	// Create two groups for use during the test, both of which are subgroups of the initial group
	groups := testutil.CreateSubGroups(t, parentGroup, 2)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupLdapLinkDestroy,
		Steps: []resource.TestStep{
			// Create a group LDAP link as a developer (uses testAccGitlabGroupLdapLinkCreateConfig for Config)
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "this" {
					group 		    = %d
					cn				= "default"
					group_access 	= "developer"
					ldap_provider   = "default"	
					
					force           = true
				}`, groups[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupLdapLinkExists("gitlab_group_ldap_link.this", &ldapLink),
				),
			},
			{
				// Destroy the group outside of TF, which will also destroy the LDAP link by proxy
				PreConfig: func() {
					// Mark the group for deletion, then delete it
					// We don't need to check error on the first call because the second will fail if the first one does.
					_, _ = testutil.TestGitlabClient.Groups.DeleteGroup(groups[0].ID, nil)
					softDeletedGroup, _, err := testutil.TestGitlabClient.Groups.GetGroup(groups[0].ID, nil)
					if err != nil {
						t.Fatalf("Failed to get deleted group outside of TF. err: %v", err)
					}
					_, err = testutil.TestGitlabClient.Groups.DeleteGroup(groups[0].ID, &gitlab.DeleteGroupOptions{
						PermanentlyRemove: gitlab.Ptr(true),
						FullPath:          gitlab.Ptr(softDeletedGroup.FullPath),
					})
					if err != nil {
						t.Fatalf("Failed to delete group outside of TF. err: %v", err)
					}
				},
				Config: fmt.Sprintf(`				
				resource "gitlab_group_ldap_link" "this" {
					group 		    = %d
					cn				= "default"
					group_access 	= "developer"
					ldap_provider   = "default"

					force           = true
				}`, groups[1].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupLdapLinkExists("gitlab_group_ldap_link.this", &ldapLink),
				),
			},
		},
	})
}

func TestAccGitlabGroupLdapLink_updateCnAndFilter(t *testing.T) {
	testutil.SkipIfCE(t)
	group := testutil.CreateGroups(t, 1)[0]

	// PreCheck runs after Config so load test data here
	var ldapLink gitlab.LDAPGroupLink

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupLdapLinkDestroy,
		Steps: []resource.TestStep{
			// Create a group LDAP link with CN
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "foo" {
					group 		    = "%d"
					cn				    = "hello-world"
					group_access 	= "developer"
					ldap_provider = "default"
				}`, group.ID),
				Check: testAccCheckGitlabGroupLdapLinkExists("gitlab_group_ldap_link.foo", &ldapLink),
			},
			{
				ResourceName:      "gitlab_group_ldap_link.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
			// Updating CN should force creation of a new resource, but should still work
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "foo" {
					group 		    = "%d"
					cn				    = "new-cn"
					group_access 	= "developer"
					ldap_provider = "default"
				}`, group.ID),
				Check: testAccCheckGitlabGroupLdapLinkExists("gitlab_group_ldap_link.foo", &ldapLink),
			},
			{
				ResourceName:      "gitlab_group_ldap_link.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
			// Remove CN and add filter
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "foo" {
					group 		    = "%d"
					filter			  = "(givenName=Kitty)"
					group_access 	= "developer"
					ldap_provider = "default"
				}`, group.ID),
				Check: testAccCheckGitlabGroupLdapLinkExists("gitlab_group_ldap_link.foo", &ldapLink),
			},
			{
				ResourceName:      "gitlab_group_ldap_link.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
			// Update filter, which should remove and re-create the resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_ldap_link" "foo" {
					group 		    = "%d"
					filter			  = "(givenName=Meow)"
					group_access 	= "developer"
					ldap_provider = "default"
				}`, group.ID),
				Check: testAccCheckGitlabGroupLdapLinkExists("gitlab_group_ldap_link.foo", &ldapLink),
			},
			{
				ResourceName:      "gitlab_group_ldap_link.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
		},
	})
}

func TestAccGitlabGroupLdapLink_conflictingArguments(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupLdapLinkDestroy,
		Steps: []resource.TestStep{
			// Create a group LDAP link using conflicting arguments
			// ensure both conflict errors are printed appropriately.
			{
				Config: fmt.Sprintf(`resource "gitlab_group_ldap_link" "foo" {
					group 		    = "%d"
					cn            = "default"
					filter        = "(&(objectClass=person)(objectClass=user))"
					group_access 	= "developer"
					ldap_provider = "default"
				}`, group.ID),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(`"cn": conflicts with filter`)),
			},
			{
				Config: fmt.Sprintf(`resource "gitlab_group_ldap_link" "foo" {
					group 	    	= "%d"
					cn            = "default"
					filter        = "(&(objectClass=person)(objectClass=user))"
					group_access 	= "developer"
					ldap_provider = "default"
				}`, group.ID),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(`"filter": conflicts with cn`)),
			},
		},
	})
}

func TestAccGitlabGroupLdapLink_recreatedWhenRemoved(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	ldapName := acctest.RandomWithPrefix("ldap")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupLdapLinkDestroy,
		Steps: []resource.TestStep{
			// Create an LDAP group link
			{
				Config: fmt.Sprintf(`
          resource "gitlab_group_ldap_link" "test" {
            group         = "%[1]d"
            cn            = "%[2]s"
            group_access  = "developer"
            ldap_provider = "%[2]s"
          }
          `, testGroup.ID, ldapName),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_ldap_link.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
			// Remove the LDAP link directly and re-apply the config to re-create the LDAP link
			{
				PreConfig: func() {
					if _, err := testutil.TestGitlabClient.Groups.DeleteGroupLDAPLink(testGroup.ID, ldapName); err != nil {
						t.Fatalf("Failed to delete LDAP link %q in group %d", ldapName, testGroup.ID)
					}
				},
				Config: fmt.Sprintf(`
          resource "gitlab_group_ldap_link" "test" {
            group         = "%[1]d"
            cn            = "%[2]s"
            group_access  = "developer"
            ldap_provider = "%[2]s"
          }
          `, testGroup.ID, ldapName),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_ldap_link.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
			// Create an LDAP group link with a namespace instead of ID
			{
				Config: fmt.Sprintf(`
          resource "gitlab_group_ldap_link" "test" {
            group         = "%s"
            cn            = "%[2]s"
            group_access  = "developer"
            ldap_provider = "%[2]s"
          }
          `, testGroup.Path, ldapName),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_ldap_link.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
		},
	})
}

func TestAccGitlabGroupLdapLink_forceDeletesWhenExists(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create a pre-existing LDAP link with the same metadata
	testGroup := testutil.CreateGroups(t, 1)[0]

	level := gitlab.DeveloperPermissions
	_, _, err := testutil.TestGitlabClient.Groups.AddGroupLDAPLink(testGroup.ID, &gitlab.AddGroupLDAPLinkOptions{
		CN:          gitlab.Ptr("default"),
		GroupAccess: &level,
		Provider:    gitlab.Ptr("default"),
	})
	if err != nil {
		t.Fatalf("Failed to create testing LDAP group: %v", err)
	}
	// Note: No cleanup is required for the LDAP provider because it's deleted when the Group is.

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupLdapLinkDestroy,
		Steps: []resource.TestStep{
			// Attempt to create an LDAP Group link without "force", which will receive an error
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_ldap_link" "test" {
						group         = "%[1]d"
						cn            = "default"
						group_access  = "developer"
						ldap_provider = "default"
					}
          		`, testGroup.ID),
				ExpectError: regexp.MustCompile("Cn has already been taken"),
			},
			// Attempt to create an LDAP Group link with "force", which should delete/re-create the link
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_ldap_link" "test" {
						group         = "%[1]d"
						cn            = "default"
						group_access  = "developer"
						ldap_provider = "default"
						force         = true
					}
				`, testGroup.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_ldap_link.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force",
				},
			},
		},
	})
}

func TestAccGitlabGroupLdapLink_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV0State    map[string]interface{}
		expectedV1State map[string]interface{}
	}{
		{
			name: "Project With ID and CN",
			givenV0State: map[string]interface{}{
				"group":         "99",
				"cn":            "mainScreenTurnOn",
				"ldap_provider": "allYourBase",
				"filter":        "",
				"id":            "allYourBase:mainScreenTurnOn",
			},
			expectedV1State: map[string]interface{}{
				"group":         "99",
				"cn":            "mainScreenTurnOn",
				"ldap_provider": "allYourBase",
				"filter":        "",
				"id":            "99:allYourBase:mainScreenTurnOn:",
			},
		},
		{
			name: "Project With ID and Filter",
			givenV0State: map[string]interface{}{
				"group":         "99",
				"cn":            "",
				"filter":        "thisIsAFilter",
				"ldap_provider": "allYourBase",
				"id":            "allYourBase:mainScreenTurnOn",
			},
			expectedV1State: map[string]interface{}{
				"group":         "99",
				"cn":            "",
				"filter":        "thisIsAFilter",
				"ldap_provider": "allYourBase",
				"id":            "99:allYourBase::thisIsAFilter",
			},
		},
		{
			name: "Project With ID and Filter using old group_id",
			givenV0State: map[string]interface{}{
				"group_id":      "99",
				"cn":            "",
				"filter":        "thisIsAFilter",
				"ldap_provider": "allYourBase",
				"id":            "allYourBase:mainScreenTurnOn",
			},
			expectedV1State: map[string]interface{}{
				"group_id":      "99",
				"cn":            "",
				"filter":        "thisIsAFilter",
				"ldap_provider": "allYourBase",
				"id":            "99:allYourBase::thisIsAFilter",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV1State, err := resourceGitlabGroupLDAPLinkStateUpgradeV0(context.Background(), tc.givenV0State, nil)
			if err != nil {
				t.Fatalf("Error migrating state: %s", err)
			}

			if !reflect.DeepEqual(tc.expectedV1State, actualV1State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV1State, actualV1State)
			}
		})
	}
}

func testAccCheckGitlabGroupLdapLinkExists(resourceName string, ldapLink *gitlab.LDAPGroupLink) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Clear the "found" LDAP link before checking for existence
		*ldapLink = gitlab.LDAPGroupLink{}

		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		err := testAccGetGitlabGroupLdapLink(ldapLink, resourceState)
		if err != nil {
			return err
		}

		return nil
	}
}

type testAccGitlabGroupLdapLinkExpectedAttributes struct {
	accessLevel string
}

func testAccCheckGitlabGroupLdapLinkAttributes(ldapLink *gitlab.LDAPGroupLink, want *testAccGitlabGroupLdapLinkExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		accessLevelId, ok := api.AccessLevelValueToName[ldapLink.GroupAccess]
		if !ok {
			return fmt.Errorf("Invalid access level '%s'", accessLevelId)
		}
		if accessLevelId != want.accessLevel {
			return fmt.Errorf("Has access level %s; want %s", accessLevelId, want.accessLevel)
		}
		return nil
	}
}

func testAccCheckGitlabGroupLdapLinkDestroy(s *terraform.State) error {
	// Can't check for links if the group is destroyed so make sure all groups are destroyed instead
	for _, resourceState := range s.RootModule().Resources {
		if resourceState.Type != "gitlab_group" {
			continue
		}

		group, _, err := testutil.TestGitlabClient.Groups.GetGroup(resourceState.Primary.ID, nil)
		if err == nil {
			if group != nil && fmt.Sprintf("%d", group.ID) == resourceState.Primary.ID {
				if group.MarkedForDeletionOn == nil {
					return fmt.Errorf("Group still exists")
				}
			}
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}

func testAccGetGitlabGroupLdapLink(ldapLink *gitlab.LDAPGroupLink, resourceState *terraform.ResourceState) error {
	group := resourceState.Primary.Attributes["group"]
	if group == "" {
		return fmt.Errorf("No group ID is set")
	}

	// Construct our desired LDAP Link from the config values
	desiredLdapLink := gitlab.LDAPGroupLink{
		CN:          resourceState.Primary.Attributes["cn"],
		GroupAccess: api.AccessLevelNameToValue[resourceState.Primary.Attributes["group_access"]],
		Provider:    resourceState.Primary.Attributes["ldap_provider"],
	}

	desiredLdapLinkId := utils.BuildTwoPartID(&desiredLdapLink.Provider, &desiredLdapLink.CN)

	// Try to fetch all group links from GitLab
	currentLdapLinks, _, err := testutil.TestGitlabClient.Groups.ListGroupLDAPLinks(group, nil)
	if err != nil {
		return err
	}

	// If we got here and don't have links, assume GitLab is below version 12.8 and skip the check
	if currentLdapLinks != nil {
		found := false

		// Check if the LDAP link exists in the returned list of links
		for _, currentLdapLink := range currentLdapLinks {
			if utils.BuildTwoPartID(&currentLdapLink.Provider, &currentLdapLink.CN) == desiredLdapLinkId {
				found = true
				*ldapLink = *currentLdapLink
				break
			}
		}

		if !found {
			return fmt.Errorf("LdapLink %s does not exist.", desiredLdapLinkId)
		}
	} else {
		*ldapLink = desiredLdapLink
	}

	return nil
}
