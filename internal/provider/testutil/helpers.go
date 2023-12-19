//go:build acceptance || flakey
// +build acceptance flakey

package testutil

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/onsi/gomega"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

type SkipFunc = func() (bool, error)

var testGitlabConfig = api.Config{
	Token:         os.Getenv("GITLAB_TOKEN"),
	BaseURL:       os.Getenv("GITLAB_BASE_URL"),
	CACertFile:    "",
	Insecure:      false,
	ClientCert:    "",
	ClientKey:     "",
	EarlyAuthFail: false,
}

var TestGitlabClient *gitlab.Client

func init() {
	client, err := testGitlabConfig.NewGitLabClient(context.Background())
	if err != nil {
		panic("failed to create test client: " + err.Error()) // lintignore: R009 // TODO: Resolve this tfproviderlint issue
	}
	TestGitlabClient = client

	// We are using the gomega package for its matchers only, but it requires us to register a handler anyway.
	gomega.RegisterFailHandler(func(_ string, _ ...int) {
		panic("gomega fail handler should not be used") // lintignore: R009
	})
}

// Global variable to cache the result of EE evaluation for all the tests
var isEE *bool

// Returns true if the acceptance test is running Gitlab EE.
// Meant to be used as SkipFunc to skip tests that work only on Gitlab CE.
func IsRunningInEE() (bool, error) {
	if isEE != nil {
		return *isEE, nil
	}
	eeContext, err := utils.IsRunningInEEContext(TestGitlabClient)
	if err != nil {
		return false, err
	}
	isEE := gitlab.Ptr(eeContext)
	return *isEE, err
}

// IsRunningInCE returns true if the acceptance test is running Gitlab CE.
// Meant to be used as SkipFunc to skip tests that work only on Gitlab EE.
func IsRunningInCE() (bool, error) {
	isEE, err := IsRunningInEE()
	return !isEE, err
}

// SkipIfCE is a test helper that skips the current test if the GitLab version is not GitLab Enterprise.
// This is useful when the version needs to be checked during setup, before the Terraform acceptance test starts.
func SkipIfCE(t *testing.T) {
	t.Helper()

	isCE, err := IsRunningInCE()
	if err != nil {
		t.Fatalf("could not check GitLab version is CE: %v", err)
	}
	if isCE {
		t.Skipf("Test is skipped for CE (non-Enterprise) version of GitLab")
	}
}

func SkipIfEE(t *testing.T) {
	t.Helper()

	isEE, err := IsRunningInEE()
	if err != nil {
		t.Fatalf("could not check GitLab version is EE: %v", err)
	}
	if isEE {
		t.Skipf("Test is skipped for EE (Enterprise) version of GitLab")
	}
}

func RunIfLessThan(t *testing.T, requiredMaxVersion string) {
	isLessThan, err := api.IsGitLabVersionLessThan(context.TODO(), TestGitlabClient, requiredMaxVersion)()
	if err != nil {
		t.Fatalf("Failed to fetch GitLab version: %+v", err)
	}

	if !isLessThan {
		t.Skipf("This test is only valid for GitLab versions less than %s", requiredMaxVersion)
	}
}

func RunIfAtLeast(t *testing.T, requiredMinVersion string) {
	isAtLeast, err := api.IsGitLabVersionAtLeast(context.TODO(), TestGitlabClient, requiredMinVersion)()
	if err != nil {
		t.Fatalf("Failed to fetch GitLab version: %+v", err)
	}

	if !isAtLeast {
		t.Skipf("This test is only valid for GitLab versions newer than %s", requiredMinVersion)
	}
}

func IsRunningAtLeast(t *testing.T, requiredMinVersion string) bool {
	isAtLeast, err := api.IsGitLabVersionAtLeast(context.TODO(), TestGitlabClient, requiredMinVersion)()
	if err != nil {
		t.Fatalf("Failed to fetch GitLab version: %+v", err)
	}

	return isAtLeast
}

// GetCurrentUser is a test helper for getting the current user of the provided client.
func GetCurrentUser(t *testing.T) *gitlab.User {
	t.Helper()

	user, _, err := TestGitlabClient.Users.CurrentUser()
	if err != nil {
		t.Fatalf("could not get current user: %v", err)
	}

	return user
}

