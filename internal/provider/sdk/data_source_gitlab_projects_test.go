//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	gitlab "github.com/xanzy/go-gitlab"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjects_search(t *testing.T) {
	projectName := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: testAccDataGitlabProjectsConfigGetProjectSearch(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDataSourceGitlabProjects(
						"gitlab_project.search",
						"data.gitlab_projects.search",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_projects.search",
						"projects.0.owner.0.id",
						"1",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_projects.search",
						"projects.0.permissions.0.project_access.access_level",
						"50",
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_projects.search",
						"projects.0.permissions.0.project_access.group_level",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_projects.search",
						"projects.0.namespace.0.kind",
						"user",
					),
				),
			},
		},
	})
}

func TestAccDataGitlabProjects_groups(t *testing.T) {
	projectName := fmt.Sprintf("tf-%s", acctest.RandString(5))
	groupName := fmt.Sprintf("tf-%s", acctest.RandString(5))
	parentGroupName := fmt.Sprintf("tf-%s", acctest.RandString(5))
	subGroupName1 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	subGroupName2 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	subGroupProjectName1 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	subGroupProjectName2 := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: testAccDataGitlabProjectsConfigGetGroupProjectsByGroupId(groupName, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDataSourceGitlabProjects(
						"gitlab_project.testProject",
						"data.gitlab_projects.group",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_projects.group",
						"projects.0.namespace.0.kind",
						"group",
					),
				),
			},
			{
				Config: testAccDataGitlabProjectsConfigGetNestedProjectsByParentGroupId(parentGroupName, subGroupName1, subGroupName2, subGroupProjectName1, subGroupProjectName2),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabProjects(
						"gitlab_project.testProject1",
						"data.gitlab_projects.subGroups",
					),
					testAccDataSourceGitlabProjects(
						"gitlab_project.testProject2",
						"data.gitlab_projects.subGroups",
					),
				),
			},
		},
	})
}

func TestAccDataGitlabProjects_searchArchivedRepository(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: testAccDataGitlabProjectsConfigGetProjectArchivedRepositoryAll(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_projects.search",
						"projects.0.name",
						fmt.Sprintf("archived-%d", rInt),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_projects.search",
						"projects.1.name",
						fmt.Sprintf("not-archived-%d", rInt),
					),
				),
			},
			{
				Config: testAccDataGitlabProjectsConfigGetProjectArchivedRepository(rInt, "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_projects.search",
						"projects.0.name",
						fmt.Sprintf("archived-%d", rInt),
					),
				),
			},
			{
				Config: testAccDataGitlabProjectsConfigGetProjectArchivedRepository(rInt, "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_projects.search",
						"projects.0.name",
						fmt.Sprintf("not-archived-%d", rInt),
					),
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

	// Requires GitLab 16.8 or greater
	testutil.RunIfAtLeast(t, "16.8")

	// Create a new project using testutil, and update it's pipelines cancellation
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
					resource.TestCheckResourceAttr("data.gitlab_projects.this", "projects.0.ci_restrict_pipeline_cancellation_role", "developer"),
				),
			},
		},
	})
}

// Create a test that populates the CI pipeline variables minimum
// override role value using testUtil, then uses a terraform
// `gitlab_projects` datasource to read and validate that it matches
func TestAccDataGitlabProjects_CIPipelineVariablesMinimumOverrideRole(t *testing.T) {
	// Requires GitLab 17.1 or greater
	testutil.RunIfAtLeast(t, "17.1")

	// Create a new project using testutil, and update it's pipelines cancellation
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
					resource.TestCheckResourceAttr("data.gitlab_projects.this", "projects.0.ci_pipeline_variables_minimum_override_role", "no_one_allowed"),
				),
			},
		},
	})
}

