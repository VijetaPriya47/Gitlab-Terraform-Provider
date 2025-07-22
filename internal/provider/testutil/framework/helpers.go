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