// CreateProject is a test helper for creating a project.
func CreateProject(t *testing.T) *gitlab.Project {
	return CreateProjectWithNamespace(t, 0)
}

// CreateProjectWithNamespace is a test helper for creating a project. This method accepts a namespace to great a project
// within a group
func CreateProjectWithNamespace(t *testing.T, namespaceID int) *gitlab.Project {
	t.Helper()

	options := &gitlab.CreateProjectOptions{
		Name:        gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
		Description: gitlab.Ptr("Terraform acceptance tests"),
		// So that acceptance tests can be run in a gitlab organization with no billing.
		Visibility: gitlab.Ptr(gitlab.PublicVisibility),
		// So that a branch is created.
		InitializeWithReadme: gitlab.Ptr(true),
	}

	//Apply a namespace if one is passed in.
	if namespaceID != 0 {
		options.NamespaceID = gitlab.Ptr(namespaceID)
	}

	return CreateProjectWithOptions(t, options)
}

// CreateProjectWithOptions is a test helper for creating a project given some options
func CreateProjectWithOptions(t *testing.T, opts *gitlab.CreateProjectOptions) *gitlab.Project {
	t.Helper()

	project, _, err := TestGitlabClient.Projects.CreateProject(opts)
	if err != nil {
		t.Fatalf("could not create test project: %v", err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.Projects.DeleteProject(project.ID); err != nil {
			t.Fatalf("could not cleanup test project: %v", err)
		}
	})

	return project
}

// CreateTopic is a test helper for creating a topic.
func CreateTopic(t *testing.T) *gitlab.Topic {
	t.Helper()

	name := gitlab.Ptr(acctest.RandomWithPrefix("acctest"))
	options := &gitlab.CreateTopicOptions{
		Name:  name,
		Title: name,
	}

	topic, _, err := TestGitlabClient.Topics.CreateTopic(options)
	if err != nil {
		t.Fatalf("could not create test topic: %v", err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.Topics.DeleteTopic(topic.ID); err != nil {
			t.Fatalf("could not cleanup test topic: %v", err)
		}
	})

	return topic
}

// CreateUsers is a test helper for creating a specified number of users.
func CreateUsers(t *testing.T, n int) []*gitlab.User {
	return CreateUsersWithPrefix(t, n, "acctest-user")
}

func CreateUsersWithPrefix(t *testing.T, n int, prefix string) []*gitlab.User {
	t.Helper()

	users := make([]*gitlab.User, n)

	for i := range users {
		var err error
		username := acctest.RandomWithPrefix(prefix)
		users[i], _, err = TestGitlabClient.Users.CreateUser(&gitlab.CreateUserOptions{
			Name:             gitlab.Ptr(username),
			Username:         gitlab.Ptr(username),
			Email:            gitlab.Ptr(username + "@example.com"),
			Password:         gitlab.Ptr(acctest.RandString(16)),
			SkipConfirmation: gitlab.Ptr(true),
		})
		if err != nil {
			t.Fatalf("could not create test user (username=%q): %v", username, err)
		}

		userID := users[i].ID // Needed for closure.
		t.Cleanup(func() {
			if _, err := TestGitlabClient.Users.DeleteUser(userID); err != nil {
				t.Fatalf("could not cleanup test user: %v", err)
			}
		})
	}

	return users
}

// Create Personal Access Token for a given user.
func CreatePersonalAccessToken(t *testing.T, user *gitlab.User) *gitlab.PersonalAccessToken {
	t.Helper()

	token, _, err := TestGitlabClient.Users.CreatePersonalAccessToken(user.ID, &gitlab.CreatePersonalAccessTokenOptions{Name: gitlab.Ptr(acctest.RandomWithPrefix("acctest")), Scopes: &[]string{"api"}})
	if err != nil {
		t.Fatalf("could not create Personal Access Token for user %d", user.ID)
	}

	return token
}

// CreateGroups is a test helper for creating a specified number of groups.
func CreateGroups(t *testing.T, n int) []*gitlab.Group {
	t.Helper()

	return CreateGroupsWithPrefix(t, n, "acctest-group")
}

