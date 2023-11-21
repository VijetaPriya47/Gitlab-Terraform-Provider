//go:build acceptance
// +build acceptance

package sdk

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabInstanceDeployKeys_basic(t *testing.T) {
	testKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCi+ErxScCKIVqg2ZRJ6Mx2Yd/RTsh2DGyhUR8z8Iey4rpi1YOBlpTgjxxnSLy26J++Un/iWYDP8wMvEjXElkWz3z4I+Z3mfF3dv039FTCu+O17Mw20Ek4DJxdrKvOgul040sUG/ABVHo6DjqjokjoVJwzUrUmoOtbeMMD8hFN9bWdEVyTj18XQO8nvEe/VkbhCRhAlZC1l60fM07/7Tw83SV5UNAnBtOB+nfa3b24baO+Ijc4+PqYcBuUAF6DvhXW2gZPqf5wjDBJqlDlRTYDdHarMXZAKBpWfWj0gntbtEOM+Fnp6hS1HajaeveNSs6yQwgQEDN2boQnDuvXJ8Y7zW3YQKZp8z0uqWYJSIrYRKVEVYL7gDWL9NvdRV52d/RKPnE/BlL2chiAWBRCT8buQdjVtEPPoYbA1667PXZg6PI9yhCGEIjCj71XzPssA6VL/R7yUafsmNLsirWz9Uyh3HJWCcgNuO9mglP5nfFHIXSHQVhEUEYMfzv1iX5FrenU= test"
	testKey2 := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDStVqW58VZ5afXFphIvu2JahndXslJZMkgWsNiYCNdk/NvrEbc4i7yZVoDPFQsbS9I6Ty1RMW7qy3KxJalMsVHcw8arCQFDxs/ka1NHGCUPl68t5ZxUOl900KRQ0lOzGnDQMqG/UUZdPw4CCmigTr6Z9ZBcD1fXAiUwbXR4tWrr5z9KWXC2HgF4WkIJUTIct7ilY1m9W0y79dI/+K8bZrurn3q2QK83pxqqWkLwvUsCxtlhMpwuyflyzyuz8xPZl2GlZgxeIpr68gsPHIzzWizibwFfbRYKCZO4wD0r7JCDOYs9KjcIPpCG6d3HUqijClgdQSBnLwHTdE04ZtdzO8akvy0hMzRCooI5TSc8IAHos53Gp9aaW92sPA8za+WRP6OSH6UsOW4N+iQc4jyl7/fckMSgIZlJouNqqV+P8iqIFJGs70Tj5L8G/m+P2lc3kcE4Vjmj+Fc0xG5+I/PsSOpcc6DfDfZdVDRe8yklYd/qC1jI89OCeqjxu3XcUGHj9s= test"
	testProjectTwoKeys := testutil.CreateProject(t)
	testProjectOneKey := testutil.CreateProject(t)
	canPushDeployKeyOptions := gitlab.AddDeployKeyOptions{
		Title:   gitlab.Ptr("Can Push"),
		Key:     gitlab.Ptr(testKey),
		CanPush: gitlab.Ptr(true),
	}
	canNotPushDeployKeyOptions := gitlab.AddDeployKeyOptions{
		Title:   gitlab.Ptr("Can Not Push"),
		Key:     gitlab.Ptr(testKey2),
		CanPush: gitlab.Ptr(false),
	}

	testutil.CreateDeployKey(t, testProjectTwoKeys.ID, &canPushDeployKeyOptions)
	testutil.CreateDeployKey(t, testProjectTwoKeys.ID, &canNotPushDeployKeyOptions)
	testutil.CreateDeployKey(t, testProjectOneKey.ID, &canPushDeployKeyOptions)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: `
					data "gitlab_instance_deploy_keys" "this" {}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_instance_deploy_keys.this", "deploy_keys.#", "2"),
					resource.TestCheckResourceAttr("data.gitlab_instance_deploy_keys.this", "deploy_keys.0.title", "Can Push"),
					resource.TestCheckResourceAttr("data.gitlab_instance_deploy_keys.this", "deploy_keys.0.projects_with_write_access.#", "2"),
					resource.TestCheckResourceAttr("data.gitlab_instance_deploy_keys.this", "deploy_keys.1.title", "Can Not Push"),
					resource.TestCheckResourceAttr("data.gitlab_instance_deploy_keys.this", "deploy_keys.1.projects_with_write_access.#", "0"),
				),
			},
			{
				Config: `
					data "gitlab_instance_deploy_keys" "this" {
						public = true
					}
				`,
				Check: resource.TestCheckResourceAttr("data.gitlab_instance_deploy_keys.this", "deploy_keys.#", "0"),
			},
		},
	})
}
