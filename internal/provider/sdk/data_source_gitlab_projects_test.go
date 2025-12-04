//go:build acceptance

package sdk

import (
	"fmt"
	"strconv"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjects_search(t *testing.T) {
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_projects" "search" {
				  search = "%s"
				}
				`, project.Name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.0.id", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.0.owner.0.id", "1"),
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.0.permissions.0.project_access.access_level", "50"),
					resource.TestCheckNoResourceAttr("data.gitlab_projects.search", "projects.0.permissions.0.project_access.group_level"),
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.0.namespace.0.kind", "user"),
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.0.prevent_merge_without_jira_issue", "false"),
				),
			},
		},
	})
}

func TestAccDataGitlabProjects_groups(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	subgroups := testutil.CreateSubGroups(t, group, 2)
	top_group_project := testutil.CreateProjectWithNamespace(t, group.ID)
	subgroup1_project := testutil.CreateProjectWithNamespace(t, subgroups[0].ID)
	subgroup2_project := testutil.CreateProjectWithNamespace(t, subgroups[1].ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_projects" "group" {
				  group_id = %d
				}
				`, group.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_projects.group", "projects.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_projects.group", "group_id", fmt.Sprintf("%d", group.ID)),
					resource.TestCheckResourceAttr("data.gitlab_projects.group", "projects.0.namespace.0.kind", "group"),
				),
			},
			{
				Config: fmt.Sprintf(`
				data "gitlab_projects" "subGroups" {
				  group_id = %d
				  include_subgroups = true
				}
				`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_projects.subGroups", "projects.#", "3"),
					resource.TestCheckResourceAttr("data.gitlab_projects.subGroups", "projects.0.id", fmt.Sprintf("%d", subgroup2_project.ID)),
					resource.TestCheckResourceAttr("data.gitlab_projects.subGroups", "projects.1.id", fmt.Sprintf("%d", subgroup1_project.ID)),
					resource.TestCheckResourceAttr("data.gitlab_projects.subGroups", "projects.2.id", fmt.Sprintf("%d", top_group_project.ID)),
					resource.TestCheckResourceAttr("data.gitlab_projects.subGroups", "group_id", fmt.Sprintf("%d", group.ID)),
				),
			},
		},
	})
}