// CreateGroupsWithPrefix is a test helper for creating a specified number of groups with specific prefix.
func CreateGroupsWithPrefix(t *testing.T, n int, prefix string) []*gitlab.Group {
	t.Helper()

	groups := make([]*gitlab.Group, n)

	for i := range groups {
		var err error
		name := acctest.RandomWithPrefix(prefix)
		groups[i], _, err = TestGitlabClient.Groups.CreateGroup(&gitlab.CreateGroupOptions{
			Name: gitlab.Ptr(name),
			Path: gitlab.Ptr(name),
			// So that acceptance tests can be run in a gitlab organization with no billing.
			Visibility: gitlab.Ptr(gitlab.PublicVisibility),
		})
		if err != nil {
			t.Fatalf("could not create test group: %v", err)
		}

		groupID := groups[i].ID // Needed for closure.
		t.Cleanup(func() {
			if _, err := TestGitlabClient.Groups.DeleteGroup(groupID); err != nil {
				t.Fatalf("could not cleanup test group: %v", err)
			}
		})
	}

	return groups
}

// CreateSubGroupsWithPrefix is a test helper for creating a specified number of subgroups with specific prefix.
func CreateSubGroupsWithPrefix(t *testing.T, parentGroup *gitlab.Group, n int, prefix string) []*gitlab.Group {
	t.Helper()

	groups := make([]*gitlab.Group, n)

	for i := range groups {
		var err error
		name := acctest.RandomWithPrefix(prefix)
		groups[i], _, err = TestGitlabClient.Groups.CreateGroup(&gitlab.CreateGroupOptions{
			Name: gitlab.Ptr(name),
			Path: gitlab.Ptr(name),
			// So that acceptance tests can be run in a gitlab organization with no billing.
			Visibility: gitlab.Ptr(gitlab.PublicVisibility),
			ParentID:   gitlab.Ptr(parentGroup.ID),
		})
		if err != nil {
			t.Fatalf("could not create test subgroup: %v", err)
		}
	}

	return groups
}

// CreateSubGroups is a test helper for creating a specified number of subgroups.
func CreateSubGroups(t *testing.T, parentGroup *gitlab.Group, n int) []*gitlab.Group {
	t.Helper()

	return CreateSubGroupsWithPrefix(t, parentGroup, n, "acctest-group")
}

func CreateGroupHooks(t *testing.T, gid interface{}, n int) []*gitlab.GroupHook {
	t.Helper()

	var hooks []*gitlab.GroupHook
	for i := 0; i < n; i++ {
		hook, _, err := TestGitlabClient.Groups.AddGroupHook(gid, &gitlab.AddGroupHookOptions{
			URL: gitlab.Ptr(fmt.Sprintf("https://%s.com", acctest.RandomWithPrefix("acctest"))),
		})
		if err != nil {
			t.Fatalf("could not create group hook: %v", err)
		}
		hooks = append(hooks, hook)
	}
	return hooks
}

// CreateBranches is a test helper for creating a specified number of branches.
// It assumes the project will be destroyed at the end of the test and will not cleanup created branches.
func CreateBranches(t *testing.T, project *gitlab.Project, n int) []*gitlab.Branch {
	t.Helper()

	branches := make([]*gitlab.Branch, n)

	for i := range branches {
		var err error
		branches[i], _, err = TestGitlabClient.Branches.CreateBranch(project.ID, &gitlab.CreateBranchOptions{
			Branch: gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
			Ref:    gitlab.Ptr(project.DefaultBranch),
		})
		if err != nil {
			t.Fatalf("could not create test branches: %v", err)
		}
	}

	return branches
}

// CreateProtectedBranches is a test helper for creating a specified number of protected branches.
// It assumes the project will be destroyed at the end of the test and will not cleanup created branches.
func CreateProtectedBranches(t *testing.T, project *gitlab.Project, n int) []*gitlab.ProtectedBranch {
	t.Helper()

	branches := CreateBranches(t, project, n)
	protectedBranches := make([]*gitlab.ProtectedBranch, n)

	for i := range make([]int, n) {
		var err error
		protectedBranches[i], _, err = TestGitlabClient.ProtectedBranches.ProtectRepositoryBranches(project.ID, &gitlab.ProtectRepositoryBranchesOptions{
			Name: gitlab.Ptr(branches[i].Name),
		})
		if err != nil {
			t.Fatalf("could not protect test branches: %v", err)
		}
	}

	return protectedBranches
}

