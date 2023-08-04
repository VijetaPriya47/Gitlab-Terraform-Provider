//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabRepositoryFile_basic(t *testing.T) {
	var file gitlab.File
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGitlabRepositoryFileConfig(testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3c=",
					}),
					resource.TestCheckResourceAttr("gitlab_repository_file.this", "content", "bWVvdyBtZW93IG1lb3c="),
				),
			},
			{
				ResourceName:            "gitlab_repository_file.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"author_email", "author_name", "commit_message"},
			},
			{
				Config: testAccGitlabRepositoryFileUpdateConfig(testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg==",
					}),
					resource.TestCheckResourceAttr("gitlab_repository_file.this", "content", "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg=="),
				),
			},
			{
				ResourceName:            "gitlab_repository_file.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"author_email", "author_name", "commit_message"},
			},
		},
	})
}

func TestAccGitlabRepositoryFile_SeparateCreateUpdateCommitMessages(t *testing.T) {
	var file gitlab.File
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_repository_file" "this" {
				  project = %d
				  file_path = "meow.txt"
				  branch = "main"
				  content = "bWVvdyBtZW93IG1lb3c="
				  author_email = "meow@catnip.com"
				  author_name = "Meow Meowington"
				  create_commit_message = "feature: add launch codes"
				  update_commit_message = "update: updated launch codes"
				  delete_commit_message = "delete: deleted launch codes"
				}
					`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3c=",
					}),
					testAccCheckGitlabRepositoryFileCommitMessage("gitlab_repository_file.this", &file, "feature: add launch codes"),
					resource.TestCheckResourceAttr("gitlab_repository_file.this", "content", "bWVvdyBtZW93IG1lb3c="),
				),
			},
			{
				ResourceName:            "gitlab_repository_file.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"author_email", "author_name", "create_commit_message", "update_commit_message", "delete_commit_message"},
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_repository_file" "this" {
				  project = %d
				  file_path = "meow.txt"
				  branch = "main"
				  content = "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg=="
				  author_email = "meow@catnip.com"
				  author_name = "Meow Meowington"
				  create_commit_message = "feature: add launch codes"
				  update_commit_message = "update: updated launch codes"
				  delete_commit_message = "delete: deleted launch codes"
				}
					`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg==",
					}),
					testAccCheckGitlabRepositoryFileCommitMessage("gitlab_repository_file.this", &file, "update: updated launch codes"),
					resource.TestCheckResourceAttr("gitlab_repository_file.this", "content", "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg=="),
				),
			},
			{
				ResourceName:            "gitlab_repository_file.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"author_email", "author_name", "create_commit_message", "update_commit_message", "delete_commit_message"},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_repository_file" "this" {
						project = %d
						file_path = "meow.txt"
						branch = "main"
						content = "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg=="
						author_email = "meow@catnip.com"
						author_name = "Meow Meowington"
						create_commit_message = "feature: add launch codes"
						update_commit_message = "update: updated launch codes"
						delete_commit_message = "delete: deleted launch codes"
					}
						`, testProject.ID),
				Destroy: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileDeleteCommitMessage("gitlab_repository_file.this", "delete: deleted launch codes"),
				),
			},
		},
	})
}

func TestAccGitlabRepositoryFile_EnsureErrorsWithCommitMessage(t *testing.T) {
	testProject := testutil.CreateProject(t)

	err_incompatable_commit_messages, err := regexp.Compile("conflicts with commit_message")
	if err != nil {
		t.Errorf("Unable to format expected conflicts error regex: %s", err)
	}

	err_missing_update_commit_message, err := regexp.Compile("Missing required argument")
	if err != nil {
		t.Errorf("Unable to format expected required error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_repository_file" "this" {
				  project = %d
				  file_path = "meow.txt"
				  branch = "main"
				  content = "bWVvdyBtZW93IG1lb3c="
				  author_email = "meow@catnip.com"
				  author_name = "Meow Meowington"
				  commit_message = "Extra commit message"
				  create_commit_message = "feature: add launch codes"
				  update_commit_message = "update: updated launch codes"
				  delete_commit_message = "delete: deleted launch codes"
				}
					`, testProject.ID),
				ExpectError: err_incompatable_commit_messages,
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_repository_file" "this" {
				  project = %d
				  file_path = "meow.txt"
				  branch = "main"
				  content = "bWVvdyBtZW93IG1lb3c="
				  author_email = "meow@catnip.com"
				  author_name = "Meow Meowington"
				  create_commit_message = "feature: add launch codes"
				  delete_commit_message = "delete: deleted launch codes"
				}
					`, testProject.ID),
				ExpectError: err_missing_update_commit_message,
			},
		},
	})
}

// Test that we can successfully migrate from a pre-16.0 provider to the
// post 16.0 provider when a plaintext encoding is applied.
func TestAccGitlabRepositoryFile_stateMigration(t *testing.T) {
	var file gitlab.File
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				// Create the project with an old version of the provider
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 15.9.0",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
				resource "gitlab_repository_file" "this" {
				  project = %d
				  file_path = "meow.txt"
				  branch = "main"
				  content = "Example Plaintext Data"
				  author_email = "meow@catnip.com"
				  author_name = "Meow Meowington"
				  commit_message = "feature: add launch codes"
				}
				`, testProject.ID),
			},
			{
				// Use the same configuration with the current version of the provider
				// Adding "encoding = text" to the resource to match new requirements
				ProtoV6ProviderFactories: providerFactoriesV6,
				Config: fmt.Sprintf(`
				resource "gitlab_repository_file" "this" {
				  project = %d
				  file_path = "meow.txt"
				  branch = "main"
				  content = "Example Plaintext Data"
				  encoding = "text"
				  author_email = "meow@catnip.com"
				  author_name = "Meow Meowington"
				  commit_message = "feature: add launch codes"
				}
				`, testProject.ID),
				Check: testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
			},
		},
	})
}

