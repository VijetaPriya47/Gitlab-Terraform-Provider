//go:build acceptance

package sdk

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabDeployToken_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV0State    map[string]interface{}
		expectedV1State map[string]interface{}
	}{
		{
			name: "project deploy token",
			givenV0State: map[string]interface{}{
				"project": "999",
				"id":      "42",
			},
			expectedV1State: map[string]interface{}{
				"project": "999",
				"id":      "project:999:42",
			},
		},
		{
			name: "project deploy token",
			givenV0State: map[string]interface{}{
				"project": "foo/bar",
				"id":      "42",
			},
			expectedV1State: map[string]interface{}{
				"project": "foo/bar",
				"id":      "project:foo/bar:42",
			},
		},
		{
			name: "project deploy token with empty group in state",
			givenV0State: map[string]interface{}{
				"project": "foo/bar",
				"group":   "",
				"id":      "42",
			},
			expectedV1State: map[string]interface{}{
				"project": "foo/bar",
				"group":   "",
				"id":      "project:foo/bar:42",
			},
		},
		{
			name: "group deploy token",
			givenV0State: map[string]interface{}{
				"group": "foo/bar",
				"id":    "42",
			},
			expectedV1State: map[string]interface{}{
				"group": "foo/bar",
				"id":    "group:foo/bar:42",
			},
		},
		{
			name: "group deploy token with empty project in state",
			givenV0State: map[string]interface{}{
				"group":   "foo/bar",
				"project": "",
				"id":      "42",
			},
			expectedV1State: map[string]interface{}{
				"group":   "foo/bar",
				"project": "",
				"id":      "group:foo/bar:42",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV1State, err := resourceGitlabDeployTokenStateUpgradeV0(context.Background(), tc.givenV0State, nil)
			if err != nil {
				t.Fatalf("Error migrating state: %s", err)
			}

			if !reflect.DeepEqual(tc.expectedV1State, actualV1State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV1State, actualV1State)
			}
		})
	}
}

func TestAccGitlabDeployToken_SchemaMigration0_1(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testGroup := testutil.CreateGroups(t, 1)[0]

	config := fmt.Sprintf(`
	resource "gitlab_deploy_token" "project_token" {
	  project  = "%d"
	  name     = "project-deploy-token"
	  username = "my-username"
	
	  expires_at = "2021-03-14T07:20:50.000Z"
	
	  scopes = [
		"read_registry",
		"read_repository",
		"read_package_registry",
		"write_registry",
		"write_package_registry",
	  ]
	}
	
	resource "gitlab_deploy_token" "group_token" {
	  group  = "%d"
	  name     = "group-deploy-token"
	  username = "my-username"
	
	  expires_at = "2021-03-14T07:20:50.000Z"
	
	  scopes = [
		"read_registry",
		"read_repository",
		"read_package_registry",
		"write_registry",
		"write_package_registry",
	  ]
	}
	  `, testProject.ID, testGroup.ID)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabDeployTokenDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 15.7.0", // Earliest 15.X deployment
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
			},
			{
				ProtoV6ProviderFactories: providerFactoriesV6,
				Config:                   config,
				PlanOnly:                 true,
			},
		},
	})
}