// CreateReleases is a test helper for creating a specified number of releases.
// It assumes the project will be destroyed at the end of the test and will not cleanup created releases.
func CreateReleases(t *testing.T, project *gitlab.Project, n int) []*gitlab.Release {
	t.Helper()

	releases := make([]*gitlab.Release, n)
	linkType := gitlab.LinkTypeValue("other")
	linkURL1 := fmt.Sprintf("https://test/%v", *gitlab.Ptr(acctest.RandomWithPrefix("acctest")))
	linkURL2 := fmt.Sprintf("https://test/%v", *gitlab.Ptr(acctest.RandomWithPrefix("acctest")))

	for i := range releases {
		var err error
		releases[i], _, err = TestGitlabClient.Releases.CreateRelease(project.ID, &gitlab.CreateReleaseOptions{
			Name:    gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
			TagName: gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
			Ref:     &project.DefaultBranch,
			Assets: &gitlab.ReleaseAssetsOptions{
				Links: []*gitlab.ReleaseAssetLinkOptions{
					{
						Name:     gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
						URL:      &linkURL1,
						LinkType: &linkType,
					},
					{
						Name:     gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
						URL:      &linkURL2,
						LinkType: &linkType,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("could not create test releases: %v", err)
		}
	}

	return releases
}

// AddProjectMembers is a test helper for adding users as members of a project with Developer access level.
// It assumes the project will be destroyed at the end of the test and will not cleanup members.
func AddProjectMembers(t *testing.T, pid interface{}, users []*gitlab.User) {
	t.Helper()

	AddProjectMembersWithAccessLevel(t, pid, users, gitlab.DeveloperPermissions)
}

// AddProjectMembersWithAccessLevel is a test helper for adding users as members of a project with a given access level.
// It assumes the project will be destroyed at the end of the test and will not cleanup members.
func AddProjectMembersWithAccessLevel(t *testing.T, pid interface{}, users []*gitlab.User, accessLevel gitlab.AccessLevelValue) {
	t.Helper()

	for _, user := range users {
		_, _, err := TestGitlabClient.ProjectMembers.AddProjectMember(pid, &gitlab.AddProjectMemberOptions{
			UserID:      user.ID,
			AccessLevel: gitlab.Ptr(accessLevel),
		})
		if err != nil {
			t.Fatalf("could not add test project member: %v", err)
		}
	}
}

func CreateProjectHooks(t *testing.T, pid interface{}, n int) []*gitlab.ProjectHook {
	t.Helper()

	var hooks []*gitlab.ProjectHook
	for i := 0; i < n; i++ {
		hook, _, err := TestGitlabClient.Projects.AddProjectHook(pid, &gitlab.AddProjectHookOptions{
			URL: gitlab.Ptr(fmt.Sprintf("https://%s.com", acctest.RandomWithPrefix("acctest"))),
		})
		if err != nil {
			t.Fatalf("could not create project hook: %v", err)
		}
		hooks = append(hooks, hook)
	}
	return hooks
}

func CreateClusterAgents(t *testing.T, pid interface{}, n int) []*gitlab.Agent {
	t.Helper()

	var clusterAgents []*gitlab.Agent
	for i := 0; i < n; i++ {
		clusterAgent, _, err := TestGitlabClient.ClusterAgents.RegisterAgent(pid, &gitlab.RegisterAgentOptions{
			Name: gitlab.Ptr(fmt.Sprintf("agent-%d", i)),
		})
		if err != nil {
			t.Fatalf("could not create test cluster agent: %v", err)
		}
		t.Cleanup(func() {
			_, err := TestGitlabClient.ClusterAgents.DeleteAgent(pid, clusterAgent.ID)
			if err != nil {
				t.Fatalf("could not cleanup test cluster agent: %v", err)
			}
		})
		clusterAgents = append(clusterAgents, clusterAgent)
	}
	return clusterAgents
}

func CreateProjectIssues(t *testing.T, pid interface{}, n int) []*gitlab.Issue {
	t.Helper()

	dueDate := gitlab.ISOTime(time.Now().Add(time.Hour))
	var issues []*gitlab.Issue
	for i := 0; i < n; i++ {
		issue, _, err := TestGitlabClient.Issues.CreateIssue(pid, &gitlab.CreateIssueOptions{
			Title:       gitlab.Ptr(fmt.Sprintf("Issue %d", i)),
			Description: gitlab.Ptr(fmt.Sprintf("Description %d", i)),
			DueDate:     &dueDate,
		})
		if err != nil {
			t.Fatalf("could not create test issue: %v", err)
		}
		issues = append(issues, issue)
	}
	return issues
}

func CreateGroupEpicBoard(t *testing.T, path string) {
	t.Helper()
	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				epicBoardCreate(
					input: {
						groupPath: "%s",
						name: "%s"
					}
				) {
					epicBoard {
						id,
						name
					}
					errors
				}
			}`, path, acctest.RandomWithPrefix("acctest")),
	}

	ctx := context.Background()
	var pid interface{}
	if _, err := api.SendGraphQLRequest(ctx, TestGitlabClient, query, &pid); err != nil {
		t.Fatalf(fmt.Sprintf("Unable to create epic board: %s", err.Error()))
	}
}

func CreateGroupIssueBoard(t *testing.T, pid interface{}) *gitlab.GroupIssueBoard {
	t.Helper()

	issueBoard, _, err := TestGitlabClient.GroupIssueBoards.CreateGroupIssueBoard(pid, &gitlab.CreateGroupIssueBoardOptions{Name: gitlab.Ptr(acctest.RandomWithPrefix("acctest"))})
	if err != nil {
		t.Fatalf("could not create test group issue board: %v", err)
	}

	return issueBoard
}

func CreateProjectIssueBoard(t *testing.T, pid interface{}) *gitlab.IssueBoard {
	t.Helper()

	issueBoard, _, err := TestGitlabClient.Boards.CreateIssueBoard(pid, &gitlab.CreateIssueBoardOptions{Name: gitlab.Ptr(acctest.RandomWithPrefix("acctest"))})
	if err != nil {
		t.Fatalf("could not create test issue board: %v", err)
	}

	return issueBoard
}

func CreateGroupLabels(t *testing.T, pid interface{}, n int) []*gitlab.GroupLabel {
	t.Helper()

	var labels []*gitlab.GroupLabel
	for i := 0; i < n; i++ {
		label, _, err := TestGitlabClient.GroupLabels.CreateGroupLabel(pid, &gitlab.CreateGroupLabelOptions{Name: gitlab.Ptr(acctest.RandomWithPrefix("acctest")), Color: gitlab.Ptr("#000000")})
		if err != nil {
			t.Fatalf("could not create test group label: %v", err)
		}
		labels = append(labels, label)
	}

	return labels
}

func CreateProjectLabels(t *testing.T, pid interface{}, n int) []*gitlab.Label {
	t.Helper()

	var labels []*gitlab.Label
	for i := 0; i < n; i++ {
		label, _, err := TestGitlabClient.Labels.CreateLabel(pid, &gitlab.CreateLabelOptions{Name: gitlab.Ptr(acctest.RandomWithPrefix("acctest")), Color: gitlab.Ptr("#000000")})
		if err != nil {
			t.Fatalf("could not create test label: %v", err)
		}
		labels = append(labels, label)
	}

	return labels
}

// AddGroupMembers is a test helper for adding users as members of a group with Developer level access.
// It assumes the group will be destroyed at the end of the test and will not cleanup members.
func AddGroupMembers(t *testing.T, gid interface{}, users []*gitlab.User) {
	t.Helper()

	AddGroupMembersWithAccessLevel(t, gid, users, gitlab.DeveloperPermissions)
}

// AddGroupMembersWithAccessLevel is a test helper for adding users as members of a group with a given access level.
func AddGroupMembersWithAccessLevel(t *testing.T, gid interface{}, users []*gitlab.User, accessLevel gitlab.AccessLevelValue) {
	t.Helper()

	for _, user := range users {
		_, _, err := TestGitlabClient.GroupMembers.AddGroupMember(gid, &gitlab.AddGroupMemberOptions{
			UserID:      gitlab.Ptr(user.ID),
			AccessLevel: gitlab.Ptr(accessLevel),
		})
		if err != nil {
			t.Fatalf("could not add test group member: %v", err)
		}
	}
}

// ProjectShareGroup is a test helper for sharing a project with a group.
func ProjectShareGroup(t *testing.T, pid interface{}, gid int) {
	t.Helper()

	_, err := TestGitlabClient.Projects.ShareProjectWithGroup(pid, &gitlab.ShareWithGroupOptions{
		GroupID:     gitlab.Ptr(gid),
		GroupAccess: gitlab.Ptr(gitlab.DeveloperPermissions),
	})
	if err != nil {
		t.Fatalf("could not share project %v with group %d: %v", pid, gid, err)
	}
}

// List project members
func ListProjectMembers(t *testing.T, pid interface{}) {
	t.Helper()

	members, _, err := TestGitlabClient.ProjectMembers.ListAllProjectMembers(pid, &gitlab.ListProjectMembersOptions{})
	if err != nil {
		t.Fatalf("could not get project %d member list: %v", pid, err)
	}

	t.Log("--------------------------------------------------------")
	t.Log("Project member list")
	for _, member := range members {
		t.Logf("\nUserId: `%d`, accessLevel: `%d`, state: `%s`", member.ID, member.AccessLevel, member.State)
	}
	t.Log("\n--------------------------------------------------------")
}

// AddProjectMilestones is a test helper for adding milestones to project.
// It assumes the group will be destroyed at the end of the test and will not cleanup milestones.
func AddProjectMilestones(t *testing.T, project *gitlab.Project, n int) []*gitlab.Milestone {
	t.Helper()

	milestones := make([]*gitlab.Milestone, n)

	for i := range milestones {
		var err error
		milestones[i], _, err = TestGitlabClient.Milestones.CreateMilestone(project.ID, &gitlab.CreateMilestoneOptions{
			Title:       gitlab.Ptr(fmt.Sprintf("Milestone %d", i)),
			Description: gitlab.Ptr(fmt.Sprintf("Description %d", i)),
		})
		if err != nil {
			t.Fatalf("Could not create test milestones: %v", err)
		}
	}

	return milestones
}

func AddGroupMilestones(t *testing.T, group *gitlab.Group, n int) []*gitlab.GroupMilestone {
	t.Helper()

	milestones := make([]*gitlab.GroupMilestone, n)

	for i := range milestones {
		var err error
		startDate := time.Date(2023, 8, 1, 0, 0, 0, 0, time.UTC)
		x := gitlab.ISOTime(startDate)
		endDate := time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)
		y := gitlab.ISOTime(endDate)
		milestones[i], _, err = TestGitlabClient.GroupMilestones.CreateGroupMilestone(group.ID, &gitlab.CreateGroupMilestoneOptions{
			Title:       gitlab.Ptr(fmt.Sprintf("Milestone %d", i)),
			Description: gitlab.Ptr(fmt.Sprintf("Description %d", i)),
			StartDate:   &x,
			DueDate:     &y,
		})
		if err != nil {
			t.Fatalf("Could not create test milestones: %v", err)
		}
	}

	return milestones
}

func CreateDeployKey(t *testing.T, projectID int, options *gitlab.AddDeployKeyOptions) *gitlab.ProjectDeployKey {
	deployKey, _, err := TestGitlabClient.DeployKeys.AddDeployKey(projectID, options)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.DeployKeys.DeleteDeployKey(projectID, deployKey.ID); err != nil {
			t.Fatal(err)
		}
	})

	return deployKey
}

// CreateProjectEnvironment is a test helper function for creating a project environment
func CreateProjectEnvironment(t *testing.T, projectID int, options *gitlab.CreateEnvironmentOptions) *gitlab.Environment {
	t.Helper()

	projectEnvironment, _, err := TestGitlabClient.Environments.CreateEnvironment(projectID, options)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if projectEnvironment.State != "stopped" {
			_, _, err = TestGitlabClient.Environments.StopEnvironment(projectID, projectEnvironment.ID, nil)
			if err != nil {
				t.Fatal(err)
			}
		}
		if _, err := TestGitlabClient.Environments.DeleteEnvironment(projectID, projectEnvironment.ID); err != nil {
			t.Fatal(err)
		}
	})

	return projectEnvironment
}

func CreateProjectVariable(t *testing.T, projectID int) *gitlab.ProjectVariable {
	variable, _, err := TestGitlabClient.ProjectVariables.CreateVariable(projectID, &gitlab.CreateProjectVariableOptions{
		Key:   gitlab.Ptr(fmt.Sprintf("test_key_%d", acctest.RandInt())),
		Value: gitlab.Ptr("test_value"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.ProjectVariables.RemoveVariable(projectID, variable.Key, nil); err != nil {
			t.Fatal(err)
		}
	})

	return variable
}

func CreateGroupVariable(t *testing.T, groupID int) *gitlab.GroupVariable {
	variable, _, err := TestGitlabClient.GroupVariables.CreateVariable(groupID, &gitlab.CreateGroupVariableOptions{
		Key:   gitlab.Ptr(fmt.Sprintf("test_key_%d", acctest.RandInt())),
		Value: gitlab.Ptr("test_value"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.GroupVariables.RemoveVariable(groupID, variable.Key, nil); err != nil {
			t.Fatal(err)
		}
	})

	return variable
}

func CreateInstanceVariable(t *testing.T) *gitlab.InstanceVariable {
	variable, _, err := TestGitlabClient.InstanceVariables.CreateVariable(&gitlab.CreateInstanceVariableOptions{
		Key:   gitlab.Ptr(fmt.Sprintf("test_key_%d", acctest.RandInt())),
		Value: gitlab.Ptr("test_value"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.InstanceVariables.RemoveVariable(variable.Key, nil); err != nil {
			t.Fatal(err)
		}
	})

	return variable
}

func CreateProjectFile(t *testing.T, projectID int, fileContent string, filePath string, branch string) *gitlab.FileInfo {

	file, _, err := TestGitlabClient.RepositoryFiles.CreateFile(projectID, filePath, &gitlab.CreateFileOptions{
		Branch:        &branch,
		Encoding:      gitlab.Ptr("base64"),
		Content:       &fileContent,
		CommitMessage: gitlab.Ptr(fmt.Sprintf("Random_Commit_Message_%d", acctest.RandInt())),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.RepositoryFiles.DeleteFile(projectID, filePath, &gitlab.DeleteFileOptions{
			Branch:        &branch,
			CommitMessage: gitlab.Ptr(fmt.Sprintf("Delete_Random_Commit_Message_%d", acctest.RandInt())),
		}); err != nil {
			t.Fatal(err)
		}
	})

	return file
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func CreateComplianceFramework(t *testing.T, group *gitlab.Group) *api.GraphQLComplianceFramework {
	t.Helper()

	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				createComplianceFramework(
					input: {
						params: {
							name: "Compliance Framework %d",
							description: "Test Compliance Framework",
							color: "#042",
							default: false
						},
						namespacePath: "%s"
					}
				) {
					framework {
						id,
						name,
						description,
						color,
						default,
						pipelineConfigurationFullPath
					}
					errors
				}
			}`, acctest.RandInt(), group.FullPath),
	}

	type createComplianceFrameworkResponse struct {
		Data struct {
			CreateComplianceFramework struct {
				Framework api.GraphQLComplianceFramework `json:"framework"`
			} `json:"createComplianceFramework"`
		} `json:"data"`
	}

	var response createComplianceFrameworkResponse
	if _, err := api.SendGraphQLRequest(context.Background(), TestGitlabClient, query, &response); err != nil {
		t.Fatalf("Unable to create compliance framework: %s", err.Error())
	}

	t.Cleanup(func() {
		query := api.GraphQLQuery{
			Query: fmt.Sprintf(`
				mutation {
					destroyComplianceFramework(
						input: {
							id: "%s"
						}
					) {
						errors
					}
				}`, response.Data.CreateComplianceFramework.Framework.ID),
		}

		if _, err := api.SendGraphQLRequest(context.Background(), TestGitlabClient, query, nil); err != nil {
			t.Fatalf("Unable to delete compliance framework: %s", err.Error())
		}
	})

	return &response.Data.CreateComplianceFramework.Framework
}