func TestAccGitlabRepositoryFile_overwriteOnCreate(t *testing.T) {
	var file gitlab.File
	testProject := testutil.CreateProject(t)
	options := &gitlab.CreateFileOptions{
		Branch:        gitlab.String("main"),
		Encoding:      gitlab.String("base64"),
		AuthorEmail:   gitlab.String("meow@catnip.com"),
		AuthorName:    gitlab.String("Meow Meowington"),
		Content:       gitlab.String("bWVvdyBtZW93IG1lb3c="),
		CommitMessage: gitlab.String("feature: cat"),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					if _, _, err := testutil.TestGitlabClient.RepositoryFiles.CreateFile(testProject.ID, "animal-noise.txt", options); err != nil {
						t.Fatalf("failed to create file: %v", err)
					}
				},
				Config: fmt.Sprintf(`
				resource "gitlab_repository_file" "this" {
				  project = %d
				  file_path = "animal-noise.txt"
				  branch = "main"
				  content = "d29vZiB3b29mIHdvb2YK"
				  author_email = "bark@dogbone.com"
				  author_name = "Bark Woofman"
				  commit_message = "feature: dog"
				  overwrite_on_create = true
				}
					`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "animal-noise.txt",
						Content:  "d29vZiB3b29mIHdvb2YK",
					}),
				),
			},
			{
				ResourceName:            "gitlab_repository_file.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"author_email", "author_name", "commit_message", "overwrite_on_create"},
			},
		},
	})
}

