//go:build acceptance || flakey || settings || saas
// +build acceptance flakey settings saas

package framework

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestCheckResourceAttrSetIfGitLabAtLeast(t *testing.T, requiredMinVersion, name, key string) resource.TestCheckFunc {
	isAtLeast, err := api.IsGitLabVersionAtLeast(context.Background(), testutil.TestGitlabClient, requiredMinVersion)()
	if err != nil {
		t.Fatalf("Failed to fetch GitLab version: %+v", err)
	}

	if !isAtLeast {
		return NopTestCheckFunc
	}

	return resource.TestCheckResourceAttrSet(name, key)
}

func NopTestCheckFunc(*terraform.State) error { return nil }

// This helper function sets the `TF_ACC_TERRAFORM_VERSION` env variable
// that tells the test framework to run the test with a specific Terraform
// version. It uses a cleanup function to unset it when the test finishes.
// It accepts a TestCase to ensure that the test doesn't run with Parallel
// to prevent polluting other tests
func RunTestWithVersion(t *testing.T, version string, testcase resource.TestCase) {
	t.Helper()

	t.Setenv("TF_ACC_TERRAFORM_VERSION", version)

	resource.Test(t, testcase)
}