func TestAccGitlabDeployToken_basic(t *testing.T) {
	var projectDeployToken gitlab.DeployToken
	var groupDeployToken gitlab.DeployToken

	testProject := testutil.CreateProject(t)
	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabDeployTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_deploy_token" "project_token" {
						project  = "%d"
						name     = "project-deploy-token"
						username = "my-username"
						
						expires_at = "2021-03-14T07:20:50.000Z"
						
						scopes = [
							"read_registry",
							"read_repository",
							"read_package_registry",
							"write_registry",
							"write_package_registry",
						]
					}
					
					resource "gitlab_deploy_token" "group_token" {
						group  = "%d"
						name     = "group-deploy-token"
						username = "my-username"
						
						expires_at = "2021-03-14T07:20:50.000Z"
						
						scopes = [
							"read_registry",
							"read_repository",
							"read_package_registry",
							"write_registry",
							"write_package_registry",
						]
					}
				`, testProject.ID, testGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabDeployTokenExists("gitlab_deploy_token.project_token", &projectDeployToken),
					resource.TestCheckResourceAttrSet("gitlab_deploy_token.project_token", "token"),
					resource.TestCheckResourceAttrSet("gitlab_deploy_token.project_token", "username"),
					testAccCheckGitlabDeployTokenExists("gitlab_deploy_token.group_token", &groupDeployToken),
					resource.TestCheckResourceAttrSet("gitlab_deploy_token.group_token", "token"),
					resource.TestCheckResourceAttrSet("gitlab_deploy_token.group_token", "username"),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_deploy_token.project_token",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			{
				ResourceName:            "gitlab_deploy_token.group_token",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAccGitlabDeployToken_nousername(t *testing.T) {
	var projectDeployToken gitlab.DeployToken
	var groupDeployToken gitlab.DeployToken

	testProject := testutil.CreateProject(t)
	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabDeployTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_deploy_token" "project_token" {
						project = "%d"
						name    = "project-deploy-token"
						
						expires_at = "2021-03-14T07:20:50.000Z"
						
						scopes = [
							"read_registry",
							"read_repository",
							"read_package_registry",
							"write_registry",
							"write_package_registry",
						]
					}
					
					resource "gitlab_deploy_token" "group_token" {
						group = "%d"
						name  = "group-deploy-token"
						
						expires_at = "2021-03-14T07:20:50.000Z"
						
						scopes = [
							"read_registry",
							"read_repository",
							"read_package_registry",
							"write_registry",
							"write_package_registry",
						]
					}
				`, testProject.ID, testGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabDeployTokenExists("gitlab_deploy_token.project_token", &projectDeployToken),
					resource.TestCheckResourceAttrSet("gitlab_deploy_token.project_token", "token"),
					resource.TestCheckResourceAttrSet("gitlab_deploy_token.project_token", "username"),
					testAccCheckGitlabDeployTokenExists("gitlab_deploy_token.group_token", &groupDeployToken),
					resource.TestCheckResourceAttrSet("gitlab_deploy_token.group_token", "token"),
					resource.TestCheckResourceAttrSet("gitlab_deploy_token.group_token", "username"),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_deploy_token.project_token",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			{
				ResourceName:            "gitlab_deploy_token.group_token",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAccGitlabDeployToken_pagination(t *testing.T) {
	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabDeployTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_deploy_token" "example_group" {
						group  = %d
						name   = "deploy-token-${count.index}"
						scopes = ["read_registry"]
						
						count = %d
					}
					
					resource "gitlab_deploy_token" "example_project" {
						project  = %d
						name   = "deploy-token-${count.index}"
						scopes = ["read_registry"]
						
						count = %d
					}
				`, testGroup.ID, 25, testProject.ID, 25),
			},
			// In case pagination wouldn't properly work, we would get that the plan isn't empty,
			// because some of the deploy tokens wouldn't be in the first page and therefore
			// considered non-existing, ...
			{
				Config: fmt.Sprintf(`
					resource "gitlab_deploy_token" "example_group" {
						group  = %d
						name   = "deploy-token-${count.index}"
						scopes = ["read_registry"]
						
						count = %d
					}
					
					resource "gitlab_deploy_token" "example_project" {
						project  = %d
						name   = "deploy-token-${count.index}"
						scopes = ["read_registry"]
						
						count = %d
					}
				`, testGroup.ID, 25, testProject.ID, 25),
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckGitlabDeployTokenExists(n string, deployToken *gitlab.DeployToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		deployTokenType, typeId, deployTokenId, err := resourceGitlabDeployTokenParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		switch deployTokenType {
		case "project":
			deployToken, _, err = testutil.TestGitlabClient.DeployTokens.GetProjectDeployToken(typeId, deployTokenId)
		case "group":
			deployToken, _, err = testutil.TestGitlabClient.DeployTokens.GetGroupDeployToken(typeId, deployTokenId)
		default:
			return fmt.Errorf("No project or group ID is set")
		}

		if err != nil {
			if api.Is404(err) {
				return fmt.Errorf("Deploy Token doesn't exist")
			}
			return err

		}
		return nil
	}
}

func testAccCheckGitlabDeployTokenDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_deploy_token" {
			continue
		}

		deployTokenType, typeId, deployTokenId, err := resourceGitlabDeployTokenParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		switch deployTokenType {
		case "project":
			_, _, err = testutil.TestGitlabClient.DeployTokens.GetProjectDeployToken(typeId, deployTokenId)
		case "group":
			_, _, err = testutil.TestGitlabClient.DeployTokens.GetGroupDeployToken(typeId, deployTokenId)
		default:
			return fmt.Errorf("No project or group ID is set")
		}

		if err == nil {
			return fmt.Errorf("Deploy token still exists")
		}

		if !api.Is404(err) {
			return err
		}
	}

	return nil
}

type expiresAtSuppressFuncTest struct {
	description string
	old         string
	new         string
	expected    bool
}

func TestExpiresAtSuppressFunc(t *testing.T) {
	t.Parallel()

	testcases := []expiresAtSuppressFuncTest{
		{
			description: "same dates without millis",
			old:         "2025-03-14T00:00:00Z",
			new:         "2025-03-14T00:00:00Z",
			expected:    true,
		}, {
			description: "different date without millis",
			old:         "2025-03-14T00:00:00Z",
			new:         "2025-03-14T11:11:11Z",
			expected:    false,
		}, {
			description: "same date with and without millis",
			old:         "2025-03-14T00:00:00Z",
			new:         "2025-03-14T00:00:00.000Z",
			expected:    true,
		}, {
			description: "cannot parse new date",
			old:         "2025-03-14T00:00:00Z",
			new:         "invalid-date",
			expected:    false,
		},
	}

	for _, test := range testcases {
		t.Run(test.description, func(t *testing.T) {
			actual := expiresAtSuppressFunc("", test.old, test.new, nil)
			if actual != test.expected {
				t.Fatalf("FAIL\n\told: %s, new: %s\n\texpected: %t\n\tactual: %t",
					test.old, test.new, test.expected, actual)
			}
		})
	}
}