func TestAccDataGitlabProjects_searchArchivedRepository(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	project := testutil.CreateProjectWithNamespace(t, group.ID)
	archivedProject := testutil.CreateProjectWithNamespace(t, group.ID)
	_, _, err := testutil.TestGitlabClient.Projects.ArchiveProject(archivedProject.ID)
	if err != nil {
		t.Fatalf("error archiving test project: %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_projects" "search" {
				  group_id = %d
				}
					`, group.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.#", "2"),
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.0.name", archivedProject.Name),
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.1.name", project.Name),
				),
			},
			{
				Config: fmt.Sprintf(`
				data "gitlab_projects" "search" {
				  group_id = %d
				
				  archived = true
				}
					`, group.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.0.name", archivedProject.Name),
				),
			},
			{
				Config: fmt.Sprintf(`
				data "gitlab_projects" "search" {
				  group_id = %d
				
				  archived = false
				}
					`, group.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_projects.search", "projects.0.name", project.Name),
				),
			},
		},
	})
}

func TestAccDataGitlabProjects_topic(t *testing.T) {
	topic1 := testutil.CreateTopic(t)
	topic2 := testutil.CreateTopic(t)
	topic3 := testutil.CreateTopic(t)

	topicSlice := func(topics ...*gitlab.Topic) *[]string {
		s := make([]string, len(topics))
		for i, t := range topics {
			s[i] = t.Name
		}
		return &s
	}

	project1 := testutil.CreateProject(t)
	_, _, err := testutil.TestGitlabClient.Projects.EditProject(project1.ID, &gitlab.EditProjectOptions{
		Topics: topicSlice(topic1),
	})
	if err != nil {
		t.Fatalf("error adding topics to test project: %v", err)
	}

	project12 := testutil.CreateProject(t)
	_, _, err = testutil.TestGitlabClient.Projects.EditProject(project12.ID, &gitlab.EditProjectOptions{
		Topics: topicSlice(topic1, topic2),
	})
	if err != nil {
		t.Fatalf("error adding topics to test project: %v", err)
	}

	project13 := testutil.CreateProject(t)
	_, _, err = testutil.TestGitlabClient.Projects.EditProject(project13.ID, &gitlab.EditProjectOptions{
		Topics: topicSlice(topic1, topic3),
	})
	if err != nil {
		t.Fatalf("error adding topics to test project: %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			// Match multiple projects.
			{
				Config: fmt.Sprintf(`
data "gitlab_projects" "search" {
  topic = ["%s"]
}`, topic1.Name),
				Check: testAccDataSourceGitlabProjectsContainsProjects("data.gitlab_projects.search", project1, project12, project13),
			},
			// Match one project.
			{
				Config: fmt.Sprintf(`
data "gitlab_projects" "search" {
  topic = ["%s"]
}`, topic2.Name),
				Check: testAccDataSourceGitlabProjectsContainsProjects("data.gitlab_projects.search", project12),
			},
			// Match one project using multiple topics.
			{
				Config: fmt.Sprintf(`
data "gitlab_projects" "search" {
  topic = ["%s", "%s"]
}`, topic1.Name, topic2.Name),
				Check: testAccDataSourceGitlabProjectsContainsProjects("data.gitlab_projects.search", project12),
			},
			// Match no projects.
			{
				Config: fmt.Sprintf(`
data "gitlab_projects" "search" {
  topic = ["%s", "%s"]
}`, topic2.Name, topic3.Name),
				Check: testAccDataSourceGitlabProjectsContainsProjects("data.gitlab_projects.search"),
			},
		},
	})
}

// Create a test that populates the CI Restrict Pipeline value using testUtil,
// then uses a terraform `gitlab_projects` datasource to read and validate that it matches
func TestAccDataGitlabProjects_CIRestrictPipeline(t *testing.T) {
	// Requires EE
	testutil.SkipIfCE(t)

	// Create a new project using testutil, and update its pipelines cancellation
	// to "developer"
	client := testutil.TestGitlabClient
	group := testutil.CreateGroups(t, 1)[0]
	project := testutil.CreateProjectWithNamespace(t, group.ID)
	var devAccessLevel gitlab.AccessControlValue = "developer"
	_, _, err := client.Projects.EditProject(project.ID, &gitlab.EditProjectOptions{
		CIRestrictPipelineCancellationRole: &devAccessLevel,
	})
	if err != nil {
		t.Fatalf("Error updating project: %v", err)
	}

	// Create the terraform test
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					 data "gitlab_projects" "this" {
						group_id = %d
					 }
					`, group.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDataSourceGitlabProjectsContainsProjects("data.gitlab_projects.this", project),
					resource.TestCheckResourceAttr("data.gitlab_projects.this", "projects.0.ci_restrict_pipeline_cancellation_role", "developer"),
				),
			},
		},
	})
}