func TestAccGitlabRepositoryFile_overwriteOnCreateNewFile(t *testing.T) {
	var file gitlab.File
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_repository_file" "this" {
				  project = %d
				  file_path = "animal-noise.txt"
				  branch = "main"
				  content = "d29vZiB3b29mIHdvb2YK"
				  author_email = "bark@dogbone.com"
				  author_name = "Bark Woofman"
				  commit_message = "feature: dog"
				  overwrite_on_create = true
				}
					`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "animal-noise.txt",
						Content:  "d29vZiB3b29mIHdvb2YK",
					}),
				),
			},
			{
				ResourceName:            "gitlab_repository_file.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"author_email", "author_name", "commit_message", "overwrite_on_create"},
			},
		},
	})
}

func TestAccGitlabRepositoryFile_createSameFileDifferentRepository(t *testing.T) {
	var fooFile gitlab.File
	var barFile gitlab.File
	firstTestProject := testutil.CreateProject(t)
	secondTestProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGitlabRepositoryFileSameFileDifferentRepositoryConfig(firstTestProject.ID, secondTestProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.foo_file", &fooFile),
					testAccCheckGitlabRepositoryFileAttributes(&fooFile, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3c=",
					}),
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.bar_file", &barFile),
					testAccCheckGitlabRepositoryFileAttributes(&barFile, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3c=",
					}),
				),
			},
		},
	})
}

func TestAccGitlabRepositoryFile_concurrentResources(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			// NOTE: we don't need to check anything here, just make sure no terraform errors are being raised,
			//       the other test cases will do the actual testing :)
			{
				Config: testAccGitlabRepositoryFileConcurrentResourcesConfig(testProject.ID),
			},
			{
				Config: testAccGitlabRepositoryFileConcurrentResourcesConfigUpdate(testProject.ID),
			},
			{
				Config:  testAccGitlabRepositoryFileConcurrentResourcesConfigUpdate(testProject.ID),
				Destroy: true,
			},
		},
	})
}

func TestAccGitlabRepositoryFile_createOnNewBranch(t *testing.T) {
	var file gitlab.File
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGitlabRepositoryFileStartBranchConfig(testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3c=",
					}),
				),
			},
		},
	})
}

// This test ensures that the filePath doesn't start with / or ./
// see https://gitlab.com/gitlab-org/gitlab/-/issues/363112 for more info.
func TestAccGitlabRepositoryFile_validationFuncOnfilePath(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_repository_file" "this" {
					project = %d
					file_path = "./meow.txt"
					branch = "main"
					content = "bWVvdyBtZW93IG1lb3c="
					author_email = "meow@catnip.com"
					author_name = "Meow Meowington"
					commit_message = "feature: add launch codes"
				  }`, testProject.ID),
				ExpectError: regexp.MustCompile("`file_path` cannot start with a `/` or `./`. See https://gitlab.com/gitlab-org/gitlab/-/issues/363112 for more information."),
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_repository_file" "this" {
					project = %d
					file_path = "/meow.txt"
					branch = "main"
					content = "bWVvdyBtZW93IG1lb3c="
					author_email = "meow@catnip.com"
					author_name = "Meow Meowington"
					commit_message = "feature: add launch codes"
				  }`, testProject.ID),
				ExpectError: regexp.MustCompile("`file_path` cannot start with a `/` or `./`. See https://gitlab.com/gitlab-org/gitlab/-/issues/363112 for more information."),
			},
		},
	})
}

func TestAccGitlabRepositoryFile_base64EncodingWithTextContent(t *testing.T) {
	var file gitlab.File
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_repository_file" "this" {
						project = %d
						file_path = "meow.txt"
						branch = "main"

						encoding = "text"
						content  = "Hello World, meow"
						
						author_email = "meow@catnip.com"
						author_name = "Meow Meowington"
						commit_message = "feature: add launch codes"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						// This is still base64 encoded because it's from the API, not from state.
						Content: "SGVsbG8gV29ybGQsIG1lb3c=",
					}),
					// This checks the explicit state to ensure it's plaintext.
					resource.TestCheckResourceAttr("gitlab_repository_file.this", "content", "Hello World, meow"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_repository_file" "this" {
						project = %d
						file_path = "meow.txt"
						branch = "main"

						encoding = "base64"
						content = base64encode("Hello World, meow")

						author_email = "meow@catnip.com"
						author_name = "Meow Meowington"
						commit_message = "feature: add launch codes"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "SGVsbG8gV29ybGQsIG1lb3c=",
					}),
				),
			},
		},
	})
}

func TestAccGitlabRepositoryFile_createWithExecuteFilemode(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testutil.RunIfAtLeast(t, "14.10") },
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_repository_file" "this" {
						project = %d
						file_path = "meow.txt"
						branch = "main"
						content = "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg=="
						author_email = "meow@catnip.com"
						author_name = "Meow Meowington"
						commit_message = "feature: change launch codes"
						execute_filemode = false
					}
				`, testProject.ID),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_repository_file.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"author_email", "author_name", "commit_message"},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_repository_file" "this" {
						project = %d
						file_path = "meow.txt"
						branch = "main"
						content = "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg=="
						author_email = "meow@catnip.com"
						author_name = "Meow Meowington"
						commit_message = "feature: change launch codes"
						execute_filemode = true
					}
				`, testProject.ID),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_repository_file.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"author_email", "author_name", "commit_message"},
			},
		},
	})
}

func testAccCheckGitlabRepositoryFileExists(n string, file *gitlab.File) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		_, branch, fileID, err := resourceGitLabRepositoryFileParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing repository file ID: %s", err)
		}
		// branch := rs.Primary.Attributes["branch"]
		if branch == "" {
			return fmt.Errorf("No branch set")
		}
		options := &gitlab.GetFileOptions{
			Ref: gitlab.String(branch),
		}
		repoName := rs.Primary.Attributes["project"]
		if repoName == "" {
			return fmt.Errorf("No project ID set")
		}

		gotFile, _, err := testutil.TestGitlabClient.RepositoryFiles.GetFile(repoName, fileID, options)
		if err != nil {
			return fmt.Errorf("Cannot get file: %v", err)
		}

		if gotFile.FilePath == fileID {
			*file = *gotFile
			return nil
		}
		return fmt.Errorf("File does not exist")
	}
}

func testAccCheckGitlabRepositoryFileCommitMessage(n string, file *gitlab.File, message string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		_, branch, fileID, err := resourceGitLabRepositoryFileParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing repository file ID: %s", err)
		}
		if branch == "" {
			return fmt.Errorf("No branch set")
		}
		options := &gitlab.GetFileBlameOptions{
			Ref: gitlab.String(branch),
		}
		repoName := rs.Primary.Attributes["project"]
		if repoName == "" {
			return fmt.Errorf("No project ID set")
		}

		gotFile, _, err := testutil.TestGitlabClient.RepositoryFiles.GetFileBlame(repoName, fileID, options)
		if err != nil {
			return fmt.Errorf("Cannot get file: %v", err)
		}

		if len(gotFile) == 0 {
			return fmt.Errorf("No FileBlame returned")
		}

		if gotFile[0].Commit.Message == message {
			return nil
		}
		return fmt.Errorf("Commit message does not match")
	}
}

func testAccCheckGitlabRepositoryFileDeleteCommitMessage(n string, message string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		_, branch, _, err := resourceGitLabRepositoryFileParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing repository file ID: %s", err)
		}
		if branch == "" {
			return fmt.Errorf("No branch set")
		}
		options := &gitlab.ListCommitsOptions{
			RefName: gitlab.String(branch),
			All:     gitlab.Bool(true),
		}
		repoName := rs.Primary.Attributes["project"]
		if repoName == "" {
			return fmt.Errorf("No project ID set")
		}

		commits, _, err := testutil.TestGitlabClient.Commits.ListCommits(repoName, options)
		if err != nil {
			return fmt.Errorf("Cannot get commits: %v", err)
		}

		if len(commits) == 0 {
			return fmt.Errorf("No Commits returned")
		}

		if commits[len(commits)-1].Message == message {
			return nil
		}
		return fmt.Errorf("Commit message does not match")
	}
}

type testAccGitlabRepositoryFileAttributes struct {
	FilePath string
	Content  string
}

func testAccCheckGitlabRepositoryFileAttributes(got *gitlab.File, want *testAccGitlabRepositoryFileAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if got.FileName != want.FilePath {
			return fmt.Errorf("got name %q; want %q", got.FileName, want.FilePath)
		}

		if got.Content != want.Content {
			return fmt.Errorf("got content %q; want %q", got.Content, want.Content)
		}
		return nil
	}
}

func testAccCheckGitlabRepositoryFileDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project" {
			continue
		}

		gotRepo, resp, err := testutil.TestGitlabClient.Projects.GetProject(rs.Primary.ID, nil)
		if err == nil {
			if gotRepo != nil && fmt.Sprintf("%d", gotRepo.ID) == rs.Primary.ID {
				if gotRepo.MarkedForDeletionAt == nil {
					return fmt.Errorf("Repository still exists")
				}
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

func testAccGitlabRepositoryFileConfig(projectID int) string {
	return fmt.Sprintf(`