func testAccDataSourceGitlabProjectsContainsProjects(dsPath string, projects ...*gitlab.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		search := s.RootModule().Resources[dsPath]
		searchResource := search.Primary.Attributes

		projectsNumber, err := strconv.Atoi(searchResource["projects.#"])
		if err != nil {
			return fmt.Errorf("datasource returned no 'projects' attribute, got: %s", searchResource)
		}

		if projectsNumber != len(projects) {
			return fmt.Errorf("datasource contains unexpected number of projects, want: %d, got: %d", len(projects), projectsNumber)
		}

		for _, p := range projects {
			foundMatch := false
			for i := 0; i < projectsNumber; i++ {
				if searchResource[fmt.Sprintf("projects.%d.id", i)] != strconv.Itoa(p.ID) {
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

func testAccDataSourceGitlabProjects(src string, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		project := s.RootModule().Resources[src]
		projectResource := project.Primary.Attributes

		search := s.RootModule().Resources[n]
		searchResource := search.Primary.Attributes

		if searchResource["id"] == "" {
			return fmt.Errorf("expected to get a project ID from Gitlab")
		}
		if searchResource["projects.#"] == "0" {
			return fmt.Errorf("expected to find at least one matching project from the datasource")
		}

		projectsNumber, err := strconv.Atoi(searchResource["projects.#"])
		if err != nil {
			return fmt.Errorf("the datasource returned no 'projects' attribute, got: %s", searchResource)
		}

		testAttributes := []string{
			"id",
			"name",
			"path",
		}

		for i := 0; i < projectsNumber; i++ {
			for j, attribute := range testAttributes {
				if searchResource[fmt.Sprintf("projects.%d.%s", i, attribute)] != projectResource[attribute] {
					break
				}
				if j == len(testAttributes)-1 {
					// Found an exact match
					return nil
				}
			}
		}

		var errorMessageExpected strings.Builder
		for _, attr := range testAttributes {
			errorMessageExpected.WriteString(fmt.Sprintf("%s=%v, ", attr, projectResource[fmt.Sprintf("%s", attr)])) // nolint // TODO: Resolve this golangci-lint issue: S1025: the argument is already a string, there's no need to use fmt.Sprintf (gosimple)
		}

		var errorMessageGot strings.Builder
		for i := 0; i < projectsNumber; i++ {
			errorMessageGot.WriteString(fmt.Sprintf("project_%d: ", i))
			for _, attr := range testAttributes {
				errorMessageGot.WriteString(fmt.Sprintf("%s_%d=%v, ", attr, i, searchResource[fmt.Sprintf("projects.%d.%s", i, attr)]))
			}
			errorMessageGot.WriteString("\n")
		}

		return fmt.Errorf("datasource did not return any match.\nExpected: %s\nGot:\n  %s", errorMessageExpected.String(), errorMessageGot.String())
	}
}

func testAccDataGitlabProjectsConfigGetProjectSearch(projectName string) string {
	return fmt.Sprintf(`

resource "gitlab_project" "search" {
  name = "%s"
  path = "%s"
}

data "gitlab_projects" "search" {
  search = gitlab_project.search.name
}
	`, projectName, projectName)
}

func testAccDataGitlabProjectsConfigGetProjectArchivedRepositoryAll(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "test" {
	name = "test-%d"
	path = "test-%d"
}

resource "gitlab_project" "archived_repo" {
  name         = "archived-%d"
  namespace_id = gitlab_group.test.id
  archived     = true
}

resource "gitlab_project" "not_archived_repo" {
  name         = "not-archived-%d"
  namespace_id = gitlab_group.test.id
  archived     = false
}

data "gitlab_projects" "search" {
  group_id = gitlab_group.test.id
  // NOTE: is required to have deterministic results
  order_by = "name"
  sort     = "asc"

  depends_on = [gitlab_project.archived_repo, gitlab_project.not_archived_repo]
}
	`, rInt, rInt, rInt, rInt)
}

func testAccDataGitlabProjectsConfigGetProjectArchivedRepository(rInt int, archived string) string {
	return fmt.Sprintf(`
resource "gitlab_group" "test" {
	name = "test-%d"
	path = "test-%d"
}

resource "gitlab_project" "archived_repo" {
  name         = "archived-%d"
  namespace_id = gitlab_group.test.id
  archived     = true
}

resource "gitlab_project" "not_archived_repo" {
  name         = "not-archived-%d"
  namespace_id = gitlab_group.test.id
  archived     = false
}

data "gitlab_projects" "search" {
  group_id = gitlab_group.test.id
  // NOTE: is required to have deterministic results
  order_by = "name"
  sort     = "asc"

  archived = %s

  depends_on = [gitlab_project.archived_repo, gitlab_project.not_archived_repo]
}
	`, rInt, rInt, rInt, rInt, archived)
}

func testAccDataGitlabProjectsConfigGetGroupProjectsByGroupId(groupName string, projectName string) string {
	return fmt.Sprintf(`
resource "gitlab_group" "testGroup" {
  name = "%s"
  path = "%s"
  description = "Terraform acceptance tests"
}

resource "gitlab_project" "testProject"{
  name = "%s"
  namespace_id = gitlab_group.testGroup.id
}

data "gitlab_projects" "group" {
  group_id = gitlab_project.testProject.namespace_id
}
	`, groupName, groupName, projectName)
}

func testAccDataGitlabProjectsConfigGetNestedProjectsByParentGroupId(parentGroupName string, subGroupName1 string, subGroupName2 string, projectName1 string, projectName2 string) string {
	return fmt.Sprintf(`
resource "gitlab_group" "testGroup" {
  name = "%s"
  path = "%s"
}

resource "gitlab_group" "testSubGroup1" {
  name = "%s"
  path = "%s"
  parent_id = gitlab_group.testGroup.id
}

resource "gitlab_group" "testSubGroup2" {
  name = "%s"
  path = "%s"
  parent_id = gitlab_group.testGroup.id
}

resource "gitlab_project" "testProject1"{
  name = "%s"
  namespace_id = gitlab_group.testSubGroup1.id
  description = gitlab_group.testGroup.id
}

resource "gitlab_project" "testProject2"{
  name = "%s"
  namespace_id = gitlab_group.testSubGroup2.id
  // This is all just to avoid using explicit depends_on on the datasource
  // since it seems to break the acceptance tests
  description = gitlab_project.testProject1.description
}

data "gitlab_projects" "subGroups" {
  // This is to ensure the projects have been created before running the datasource
  group_id = gitlab_project.testProject2.description
  include_subgroups = true
}
	`, parentGroupName, parentGroupName, subGroupName1, subGroupName1, subGroupName2, subGroupName2, projectName1, projectName2)
}
