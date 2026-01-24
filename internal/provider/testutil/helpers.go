//go:build acceptance || flakey || settings || saas

package testutil

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/onsi/gomega"
	gitlab "gitlab.com/gitlab-org/api/client-go"

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
	Headers:       nil,
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

func SkipIfSaaS(t *testing.T) {
	t.Helper()

	if IsRunningOnSaaS(t) {
		t.Skipf("Test is skipped for SaaS version of GitLab")
	}
}

func RunIfLessThan(t *testing.T, requiredMaxVersion string) {
	isLessThan, err := api.IsGitLabVersionLessThan(context.Background(), TestGitlabClient, requiredMaxVersion)()
	if err != nil {
		t.Fatalf("Failed to fetch GitLab version: %+v", err)
	}

	if !isLessThan {
		t.Skipf("This test is only valid for GitLab versions less than %s", requiredMaxVersion)
	}
}

func RunIfAtLeast(t *testing.T, requiredMinVersion string) {
	isAtLeast, err := api.IsGitLabVersionAtLeast(context.Background(), TestGitlabClient, requiredMinVersion)()
	if err != nil {
		t.Fatalf("Failed to fetch GitLab version: %+v", err)
	}

	if !isAtLeast {
		t.Skipf("This test is only valid for GitLab versions newer than %s", requiredMinVersion)
	}
}

func IsRunningAtLeast(t *testing.T, requiredMinVersion string) bool {
	isAtLeast, err := api.IsGitLabVersionAtLeast(context.Background(), TestGitlabClient, requiredMinVersion)()
	if err != nil {
		t.Fatalf("Failed to fetch GitLab version: %+v", err)
	}

	return isAtLeast
}

