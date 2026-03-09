//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabProjectAccessTokens_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	projectAccessTokens := make([]*gitlab.ProjectAccessToken, 0)
	for i := range 5 {
		projectAccessTokens = append(projectAccessTokens, testutil.CreateProjectAccessToken(
			t, project.ID, fmt.Sprintf("basic-%d", i), []string{"read_repository", "read_registry"}, gitlab.MaintainerPermissions, gitlab.Ptr(fmt.Sprintf("basic-%d", i))),
		)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_access_tokens" "this" {
						project = %d
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.#", fmt.Sprintf("%d", len(projectAccessTokens))),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.id", strconv.FormatInt(projectAccessTokens[0].ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.name", projectAccessTokens[0].Name),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.4.id", strconv.FormatInt(projectAccessTokens[4].ID, 10)),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabProjectAccessTokens_attributes(t *testing.T) {
	project := testutil.CreateProject(t)
	projectAccessTokens := make([]*gitlab.ProjectAccessToken, 0)
	for i := range 2 {
		projectAccessTokens = append(projectAccessTokens, testutil.CreateProjectAccessToken(
			t, project.ID, fmt.Sprintf("basic-%d", i), []string{"read_api", "read_registry"}, gitlab.MaintainerPermissions, gitlab.Ptr(fmt.Sprintf("attribute-%d", i))),
		)
	}

	// Create a gitlab Client with one of the created project access token
	patClient := testutil.CreateGitlabClientWithToken(t, projectAccessTokens[1].Token)
	// Use the token once to populate last_used_at for this token
	_, _, err := patClient.Repositories.Contributors(project.ID, &gitlab.ListContributorsOptions{})
	if err != nil {
		t.Fatal(err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_access_tokens" "this" {
						project = %d
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.description", projectAccessTokens[0].Description),
					resource.TestCheckTypeSetElemAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.scopes.*", projectAccessTokens[0].Scopes[0]),
					resource.TestCheckTypeSetElemAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.scopes.*", projectAccessTokens[0].Scopes[1]),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.access_level", "maintainer"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_access_tokens.this", "access_tokens.0.created_at"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_access_tokens.this", "access_tokens.0.expires_at"),
					resource.TestCheckNoResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.last_used_at"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_access_tokens.this", "access_tokens.1.last_used_at"),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabProjectAccessTokens_state(t *testing.T) {
	project := testutil.CreateProject(t)
	projectAccessTokens := make([]*gitlab.ProjectAccessToken, 0)
	for i := range 2 {
		projectAccessTokens = append(projectAccessTokens, testutil.CreateProjectAccessToken(
			t, project.ID, fmt.Sprintf("state-%d", i), []string{"read_api", "write_registry"}, gitlab.OwnerPermissions, nil),
		)
	}
	// Revoke the first token to test the state filter
	if _, err := testutil.TestGitlabClient.ProjectAccessTokens.RevokeProjectAccessToken(project.ID, projectAccessTokens[0].ID); err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_access_tokens" "this" {
						project = %d
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.#", fmt.Sprintf("%d", len(projectAccessTokens))),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.id", strconv.FormatInt(projectAccessTokens[0].ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.active", "false"),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.revoked", "true"),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.1.active", "true"),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.1.revoked", "false"),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_access_tokens" "this" {
						project = %d
						state  = "inactive"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.#", fmt.Sprintf("%d", len(projectAccessTokens)-1)),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.id", strconv.FormatInt(projectAccessTokens[0].ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.active", "false"),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.revoked", "true"),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_access_tokens" "this" {
						project = %d
						state  = "active"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.#", fmt.Sprintf("%d", len(projectAccessTokens)-1)),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.id", strconv.FormatInt(projectAccessTokens[1].ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.active", "true"),
					resource.TestCheckResourceAttr("data.gitlab_project_access_tokens.this", "access_tokens.0.revoked", "false"),
				),
			},
		},
	})
}