resource "gitlab_repository_file" "this" {
  project = %d
  file_path = "meow.txt"
  branch = "main"
  content = "bWVvdyBtZW93IG1lb3c="
  author_email = "meow@catnip.com"
  author_name = "Meow Meowington"
  commit_message = "feature: add launch codes"
}
	`, projectID)
}

func testAccGitlabRepositoryFileStartBranchConfig(projectID int) string {
	return fmt.Sprintf(`
resource "gitlab_repository_file" "this" {
  project = %d
  file_path = "meow.txt"
  branch = "meow-branch"
  start_branch = "main"
  content = "bWVvdyBtZW93IG1lb3c="
  author_email = "meow@catnip.com"
  author_name = "Meow Meowington"
  commit_message = "feature: add launch codes"
}
	`, projectID)
}

func testAccGitlabRepositoryFileUpdateConfig(projectID int) string {
	return fmt.Sprintf(`
resource "gitlab_repository_file" "this" {
  project = %d
  file_path = "meow.txt"
  branch = "main"
  content = "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg=="
  author_email = "meow@catnip.com"
  author_name = "Meow Meowington"
  commit_message = "feature: change launch codes"
}
	`, projectID)
}

func testAccGitlabRepositoryFileSameFileDifferentRepositoryConfig(firstProjectID, secondProjectID int) string {
	return fmt.Sprintf(`