func IsRunningOnSaaS(t *testing.T) bool {
	t.Helper()

	return TestGitlabClient.BaseURL().Host == "gitlab.com"
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

// CreateProjectWithNamespace is a test helper for creating a project. This method accepts a namespace to create a project
// within a group
func CreateProjectWithNamespace(t *testing.T, namespaceID int64) *gitlab.Project {
	t.Helper()

	options := &gitlab.CreateProjectOptions{
		Name:        gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
		Description: gitlab.Ptr("Terraform acceptance tests"),
		// So that acceptance tests can be run in a gitlab organization with no billing.
		Visibility: gitlab.Ptr(gitlab.PublicVisibility),
		// So that a branch is created.
		InitializeWithReadme: gitlab.Ptr(true),
	}

	// Apply a namespace if one is passed in.
	if namespaceID != 0 {
		options.NamespaceID = gitlab.Ptr(namespaceID)
	}

	return CreateProjectWithOptions(t, options)
}

// CreateProjectWithOptions is a test helper for creating a project given some options
func CreateProjectWithOptions(t *testing.T, opts *gitlab.CreateProjectOptions) *gitlab.Project {
	t.Helper()

	if IsRunningOnSaaS(t) && (opts.NamespaceID == nil || opts.NamespaceID == gitlab.Ptr(int64(0))) {
		namespaceID, err := strconv.ParseInt(os.Getenv("GITLAB_SAAS_NAMESPACE_ID"), 10, 64)
		if err != nil {
			t.Fatalf("could not parse GITLAB_SAAS_NAMESPACE_ID: %v", err)
		}
		opts.NamespaceID = gitlab.Ptr(namespaceID)
	}

	project, _, err := TestGitlabClient.Projects.CreateProject(opts)
	if err != nil {
		t.Fatalf("could not create test project: %v", err)
	}

	t.Cleanup(func() {
		_, err := TestGitlabClient.Projects.DeleteProject(project.ID, nil)
		if err != nil {
			t.Fatalf("could not cleanup test project: %v", err)
		}

		if IsRunningOnSaaS(t) {
			softDeletedProject, _, err := TestGitlabClient.Projects.GetProject(project.ID, nil)
			if err != nil {
				t.Fatalf("could not check test project status: %v", err)
			}
			_, err = TestGitlabClient.Projects.DeleteProject(project.ID, &gitlab.DeleteProjectOptions{
				PermanentlyRemove: gitlab.Ptr(true),
				FullPath:          gitlab.Ptr(softDeletedProject.PathWithNamespace),
			})
			if err != nil {
				t.Fatalf("could not cleanup test project: %v", err)
			}
		}
	})

	return project
}

func CreateProjectWithDefaultPushRules(t *testing.T, namespaceID int64) *gitlab.Project {
	t.Helper()

	project := CreateProjectWithNamespace(t, namespaceID)

	/*
		Setting the default options based on what is set on gitlab.com
		when a new project is created.  The following are the push rules
		returned via the push_rule API after a project is created:

			{
			"id": 123,
			"project_id": 42,
			"created_at": "2024-07-25T15:49:50.119Z",
			"commit_message_regex": "",
			"commit_message_negative_regex": null,
			"branch_name_regex": null,
			"deny_delete_tag": false,
			"member_check": false,
			"prevent_secrets": false,
			"author_email_regex": "",
			"file_name_regex": "",
			"max_file_size": 0,
			"commit_committer_check": null,
			"commit_committer_name_check": false,
			"reject_unsigned_commits": null,
			"reject_non_dco_commits": null
			}
	*/
	options := &gitlab.AddProjectPushRuleOptions{
		AuthorEmailRegex:           nil,
		BranchNameRegex:            nil,
		CommitCommitterCheck:       nil,
		CommitCommitterNameCheck:   gitlab.Ptr(false),
		CommitMessageNegativeRegex: nil,
		CommitMessageRegex:         nil,
		DenyDeleteTag:              gitlab.Ptr(false),
		FileNameRegex:              nil,
		MaxFileSize:                gitlab.Ptr(int64(0)),
		MemberCheck:                gitlab.Ptr(false),
		PreventSecrets:             gitlab.Ptr(false),
		RejectUnsignedCommits:      nil,
		RejectNonDCOCommits:        nil,
	}

	_, _, err := TestGitlabClient.Projects.AddProjectPushRule(project.ID, options)
	if err != nil {
		t.Fatalf("could not create test project with default push rules")
	}

	return project
}

// CreateProjectMirror is a test helper for creating a project mirror.
// It assumes the project will be destroyed at the end of the test and will not cleanup created mirrors.
func CreateProjectMirrorWithOptions(t *testing.T, project *gitlab.Project, opts *gitlab.AddProjectMirrorOptions) *gitlab.ProjectMirror {
	t.Helper()

	var err error
	projectMirror, _, err := TestGitlabClient.ProjectMirrors.AddProjectMirror(project.ID, opts)
	if err != nil {
		t.Fatalf("could not create project mirror: %v", err)
	}

	return projectMirror
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
			Password:         gitlab.Ptr("42UltraSecurePotatoes."),
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

// Create Personal Access Token for a given user with `api` scope
func CreatePersonalAccessToken(t *testing.T, user *gitlab.User) *gitlab.PersonalAccessToken {
	t.Helper()

	return CreatePersonalAccessTokenWithScopes(t, user, []string{"api"})
}

// Create Personal Access Token for a given user with specified scopes
func CreatePersonalAccessTokenWithScopes(t *testing.T, user *gitlab.User, scopes []string) *gitlab.PersonalAccessToken {
	t.Helper()

	token, _, err := TestGitlabClient.Users.CreatePersonalAccessToken(user.ID, &gitlab.CreatePersonalAccessTokenOptions{Name: gitlab.Ptr(acctest.RandomWithPrefix("acctest")), Scopes: &scopes})
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

	var parentID *int64
	if IsRunningOnSaaS(t) {
		namespaceID, err := strconv.ParseInt(os.Getenv("GITLAB_SAAS_NAMESPACE_ID"), 10, 64)
		if err != nil {
			t.Fatalf("could not parse GITLAB_SAAS_NAMESPACE_ID: %v", err)
		}
		parentID = gitlab.Ptr(namespaceID)
	}
	groups := make([]*gitlab.Group, n)

	for i := range groups {
		var err error
		name := acctest.RandomWithPrefix(prefix)
		groups[i], _, err = TestGitlabClient.Groups.CreateGroup(&gitlab.CreateGroupOptions{
			Name: gitlab.Ptr(name),
			Path: gitlab.Ptr(name),
			// So that acceptance tests can be run in a gitlab organization with no billing.
			Visibility: gitlab.Ptr(gitlab.PublicVisibility),
			ParentID:   parentID,
		})
		if err != nil {
			t.Fatalf("could not create test group: %v", err)
		}

		groupID := groups[i].ID // Needed for closure.
		t.Cleanup(func() {
			_, err := TestGitlabClient.Groups.DeleteGroup(groupID, &gitlab.DeleteGroupOptions{})
			if err != nil {
				t.Fatalf("could not cleanup test group: %v", err)
			}
			if IsRunningOnSaaS(t) {
				softDeletedGroup, _, err := TestGitlabClient.Groups.GetGroup(groupID, nil)
				if err != nil {
					t.Fatalf("could not check test group status: %v", err)
				}
				_, err = TestGitlabClient.Groups.DeleteGroup(groupID, &gitlab.DeleteGroupOptions{
					PermanentlyRemove: gitlab.Ptr(true),
					FullPath:          gitlab.Ptr(softDeletedGroup.FullPath),
				})
				if err != nil {
					t.Fatalf("could not cleanup test group: %v", err)
				}
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

func CreateGroupHooks(t *testing.T, gid any, n int) []*gitlab.GroupHook {
	t.Helper()

	var hooks []*gitlab.GroupHook
	for range n {
		hook, _, err := TestGitlabClient.Groups.AddGroupHook(gid, &gitlab.AddGroupHookOptions{
			URL:         gitlab.Ptr(fmt.Sprintf("https://%s.com", acctest.RandomWithPrefix("acctest"))),
			EmojiEvents: gitlab.Ptr(true),
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

// CreateTags is a test helper for creating a specified number of tags.
// It assumes the project will be destroyed at the end of the test and will not cleanup created tags.
func CreateTags(t *testing.T, project *gitlab.Project, n int) []*gitlab.Tag {
	t.Helper()

	tags := make([]*gitlab.Tag, n)

	for i := range tags {
		var err error
		tags[i], _, err = TestGitlabClient.Tags.CreateTag(project.ID, &gitlab.CreateTagOptions{
			TagName: gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
			Ref:     gitlab.Ptr(project.DefaultBranch),
		})
		if err != nil {
			t.Fatalf("could not create test tags: %v", err)
		}
	}

	return tags
}

// CreateProtectedTags is a test helper for creating a specified number of protected tags.
// It assumes the project will be destroyed at the end of the test and will not cleanup created tags.
func CreateProtectedTags(t *testing.T, project *gitlab.Project, n int) []*gitlab.ProtectedTag {
	t.Helper()

	tags := CreateTags(t, project, n)
	protectedTags := make([]*gitlab.ProtectedTag, n)

	for i := range make([]int, n) {
		var err error
		protectedTags[i], _, err = TestGitlabClient.ProtectedTags.ProtectRepositoryTags(project.ID, &gitlab.ProtectRepositoryTagsOptions{
			Name: gitlab.Ptr(tags[i].Name),
		})
		if err != nil {
			t.Fatalf("could not protect test tags: %v", err)
		}
	}

	return protectedTags
}

// CreateProtectedTagWithOptions is a test helper for creating a protected tag with specified options.
// It assumes the project will be destroyed at the end of the test and will not cleanup created tags.
func CreateProtectedTagWithOptions(t *testing.T, project *gitlab.Project, opts *gitlab.ProtectRepositoryTagsOptions) *gitlab.ProtectedTag {
	t.Helper()

	tagName := acctest.RandomWithPrefix("acctest")
	// use the name provided on the options, otherwise if not provided use a random one
	if opts.Name != nil && *opts.Name != "" {
		tagName = *opts.Name
	} else {
		opts.Name = gitlab.Ptr(tagName)
	}

	_, _, err := TestGitlabClient.Tags.CreateTag(project.ID, &gitlab.CreateTagOptions{
		TagName: gitlab.Ptr(tagName),
		Ref:     gitlab.Ptr(project.DefaultBranch),
	})
	if err != nil {
		t.Fatalf("could not create test tag: %v", err)
	}

	protectedTag, _, err := TestGitlabClient.ProtectedTags.ProtectRepositoryTags(project.ID, opts)
	if err != nil {
		t.Fatalf("could not protect test tag: %v", err)
	}

	return protectedTag
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

// CreateMergeRequest is a test helper for creating a merge request.
// It assumes that the project will be destroyed at the end of the test and will not cleanup created merge requests.
func CreateMergeRequest(t *testing.T, assignee *gitlab.User, project *gitlab.Project, source string, target string) *gitlab.MergeRequest {
	opts := gitlab.CreateMergeRequestOptions{
		Title:        gitlab.Ptr(acctest.RandomWithPrefix(source)),
		Description:  gitlab.Ptr(acctest.RandomWithPrefix(source)),
		SourceBranch: &source,
		TargetBranch: &target,
	}
	if assignee != nil {
		opts.AssigneeID = &assignee.ID
		opts.AssigneeIDs = &[]int64{assignee.ID}
	}

	mergeRequest, _, err := TestGitlabClient.MergeRequests.CreateMergeRequest(
		project.ID, &opts,
	)
	if err != nil {
		t.Fatalf("could not create merge request: %v", err)
	}
	return mergeRequest
}

// CloseMergeRequest is a test helper for closing a merge request.
// It assumes that the project will be destroyed at the end of the test and will not cleanup closed merge requests.
func CloseMergeRequest(t *testing.T, project *gitlab.Project, mr *gitlab.MergeRequest) *gitlab.MergeRequest {
	opts := gitlab.UpdateMergeRequestOptions{StateEvent: gitlab.Ptr("close")}
	mergeRequest, _, err := TestGitlabClient.MergeRequests.UpdateMergeRequest(
		project.ID, mr.IID, &opts,
	)
	if err != nil {
		t.Fatalf("could not close merge request: %v", err)
	}
	return mergeRequest
}

// AddProjectMembers is a test helper for adding users as members of a project with Developer access level.
// It assumes the project will be destroyed at the end of the test and will not cleanup members.
func AddProjectMembers(t *testing.T, pid any, users []*gitlab.User) {
	t.Helper()

	AddProjectMembersWithAccessLevel(t, pid, users, gitlab.DeveloperPermissions)
}

// AddProjectMembersWithAccessLevel is a test helper for adding users as members of a project with a given access level.
// It assumes the project will be destroyed at the end of the test and will not cleanup members.
func AddProjectMembersWithAccessLevel(t *testing.T, pid any, users []*gitlab.User, accessLevel gitlab.AccessLevelValue) {
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

func CreateProjectHooks(t *testing.T, pid any, n int) []*gitlab.ProjectHook {
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

func CreateClusterAgents(t *testing.T, pid any, n int) []*gitlab.Agent {
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

func SetupUserAccess(t *testing.T, project *gitlab.Project, agent *gitlab.Agent) {
	t.Helper()

	agentCfgPath := fmt.Sprintf(".gitlab/agents/%s/config.yaml", agent.Name)
	userAccessCfg := fmt.Appendf(nil, `
user_access:
  access_as:
    agent: {}
  projects:
    - id: %q
`, project.PathWithNamespace)
	_, _, err := TestGitlabClient.RepositoryFiles.CreateFile(project.ID, agentCfgPath, &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr(project.DefaultBranch),
		Encoding:      gitlab.Ptr("base64"),
		Content:       gitlab.Ptr(base64.StdEncoding.EncodeToString(userAccessCfg)),
		CommitMessage: gitlab.Ptr(fmt.Sprintf("Setup user access for agent %s in acceptance tests", agent.Name)),
	})
	if err != nil {
		t.Fatalf("unable to setup user access for agent %s in project %s: %v", agent.Name, project.PathWithNamespace, err)
	}
}

func CreateProjectIssues(t *testing.T, pid any, n int) []*gitlab.Issue {
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
	query := gitlab.GraphQLQuery{
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

	var pid any
	if _, err := TestGitlabClient.GraphQL.Do(query, &pid); err != nil {
		t.Fatalf("Unable to create epic board: %s", err.Error())
	}
}

func CreateGroupIssueBoard(t *testing.T, pid any) *gitlab.GroupIssueBoard {
	t.Helper()

	issueBoard, _, err := TestGitlabClient.GroupIssueBoards.CreateGroupIssueBoard(pid, &gitlab.CreateGroupIssueBoardOptions{Name: gitlab.Ptr(acctest.RandomWithPrefix("acctest"))})
	if err != nil {
		t.Fatalf("could not create test group issue board: %v", err)
	}

	return issueBoard
}

func CreateProjectIssueBoard(t *testing.T, pid any) *gitlab.IssueBoard {
	t.Helper()

	issueBoard, _, err := TestGitlabClient.Boards.CreateIssueBoard(pid, &gitlab.CreateIssueBoardOptions{Name: gitlab.Ptr(acctest.RandomWithPrefix("acctest"))})
	if err != nil {
		t.Fatalf("could not create test issue board: %v", err)
	}

	return issueBoard
}

func CreateGroupLabels(t *testing.T, gid any, n int) []*gitlab.GroupLabel {
	t.Helper()

	var labels []*gitlab.GroupLabel
	for range n {
		label, _, err := TestGitlabClient.GroupLabels.CreateGroupLabel(gid, &gitlab.CreateGroupLabelOptions{Name: gitlab.Ptr(acctest.RandomWithPrefix("acctest")), Color: gitlab.Ptr("#000000")})
		if err != nil {
			t.Fatalf("could not create test group label: %v", err)
		}
		labels = append(labels, label)
	}

	return labels
}

func CreateProjectLabels(t *testing.T, pid any, n int) []*gitlab.Label {
	t.Helper()

	var labels []*gitlab.Label
	for range n {
		label, _, err := TestGitlabClient.Labels.CreateLabel(pid, &gitlab.CreateLabelOptions{Name: gitlab.Ptr(acctest.RandomWithPrefix("acctest")), Color: gitlab.Ptr("#000000")})
		if err != nil {
			t.Fatalf("could not create test label: %v", err)
		}
		labels = append(labels, label)
	}

	return labels
}

// AddLabelToIssue is a test helper for adding a label to an issue.
// It assumes the project and issue will be destroyed at the end of the test and will not cleanup the label assignment.
func AddLabelToIssue(t *testing.T, projectID any, issueIID int64, label *gitlab.Label) {
	t.Helper()
	opts := &gitlab.UpdateIssueOptions{
		AddLabels: &gitlab.LabelOptions{label.Name},
	}
	_, _, err := TestGitlabClient.Issues.UpdateIssue(projectID, issueIID, opts)
	if err != nil {
		t.Fatalf("could not add label to issue %d: %v", issueIID, err)
	}
}

// AddGroupMembers is a test helper for adding users as members of a group with Developer level access.
// It assumes the group will be destroyed at the end of the test and will not cleanup members.
func AddGroupMembers(t *testing.T, gid any, users []*gitlab.User) {
	t.Helper()

	AddGroupMembersWithAccessLevel(t, gid, users, gitlab.DeveloperPermissions)
}

// GroupShareGroup shares a group with another group with a developer access level and finite date.
func GroupShareGroup(t *testing.T, parentGid any, sharedGid *int64) *gitlab.Group {
	t.Helper()

	endDate := time.Date(2023, 12, 21, 0, 0, 0, 0, time.UTC)
	exp := gitlab.ISOTime(endDate)

	group, _, err := TestGitlabClient.Groups.ShareGroupWithGroup(parentGid, &gitlab.ShareGroupWithGroupOptions{
		GroupID:     sharedGid,
		GroupAccess: gitlab.Ptr(gitlab.DeveloperPermissions),
		ExpiresAt:   &exp,
	})
	if err != nil {
		t.Fatalf("could not share group with group: %v", err)
	}
	return group
}

// AddGroupMembersWithAccessLevel is a test helper for adding users as members of a group with a given access level.
func AddGroupMembersWithAccessLevel(t *testing.T, gid any, users []*gitlab.User, accessLevel gitlab.AccessLevelValue) {
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
func ProjectShareGroup(t *testing.T, pid any, gid int64) {
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
func ListProjectMembers(t *testing.T, pid any) {
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

func CreateDeployKey(t *testing.T, projectID int64, options *gitlab.AddDeployKeyOptions) *gitlab.ProjectDeployKey {
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

// AddDeployKey is a test helper function for adding a deploy key to a project with default options
func AddDeployKey(t *testing.T, projectID int64) *gitlab.ProjectDeployKey {
	t.Helper()

	rInt := acctest.RandInt()
	// Generate an ED25519 SSH key (compatible with GitLab CE 18.6+)
	// This format is accepted by newer GitLab versions that have stricter SSH key validation
	sshKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGbPhiQgg7/l7kicmUmT9WdwecHsxJmv7rTXM5phfUML test@example.com"

	options := &gitlab.AddDeployKeyOptions{
		Title:   gitlab.Ptr(fmt.Sprintf("test-deploy-key-%d", rInt)),
		Key:     gitlab.Ptr(sshKey),
		CanPush: gitlab.Ptr(true),
	}

	return CreateDeployKey(t, projectID, options)
}

// CreateProjectEnvironment is a test helper function for creating a project environment
func CreateProjectEnvironment(t *testing.T, projectID int64, options *gitlab.CreateEnvironmentOptions) *gitlab.Environment {
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

func CreateProjectVariable(t *testing.T, projectID int64) *gitlab.ProjectVariable {
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

func CreateGroupVariable(t *testing.T, groupID int64) *gitlab.GroupVariable {
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
		Key:         gitlab.Ptr(fmt.Sprintf("test_key_%d", acctest.RandInt())),
		Value:       gitlab.Ptr("test_value"),
		Description: gitlab.Ptr("test_description"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.InstanceVariables.RemoveVariable(variable.Key); err != nil {
			t.Fatal(err)
		}
	})

	return variable
}

func CreateProjectFile(t *testing.T, projectID int64, fileContent string, filePath string, branch string) *gitlab.FileInfo {
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

func CreateProjectFilePlaintext(t *testing.T, projectID int64, fileContent string, filePath string, branch string) *gitlab.FileInfo {
	file, _, err := TestGitlabClient.RepositoryFiles.CreateFile(projectID, filePath, &gitlab.CreateFileOptions{
		Branch:        &branch,
		Encoding:      gitlab.Ptr("text"),
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

	query := gitlab.GraphQLQuery{
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
	if _, err := TestGitlabClient.GraphQL.Do(query, &response); err != nil {
		t.Fatalf("Unable to create compliance framework: %s", err.Error())
	}

	t.Cleanup(func() {
		query := gitlab.GraphQLQuery{
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

		if _, err := TestGitlabClient.GraphQL.Do(query, nil); err != nil {
			t.Fatalf("Unable to delete compliance framework: %s", err.Error())
		}
	})

	return &response.Data.CreateComplianceFramework.Framework
}

func DeleteProjectComplianceFrameworks(t *testing.T, project *gitlab.Project) {
	t.Helper()

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				projectUpdateComplianceFrameworks(
					input: {
						projectId: "gid://gitlab/Project/%d",
						complianceFrameworkIds: []
					}
				) {
					project {
						id,
						complianceFrameworks {
							nodes {
								id
							}
						}
					}
					errors
				}
			}`, project.ID),
	}

	if _, err := TestGitlabClient.GraphQL.Do(query, nil); err != nil {
		t.Fatalf("Unable to delete project compliance frameworks: %s", err.Error())
	}
}

func CreateScheduledPipeline(t *testing.T, project int64, branch string) (*gitlab.PipelineSchedule, error) {
	t.Helper()

	// check if the branch is a full ref value, otherwise add "refs/heads/" to the front
	if !strings.HasPrefix(branch, "refs/heads") {
		branch = fmt.Sprintf("refs/heads/%s", branch)
	}

	var pipeline *gitlab.PipelineSchedule
	pipeline, _, err := TestGitlabClient.PipelineSchedules.CreatePipelineSchedule(project, &gitlab.CreatePipelineScheduleOptions{
		Description:  gitlab.Ptr("test"),
		Ref:          gitlab.Ptr(branch),
		Cron:         gitlab.Ptr("0 0 1 1 *"),
		CronTimezone: gitlab.Ptr("UTC"),
	})

	t.Cleanup(func() {
		_, _ = TestGitlabClient.PipelineSchedules.DeletePipelineSchedule(project, pipeline.ID)
	})

	return pipeline, err
}

// Function for easily calculating the expiry days from the current time.
func GetCurrentTimePlusDays(t *testing.T, days int) gitlab.ISOTime {
	now := time.Now()
	expiryDate := now.AddDate(0, 0, days)
	return gitlab.ISOTime(expiryDate)
}

// Function for easily calculating a new timestamp value from the current time.
func GetCurrentTimestampPlusDays(t *testing.T, days int) time.Time {
	now := time.Now()
	return now.AddDate(0, 0, days)
}

// CreateGroupServiceAccounts is a test helper for creating a specified number of service accounts.
func CreateGroupServiceAccounts(t *testing.T, n int, groupID string) []*gitlab.GroupServiceAccount {
	return CreateGroupServiceAccountsWithPrefix(t, n, groupID, "acctest-service-account")
}

func CreateGroupServiceAccountsWithPrefix(t *testing.T, n int, groupID, prefix string) []*gitlab.GroupServiceAccount {
	t.Helper()

	serviceAccounts := make([]*gitlab.GroupServiceAccount, n)

	for i := range serviceAccounts {
		var err error
		name := acctest.RandomWithPrefix(prefix)
		username := acctest.RandomWithPrefix(prefix)
		serviceAccounts[i], _, err = TestGitlabClient.Groups.CreateServiceAccount(groupID, &gitlab.CreateServiceAccountOptions{
			Name:     gitlab.Ptr(name),
			Username: gitlab.Ptr(username),
		})
		if err != nil {
			t.Fatalf("could not create test service account (group_id=%q username=%q): %v", groupID, username, err)
		}

		serviceAccountID := serviceAccounts[i].ID // Needed for closure.
		t.Cleanup(func() {
			if _, err := TestGitlabClient.Groups.DeleteServiceAccount(groupID, serviceAccountID, nil); err != nil {
				t.Fatalf("could not cleanup test service account: %v", err)
			}
		})
	}
	return serviceAccounts
}

func CreateInstanceServiceAccounts(t *testing.T, n int) []*gitlab.User {
	return CreateInstanceServiceAccountsWithPrefix(t, n, "acctest-service-account")
}

func CreateInstanceServiceAccountsWithPrefix(t *testing.T, n int, prefix string) []*gitlab.User {
	t.Helper()

	serviceAccounts := make([]*gitlab.User, n)

	for i := range serviceAccounts {
		var err error
		name := acctest.RandomWithPrefix(prefix)
		username := acctest.RandomWithPrefix(prefix)
		serviceAccounts[i], _, err = TestGitlabClient.Users.CreateServiceAccountUser(&gitlab.CreateServiceAccountUserOptions{
			Name:     gitlab.Ptr(name),
			Username: gitlab.Ptr(username),
		})
		if err != nil {
			t.Fatalf("could not create test service account (username=%q): %v", username, err)
		}

		serviceAccountID := serviceAccounts[i].ID // Needed for closure.
		t.Cleanup(func() {
			if _, err := TestGitlabClient.Users.DeleteUser(serviceAccountID); err != nil {
				t.Fatalf("could not cleanup test service account: %v", err)
			}
		})
	}
	return serviceAccounts
}

func CreateGroupServiceAccountAccessToken(t *testing.T, groupID int64, serviceAccountID int64, name string, scopes []string) *gitlab.PersonalAccessToken {
	t.Helper()

	groupServiceAccountAccessToken, _, err := TestGitlabClient.Groups.CreateServiceAccountPersonalAccessToken(groupID, serviceAccountID, &gitlab.CreateServiceAccountPersonalAccessTokenOptions{
		Name:      gitlab.Ptr(name),
		Scopes:    gitlab.Ptr(scopes),
		ExpiresAt: gitlab.Ptr(gitlab.ISOTime(time.Now().AddDate(0, 0, 7))),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.Groups.RevokeServiceAccountPersonalAccessToken(groupID, serviceAccountID, groupServiceAccountAccessToken.ID); err != nil {
			t.Fatal(err)
		}
	})

	return groupServiceAccountAccessToken
}

// CreateRunnerWithOptions is a test helper for creating a Runner given some options
func CreateRunnerWithOptions(t *testing.T, opts *gitlab.CreateUserRunnerOptions) *gitlab.UserRunner {
	t.Helper()

	runner, _, err := TestGitlabClient.Users.CreateUserRunner(opts)
	if err != nil {
		t.Fatalf("could not create runner: %v", err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.Runners.DeleteRegisteredRunnerByID(runner.ID); err != nil {
			t.Fatalf("could not cleanup runner: %v", err)
		}
	})
	return runner
}

func CreateGroupAccessToken(t *testing.T, groupID int64, name string, scopes []string, accessLevel gitlab.AccessLevelValue) *gitlab.GroupAccessToken {
	groupAccessToken, _, err := TestGitlabClient.GroupAccessTokens.CreateGroupAccessToken(groupID, &gitlab.CreateGroupAccessTokenOptions{
		Name:        gitlab.Ptr(name),
		Scopes:      gitlab.Ptr(scopes),
		AccessLevel: gitlab.Ptr(accessLevel),
		ExpiresAt:   gitlab.Ptr(gitlab.ISOTime(time.Now().AddDate(0, 0, 7))),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if _, err := TestGitlabClient.GroupAccessTokens.RevokeGroupAccessToken(groupID, groupAccessToken.ID); err != nil {
			t.Fatal(err)
		}
	})

	return groupAccessToken
}

func CreateCustomInstanceRole(t *testing.T, input *gitlab.CreateMemberRoleOptions) *gitlab.MemberRole {
	role, _, err := TestGitlabClient.MemberRolesService.CreateInstanceMemberRole(input)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_, err = TestGitlabClient.MemberRolesService.DeleteInstanceMemberRole(role.ID)
		if err != nil {
			t.Fatalf("Failed to destroy custom role: %v", err)
		}
	})

	return role
}

func CreateProjectAccessToken(t *testing.T, projectID int64, name string, scopes []string, accessLevel gitlab.AccessLevelValue, description *string) *gitlab.
	ProjectAccessToken {
	options := &gitlab.CreateProjectAccessTokenOptions{
		Name:        gitlab.Ptr(name),
		Description: description,
		Scopes:      gitlab.Ptr(scopes),
		AccessLevel: gitlab.Ptr(accessLevel),
		ExpiresAt:   gitlab.Ptr(gitlab.ISOTime(time.Now().AddDate(0, 11, 0))),
	}

	projectAccessToken, _, err := TestGitlabClient.ProjectAccessTokens.CreateProjectAccessToken(projectID, options)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := TestGitlabClient.ProjectAccessTokens.RevokeProjectAccessToken(projectID, projectAccessToken.ID); err != nil && !api.Is404(err) {
			t.Fatal(err)
		}
	})

	return projectAccessToken
}

// This helper should only be used if you need to create a GitLab Client with a bespoke token.
// Generally you should use
//
//	testutil.TestGitLabClient
func CreateGitlabClientWithToken(t *testing.T, token string) *gitlab.Client {
	clientConfig := api.Config{
		Token:   token,
		BaseURL: os.Getenv("GITLAB_BASE_URL"),
	}
	client, err := clientConfig.NewGitLabClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func CreateProjectApprovalRule(t *testing.T, project int64, ruleName string, approvalsRequired int64, userIDs []int64, groupIDs []int64, protectedBranchIDs []int64) (*gitlab.ProjectApprovalRule, error) {
	t.Helper()

	approvalRuleOptions := gitlab.CreateProjectLevelRuleOptions{
		Name:                          gitlab.Ptr(ruleName),
		RuleType:                      gitlab.Ptr("regular"),
		ApprovalsRequired:             gitlab.Ptr(approvalsRequired),
		AppliesToAllProtectedBranches: gitlab.Ptr(false),
	}

	if len(userIDs) > 0 {
		approvalRuleOptions.UserIDs = gitlab.Ptr(userIDs)
	}
	if len(groupIDs) > 0 {
		approvalRuleOptions.GroupIDs = gitlab.Ptr(groupIDs)
	}
	if len(protectedBranchIDs) > 0 {
		approvalRuleOptions.ProtectedBranchIDs = gitlab.Ptr(protectedBranchIDs)
	}

	approvalRule, _, err := TestGitlabClient.Projects.CreateProjectApprovalRule(project, &approvalRuleOptions)

	t.Cleanup(func() {
		_, _ = TestGitlabClient.Projects.DeleteProjectApprovalRule(project, approvalRule.ID)
	})

	return approvalRule, err
}

func CreateMemberRole(t *testing.T) *api.GraphQLMemberRole {
	t.Helper()

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				memberRoleCreate(
					input: {
						name: "%s",
						baseAccessLevel: REPORTER,
						permissions: [READ_VULNERABILITY]
					}
				) {
					memberRole {
						baseAccessLevel {
							stringValue
						},
						createdAt,
						description,
						editPath,
						enabledPermissions {
							nodes {
								value
							}
						},
						id,
						name
					}
					errors
				}
			}`, acctest.RandomWithPrefix("acctest")),
	}

	type createMemberRoleResponse struct {
		Data struct {
			MemberRoleCreate struct {
				MemberRole api.GraphQLMemberRole `json:"memberRole"`
				Errors     []string              `json:"errors"`
			} `json:"memberRoleCreate"`
		} `json:"data"`
		Errors []struct {
			Message   string `json:"message"`
			Locations []struct {
				Line   int `json:"line"`
				Column int `json:"column"`
			} `json:"locations"`
			Path []string `json:"path"`
		} `json:"errors"`
	}

	var response createMemberRoleResponse
	if _, err := TestGitlabClient.GraphQL.Do(query, &response); err != nil {
		t.Fatalf("Unable to create member role: %s", err.Error())
	}

	// check response for errors
	var allerr string
	if len(response.Errors) > 0 {
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
		}
	}
	if len(response.Data.MemberRoleCreate.Errors) > 0 {
		for i, err := range response.Data.MemberRoleCreate.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
		}
	}
	if len(allerr) > 0 {
		t.Fatalf("GitLab GraphQL error occurred: %s", allerr)
	}

	t.Cleanup(func() {
		query := gitlab.GraphQLQuery{
			Query: fmt.Sprintf(`
				mutation {
					memberRoleDelete(
						input: {
							id: "%s"
						}
					) {
						errors
					}
				}`, response.Data.MemberRoleCreate.MemberRole.ID),
		}

		if _, err := TestGitlabClient.GraphQL.Do(query, nil); err != nil {
			t.Fatalf("Unable to delete member role: %s", err.Error())
		}
	})

	return &response.Data.MemberRoleCreate.MemberRole
}

func CreateProjectSecureFile(t *testing.T, pid any, n int) []*gitlab.SecureFile {
	var secureFiles = make([]*gitlab.SecureFile, 0, n)
	for i := range n {
		contentReader := strings.NewReader(fmt.Sprintf("secure file content %d", i))
		secureFile, _, err := TestGitlabClient.SecureFiles.CreateSecureFile(pid, contentReader, &gitlab.CreateSecureFileOptions{
			Name: gitlab.Ptr(fmt.Sprintf("secure-file-%d", i)),
		})
		if err != nil {
			t.Fatalf("could not create test secure file: %v", err)
		}
		t.Cleanup(func() {
			if _, err := TestGitlabClient.SecureFiles.RemoveSecureFile(pid, secureFile.ID); err != nil {
				t.Fatalf("could not cleanup test secure file: %v", err)
			}
		})
		secureFiles = append(secureFiles, secureFile)
	}

	return secureFiles
}

// This helper updates the `EnforceCIInboundJobTokenScopeEnabled` setting on the instance, then
// when the test completes, reverts the setting to the value that it was previously configured to.
func UpdateEnforceCIInboundJobTokenScopeEnabledSetting(t *testing.T, enabled bool) {

	// Get the current settings for the application so when we revert, we revert properly
	setting, _, err := TestGitlabClient.Settings.GetSettings()
	if err != nil {
		t.Fatalf("failed to get existing application settings: %v", err)
	}
	currentCISetting := setting.EnforceCIInboundJobTokenScopeEnabled

	// Update the application setting to the value passed into the helper
	_, _, err = TestGitlabClient.Settings.UpdateSettings(&gitlab.UpdateSettingsOptions{
		EnforceCIInboundJobTokenScopeEnabled: gitlab.Ptr(enabled),
	})
	if err != nil {
		t.Fatalf("failed to update application setting for enforcing CI/CD job tokens: %v", err)
	}

	// Restore the original settings after the test
	t.Cleanup(func() {
		_, _, err = TestGitlabClient.Settings.UpdateSettings(&gitlab.UpdateSettingsOptions{
			EnforceCIInboundJobTokenScopeEnabled: &currentCISetting,
		})
		if err != nil {
			t.Logf("Warning: Failed to restore original application settings: %v", err)
		}
	})
}
