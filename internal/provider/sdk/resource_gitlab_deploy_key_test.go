//go:build acceptance

package sdk

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabDeployKey_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV0State    map[string]any
		expectedV1State map[string]any
	}{
		{
			name: "Project With ID",
			givenV0State: map[string]any{
				"project": "99",
				"id":      "42",
			},
			expectedV1State: map[string]any{
				"project": "99",
				"id":      "99:42",
			},
		},
		{
			name: "Project With Namespace",
			givenV0State: map[string]any{
				"project": "foo/bar",
				"id":      "42",
			},
			expectedV1State: map[string]any{
				"project": "foo/bar",
				"id":      "foo/bar:42",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV1State, err := resourceGitlabProjectDeployKeyStateUpgradeV0(context.Background(), tc.givenV0State, nil)
			if err != nil {
				t.Fatalf("Error migrating state: %s", err)
			}

			if !reflect.DeepEqual(tc.expectedV1State, actualV1State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV1State, actualV1State)
			}
		})
	}
}

func TestAccGitlabDeployKey_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabDeployKeyDestroy,
		Steps: []resource.TestStep{
			// Create a project and deployKey with default options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_deploy_key" "foo" {
						project = %[3]d
						title = "deployKey-%[1]d"
						key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCj13ozEBZ0s4el4k6mYqoyIKKKMh9hHY0sAYqSPXs2zGuVFZss1P8TPuwmdXVjHR7TiRXwC49zDrkyWJgiufggYJ1VilOohcMOODwZEJz+E5q4GCfHuh90UEh0nl8B2R0Uoy0LPeg93uZzy0hlHApsxRf/XZJz/1ytkZvCtxdllxfImCVxJReMeRVEqFCTCvy3YuJn0bce7ulcTFRvtgWOpQsr6GDK8YkcCCv2eZthVlrEwy6DEpAKTRiRLGgUj4dPO0MmO4cE2qD4ualY01PhNORJ8Q++I+EtkGt/VALkecwFuBkl18/gy+yxNJHpKc/8WVVinDeFrd/HhiY9yU0d richardc@tamborine.example.1%[2]s"
					}
				`, rInt, "", testProject.ID),
			},
			// Verify import
			{
				ResourceName:      "gitlab_deploy_key.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the project deployKey to toggle all the values to their inverse
			{
				Config: fmt.Sprintf(`
					resource "gitlab_deploy_key" "foo" {
						project = %[2]d
						title = "modifiedDeployKey-%[1]d"
						key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC6pSke2kb7YBjo65xDKegbOQsAtnMupRcFxXji7L1iXivGwORq0qpC2xzbhez5jk1WgPckEaNv2/Bz0uEW6oSIXw1KT1VN2WzEUfQCbpNyZPtn4iV3nyl6VQW/Nd1SrxiFJtH1H4vu+eCo4McMXTjuBBD06fiJNrHaSw734LjQgqtXWJuVym9qS5MqraZB7wDwTQwSM6kslL7KTgmo3ONsTLdb2zZhv6CS+dcFKinQo7/ttTmeMuXGbPOVuNfT/bePVIN1MF1TislHa2L2dZdGeoynNJT4fVPjA2Xl6eHWh4ySbvnfPznASsjBhP0n/QKprYJ/5fQShdBYBcuQiIMd richardc@tamborine.example.2"
						can_push = true
					}
				`, rInt, testProject.ID),
			},
			// Verify import
			{
				ResourceName:      "gitlab_deploy_key.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the project deployKey to toggle the options back
			{
				Config: fmt.Sprintf(`
					resource "gitlab_deploy_key" "foo" {
						project = %[3]d
						title = "deployKey-%[1]d"
						key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCj13ozEBZ0s4el4k6mYqoyIKKKMh9hHY0sAYqSPXs2zGuVFZss1P8TPuwmdXVjHR7TiRXwC49zDrkyWJgiufggYJ1VilOohcMOODwZEJz+E5q4GCfHuh90UEh0nl8B2R0Uoy0LPeg93uZzy0hlHApsxRf/XZJz/1ytkZvCtxdllxfImCVxJReMeRVEqFCTCvy3YuJn0bce7ulcTFRvtgWOpQsr6GDK8YkcCCv2eZthVlrEwy6DEpAKTRiRLGgUj4dPO0MmO4cE2qD4ualY01PhNORJ8Q++I+EtkGt/VALkecwFuBkl18/gy+yxNJHpKc/8WVVinDeFrd/HhiY9yU0d richardc@tamborine.example.1%[2]s"
					}
				`, rInt, "", testProject.ID),
			},
			// Verify import
			{
				ResourceName:      "gitlab_deploy_key.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add an expiry
			{
				Config: fmt.Sprintf(`
					resource "gitlab_deploy_key" "foo" {
						project = %[3]d
						title = "deployKey-%[1]d"
						key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCj13ozEBZ0s4el4k6mYqoyIKKKMh9hHY0sAYqSPXs2zGuVFZss1P8TPuwmdXVjHR7TiRXwC49zDrkyWJgiufggYJ1VilOohcMOODwZEJz+E5q4GCfHuh90UEh0nl8B2R0Uoy0LPeg93uZzy0hlHApsxRf/XZJz/1ytkZvCtxdllxfImCVxJReMeRVEqFCTCvy3YuJn0bce7ulcTFRvtgWOpQsr6GDK8YkcCCv2eZthVlrEwy6DEpAKTRiRLGgUj4dPO0MmO4cE2qD4ualY01PhNORJ8Q++I+EtkGt/VALkecwFuBkl18/gy+yxNJHpKc/8WVVinDeFrd/HhiY9yU0d richardc@tamborine.example.1%[2]s"
						expires_at = "3016-01-21T00:00:00Z"
					}
				`, rInt, "", testProject.ID),
			},
			// Verify import
			{
				ResourceName:      "gitlab_deploy_key.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabDeployKey_suppressTrailingSpace(t *testing.T) {
	testProject := testutil.CreateProject(t)
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabDeployKeyDestroy,
		Steps: []resource.TestStep{
			// Create a project and deployKey with space as suffix
			{
				Config: fmt.Sprintf(`
					resource "gitlab_deploy_key" "foo" {
						project = %[3]d
						title = "deployKey-%[1]d"
						key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCj13ozEBZ0s4el4k6mYqoyIKKKMh9hHY0sAYqSPXs2zGuVFZss1P8TPuwmdXVjHR7TiRXwC49zDrkyWJgiufggYJ1VilOohcMOODwZEJz+E5q4GCfHuh90UEh0nl8B2R0Uoy0LPeg93uZzy0hlHApsxRf/XZJz/1ytkZvCtxdllxfImCVxJReMeRVEqFCTCvy3YuJn0bce7ulcTFRvtgWOpQsr6GDK8YkcCCv2eZthVlrEwy6DEpAKTRiRLGgUj4dPO0MmO4cE2qD4ualY01PhNORJ8Q++I+EtkGt/VALkecwFuBkl18/gy+yxNJHpKc/8WVVinDeFrd/HhiY9yU0d richardc@tamborine.example.1%[2]s"
					}
				`, rInt, " ", testProject.ID),
			},
			// Verify import
			{
				ResourceName:      "gitlab_deploy_key.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabDeployKeyDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		project, deployKeyID, err := resourceGitlabProjectDeployKeyParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("unable to parse deploy key resource id: %w", err)
		}

		gotDeployKey, _, err := testutil.TestGitlabClient.DeployKeys.GetDeployKey(project, deployKeyID)
		if err == nil {
			if gotDeployKey != nil && fmt.Sprintf("%d", gotDeployKey.ID) == rs.Primary.ID {
				return fmt.Errorf("Deploy key still exists")
			}
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