resource "gitlab_repository_file" "foo_file" {
  project = %d
  file_path = "meow.txt"
  branch = "main"
  content = "bWVvdyBtZW93IG1lb3c="
  author_email = "meow@catnip.com"
  author_name = "Meow Meowington"
  commit_message = "feature: add launch codes"
}

resource "gitlab_repository_file" "bar_file" {
  project = %d
  file_path = "meow.txt"
  branch = "main"
  content = "bWVvdyBtZW93IG1lb3c="
  author_email = "meow@catnip.com"
  author_name = "Meow Meowington"
  commit_message = "feature: add launch codes"
}
	`, firstProjectID, secondProjectID)
}

func testAccGitlabRepositoryFileConcurrentResourcesConfig(projectID int) string {
	return fmt.Sprintf(`
resource "gitlab_repository_file" "this" {
  project = "%d"
  file_path = "file-${count.index}.txt"
  branch = "main"
  content = base64encode("content-${count.index}")
  commit_message = "Add file ${count.index}"

  count = 50
}
	`, projectID)
}

func testAccGitlabRepositoryFileConcurrentResourcesConfigUpdate(projectID int) string {
	return fmt.Sprintf(`
resource "gitlab_repository_file" "this" {
  project = "%d"
  file_path = "file-${count.index}.txt"
  branch = "main"
  content = base64encode("updated-content-${count.index}")
  commit_message = "Add file ${count.index}"

  count = 50
}
	`, projectID)
}