// Create a test that populates the CI Restrict Pipeline value using testUtil,
// then uses a terraform `gitlab_projects` datasource to read and validate that it matches
func TestAccDataGitlabProjects_CIIdTokenSubClaimComponents(t *testing.T) {
	// Create a new project using testutil, and update it's pipelines cancellation
	// to "developer"
	client := testutil.TestGitlabClient
	group := testutil.CreateGroups(t, 1)[0]
	project := testutil.CreateProjectWithNamespace(t, group.ID)
	_, _, err := client.Projects.EditProject(project.ID, &gitlab.EditProjectOptions{
		CIIdTokenSubClaimComponents: &[]string{"project_path", "ref_type"},
	})
	if err != nil {
		t.Fatalf("Error updating project: %v", err)
	}

	// Create the terraform test
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					 data "gitlab_projects" "this" {
						group_id = %d
					 }
					`, group.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDataSourceGitlabProjectsContainsProjects("data.gitlab_projects.this", project),
					resource.TestCheckResourceAttr("data.gitlab_projects.this", "projects.0.ci_id_token_sub_claim_components.0", "project_path"),
					resource.TestCheckResourceAttr("data.gitlab_projects.this", "projects.0.ci_id_token_sub_claim_components.1", "ref_type"),
					resource.TestCheckResourceAttr("data.gitlab_projects.this", "projects.0.ci_id_token_sub_claim_components.#", "2"),
				),
			},
		},
	})
}

// Create a test that populates the CI pipeline variables minimum
// override role value using testUtil, then uses a terraform
// `gitlab_projects` datasource to read and validate that it matches
func TestAccDataGitlabProjects_CIPipelineVariablesMinimumOverrideRole(t *testing.T) {
	// Create a new project using testutil, and update its pipelines cancellation
	// to "developer"
	client := testutil.TestGitlabClient
	group := testutil.CreateGroups(t, 1)[0]
	project := testutil.CreateProjectWithNamespace(t, group.ID)
	role := gitlab.CIPipelineVariablesNoOneAllowedRole
	_, _, err := client.Projects.EditProject(project.ID, &gitlab.EditProjectOptions{
		CIPipelineVariablesMinimumOverrideRole: &role,
	})
	if err != nil {
		t.Fatalf("Error updating project: %v", err)
	}

	// Create the terraform test
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					 data "gitlab_projects" "this" {
						group_id = %d
					 }
					`, group.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDataSourceGitlabProjectsContainsProjects("data.gitlab_projects.this", project),
					resource.TestCheckResourceAttr("data.gitlab_projects.this", "projects.0.ci_pipeline_variables_minimum_override_role", "no_one_allowed"),
				),
			},
		},
	})
}

// Create a test that populates the ci_delete_pipelines_in_seconds value using testUtil,
// then uses a terraform `gitlab_project` datasource to read and validate that it matches
func TestAccDataGitlabProjects_CIDeletePipelinesInSeconds(t *testing.T) {
	// Create a new project using testutil, and update its automatic pipeline cleanup setting
	// to 1 month
	client := testutil.TestGitlabClient
	group := testutil.CreateGroups(t, 1)[0]
	project := testutil.CreateProjectWithNamespace(t, group.ID)

	ciDeletePipelinesInSeconds1Month := int64(30 * 24 * 60 * 60)

	_, _, err := client.Projects.EditProject(project.ID, &gitlab.EditProjectOptions{
		CIDeletePipelinesInSeconds: &ciDeletePipelinesInSeconds1Month,
	})
	if err != nil {
		t.Fatalf("Error updating project: %v", err)
	}

	// Create the terraform test
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					 data "gitlab_projects" "this" {
						group_id = %d
					 }
					`, group.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDataSourceGitlabProjectsContainsProjects("data.gitlab_projects.this", project),
					resource.TestCheckResourceAttr("data.gitlab_projects.this", "projects.0.ci_delete_pipelines_in_seconds", strconv.FormatInt(ciDeletePipelinesInSeconds1Month, 10)),
				),
			},
		},
	})
}

func testAccDataSourceGitlabProjectsContainsProjects(dsPath string, projects ...*gitlab.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		search := s.RootModule().Resources[dsPath]
		searchResource := search.Primary.Attributes

		projectsNumber, err := strconv.ParseInt(searchResource["projects.#"], 10, 64)
		if err != nil {
			return fmt.Errorf("datasource returned no 'projects' attribute, got: %s", searchResource)
		}

		if projectsNumber != int64(len(projects)) {
			return fmt.Errorf("datasource contains unexpected number of projects, want: %d, got: %d", len(projects), projectsNumber)
		}

		for _, p := range projects {
			foundMatch := false
			for i := int64(0); i < projectsNumber; i++ {
				if searchResource[fmt.Sprintf("projects.%d.id", i)] != strconv.FormatInt(p.ID, 10) {
					continue
				}
				if searchResource[fmt.Sprintf("projects.%d.name", i)] != p.Name {
					continue
				}
				if searchResource[fmt.Sprintf("projects.%d.path", i)] != p.Path {
					continue
				}
				foundMatch = true
				break
			}
			if !foundMatch {
				return fmt.Errorf("datasource did not contain expected project, want: id=%d, got: %v", p.ID, searchResource)
			}
		}

		return nil
	}
}
