package sdk

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// NOTE: this lock is a bit of a hack to prevent parallel calls to the GitLab Repository Files API.
//
//	If it is called concurrently, the API will return a 400 error along the lines of:
//	```
//	(400 Bad Request) DELETE https://gitlab.com/api/v4/projects/30716/repository/files/somefile.yaml: 400
//	{message: 9:Could not update refs/heads/master. Please refresh and try again..}
//	```
//
//	This lock only solves half of the problem, where the provider is responsible for
//	the concurrency. The other half is if the API is called outside of terraform at the same time
//	this resource makes calls to the API.
//	To mitigate this, simple retries are used.
var resourceGitlabRepositoryFileApiLock = newLock()

var validEncodingValues = []string{
	"base64",
	"text",
}

var _ = registerResource("gitlab_repository_file", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_repository_file`" + ` resource manages the lifecycle of a file within a repository.

-> **Timeouts** Default timeout for *Create*, *Update* and *Delete* is one minute and can be configured in the ` + "`timeouts`" + ` block.

-> **Implementation Detail** GitLab is unable to handle concurrent calls to the GitLab repository files API for the same project.
   Therefore, this resource queues every call to the repository files API no matter of the project, which may slow down the terraform
   execution time for some configurations. In addition, retries are performed in case a refresh is required because another application
   changed the repository at the same time.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/repository_files/)`,

		CreateContext: resourceGitlabRepositoryFileCreate,
		ReadContext:   resourceGitlabRepositoryFileRead,
		UpdateContext: resourceGitlabRepositoryFileUpdate,
		DeleteContext: resourceGitlabRepositoryFileDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		// the schema matches https://docs.gitlab.com/api/repository_files/#create-new-file-in-repository
		// However, we don't support the `encoding` parameter as it seems to be broken.
		// Only a value of `base64` is supported, all others, including the documented default `text`, lead to
		// a `400 {error: encoding does not have a valid value}` error.
		Schema: constructSchema(
			map[string]*schema.Schema{
				"branch": {
					Description: "Name of the branch to which to commit to.",
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
				},
				"commit_message": {
					Description:   "Commit message.",
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"create_commit_message", "update_commit_message", "delete_commit_message"},
				},
				"create_commit_message": {
					Description:   "Create commit message.",
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"commit_message"},
					RequiredWith:  []string{"update_commit_message", "delete_commit_message"},
				},
				"update_commit_message": {
					Description:   "Update commit message.",
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"commit_message"},
					RequiredWith:  []string{"create_commit_message", "delete_commit_message"},
				},
				"delete_commit_message": {
					Description:   "Delete Commit message.",
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"commit_message"},
					RequiredWith:  []string{"create_commit_message", "update_commit_message"},
				},
				"start_branch": {
					Description: "Name of the branch to start the new commit from.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"author_email": {
					Description: "Email of the commit author.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"author_name": {
					Description: "Name of the commit author.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"encoding": {
					Description:  fmt.Sprintf("The file content encoding. Valid values are: %s.", utils.RenderValueListForDocs(validEncodingValues)),
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringInSlice(validEncodingValues, false),
				},
			},
			gitlabRepositoryFileGetSchema(),
		),
	}
})

func resourceGitlabRepositoryFileCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	project := d.Get("project").(string)
	filePath := d.Get("file_path").(string)

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab_repository_file: waiting for lock to create %s/%s", project, filePath))
	if err := resourceGitlabRepositoryFileApiLock.lock(ctx); err != nil {
		return diag.FromErr(err)
	}
	defer resourceGitlabRepositoryFileApiLock.unlock()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab_repository_file: got lock to create %s/%s", project, filePath))

	client := meta.(*gitlab.Client)
	content := d.Get("content").(string)

	options := &gitlab.CreateFileOptions{
		Branch:      gitlab.Ptr(d.Get("branch").(string)),
		AuthorEmail: gitlab.Ptr(d.Get("author_email").(string)),
		AuthorName:  gitlab.Ptr(d.Get("author_name").(string)),
		Encoding:    gitlab.Ptr(d.Get("encoding").(string)),
		Content:     gitlab.Ptr(content),
	}
	if startBranch, ok := d.GetOk("start_branch"); ok {
		options.StartBranch = gitlab.Ptr(startBranch.(string))
	}
	if executeFilemode, ok := d.GetOk("execute_filemode"); ok {
		options.ExecuteFilemode = gitlab.Ptr(executeFilemode.(bool))
	}
	if commitMessage, ok := d.GetOk("commit_message"); ok {
		options.CommitMessage = gitlab.Ptr(commitMessage.(string))
	} else {
		options.CommitMessage = gitlab.Ptr(d.Get("create_commit_message").(string))
	}

	if overwriteOnCreate, ok := d.GetOk("overwrite_on_create"); ok && overwriteOnCreate.(bool) {
		readOptions := &gitlab.GetFileOptions{
			Ref: gitlab.Ptr(*options.Branch),
		}
		var existingRepositoryFile *gitlab.File
		// Try to get the file to check if it exists
		err := retry.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate), func() *retry.RetryError {
			var err error
			existingRepositoryFile, _, err = client.RepositoryFiles.GetFile(project, filePath, readOptions, gitlab.WithContext(ctx))
			if err != nil {
				if isRefreshError(err) {
					return retry.RetryableError(err)
				}
				if !api.Is404(err) {
					return retry.NonRetryableError(err)
				}
			}

			if existingRepositoryFile != nil {
				tflog.Debug(ctx, fmt.Sprintf("[DEBUG] %s already exists and overwrite_on_create is true. File will be overwritten.", filePath))

				updateOptions := &gitlab.UpdateFileOptions{
					Branch:      gitlab.Ptr(*options.Branch),
					AuthorEmail: gitlab.Ptr(d.Get("author_email").(string)),
					AuthorName:  gitlab.Ptr(d.Get("author_name").(string)),
					Encoding:    gitlab.Ptr(d.Get("encoding").(string)),
					Content:     gitlab.Ptr(content),
				}

				if startBranch, ok := d.GetOk("start_branch"); ok {
					updateOptions.StartBranch = gitlab.Ptr(startBranch.(string))
				}
				if executeFilemode, ok := d.GetOk("execute_filemode"); ok {
					updateOptions.ExecuteFilemode = gitlab.Ptr(executeFilemode.(bool))
				}
				if commitMessage, ok := d.GetOk("commit_message"); ok {
					updateOptions.CommitMessage = gitlab.Ptr(commitMessage.(string))
				} else {
					updateOptions.CommitMessage = gitlab.Ptr(d.Get("update_commit_message").(string))
				}

				updateOptions.LastCommitID = gitlab.Ptr(existingRepositoryFile.LastCommitID)
				_, _, err := client.RepositoryFiles.UpdateFile(project, filePath, updateOptions, gitlab.WithContext(ctx))
				if err != nil {
					if isRefreshError(err) {
						return retry.RetryableError(err)
					}
					return retry.NonRetryableError(err)
				}

			}
			return nil
		})
		if err != nil {
			return diag.FromErr(err)
		}

		if existingRepositoryFile != nil {
			d.SetId(resourceGitLabRepositoryFileBuildId(project, *options.Branch, existingRepositoryFile.FilePath))
			return resourceGitlabRepositoryFileRead(ctx, d, meta)
		}
	}

	err := retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
		repositoryFile, _, err := client.RepositoryFiles.CreateFile(project, filePath, options, gitlab.WithContext(ctx))
		if err != nil {
			if isRefreshError(err) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		d.SetId(resourceGitLabRepositoryFileBuildId(project, repositoryFile.Branch, repositoryFile.FilePath))
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabRepositoryFileRead(ctx, d, meta)
}

func resourceGitlabRepositoryFileRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, branch, filePath, err := resourceGitLabRepositoryFileParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.GetFileOptions{
		Ref: gitlab.Ptr(branch),
	}

	repositoryFile, _, err := client.RepositoryFiles.GetFile(project, filePath, options, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] file %s not found, removing from state", filePath))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	configContent := d.Get("content").(string)
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab_repository_file: comparing content of %s with %s", repositoryFile.Content, configContent))

	// check what our encoding is to determine if we need to decode the content for checking.
	configEncoding := d.Get("encoding").(string)

	// If we are storing the value in plaintext, we need to decode the response from the API to store in the config
	if configEncoding == "text" {
		if decodedContent, err := base64.StdEncoding.DecodeString(repositoryFile.Content); err == nil {
			repositoryFile.Content = string(decodedContent)
			repositoryFile.Encoding = "text"
		}
	}

	d.SetId(resourceGitLabRepositoryFileBuildId(project, branch, repositoryFile.FilePath))
	d.Set("branch", repositoryFile.Ref)
	stateMap := gitlabRepositoryFileToStateMap(project, repositoryFile)
	if err = setStateMapInResourceData(stateMap, d); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitlabRepositoryFileUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	project, branch, filePath, err := resourceGitLabRepositoryFileParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab_repository_file: waiting for lock to update %s/%s", project, filePath))
	if err := resourceGitlabRepositoryFileApiLock.lock(ctx); err != nil {
		return diag.FromErr(err)
	}
	defer resourceGitlabRepositoryFileApiLock.unlock()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab_repository_file: got lock to update %s/%s", project, filePath))

	client := meta.(*gitlab.Client)

	readOptions := &gitlab.GetFileOptions{
		Ref: gitlab.Ptr(branch),
	}

	content := d.Get("content").(string)
	updateOptions := &gitlab.UpdateFileOptions{
		Branch:      gitlab.Ptr(branch),
		AuthorEmail: gitlab.Ptr(d.Get("author_email").(string)),
		AuthorName:  gitlab.Ptr(d.Get("author_name").(string)),
		Encoding:    gitlab.Ptr(d.Get("encoding").(string)),
		Content:     gitlab.Ptr(content),
	}

	if startBranch, ok := d.GetOk("start_branch"); ok {
		updateOptions.StartBranch = gitlab.Ptr(startBranch.(string))
	}
	if executeFilemode, ok := d.GetOk("execute_filemode"); ok {
		updateOptions.ExecuteFilemode = gitlab.Ptr(executeFilemode.(bool))
	}
	if commitMessage, ok := d.GetOk("commit_message"); ok {
		updateOptions.CommitMessage = gitlab.Ptr(commitMessage.(string))
	} else {
		updateOptions.CommitMessage = gitlab.Ptr(d.Get("update_commit_message").(string))
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate), func() *retry.RetryError {
		// NOTE: we also re-read the file to obtain an eventually changed `LastCommitID` for which we needed the refresh
		existingRepositoryFile, _, err := client.RepositoryFiles.GetFile(project, filePath, readOptions, gitlab.WithContext(ctx))
		if err != nil {
			return retry.NonRetryableError(err)
		}

		updateOptions.LastCommitID = gitlab.Ptr(existingRepositoryFile.LastCommitID)
		_, _, err = client.RepositoryFiles.UpdateFile(project, filePath, updateOptions, gitlab.WithContext(ctx))
		if err != nil {
			if isRefreshError(err) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabRepositoryFileRead(ctx, d, meta)
}

func resourceGitlabRepositoryFileDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	project, branch, filePath, err := resourceGitLabRepositoryFileParseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab_repository_file: waiting for lock to delete %s/%s", project, filePath))
	if err := resourceGitlabRepositoryFileApiLock.lock(ctx); err != nil {
		return diag.FromErr(err)
	}
	defer resourceGitlabRepositoryFileApiLock.unlock()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab_repository_file: got lock to delete %s/%s", project, filePath))

	client := meta.(*gitlab.Client)

	readOptions := &gitlab.GetFileOptions{
		Ref: gitlab.Ptr(branch),
	}
	deleteOptions := &gitlab.DeleteFileOptions{
		Branch:      gitlab.Ptr(d.Get("branch").(string)),
		AuthorEmail: gitlab.Ptr(d.Get("author_email").(string)),
		AuthorName:  gitlab.Ptr(d.Get("author_name").(string)),
	}
	if commitMessage, ok := d.GetOk("commit_message"); ok {
		deleteOptions.CommitMessage = gitlab.Ptr(fmt.Sprintf("[DELETE]: %s", commitMessage.(string)))
	} else {
		deleteOptions.CommitMessage = gitlab.Ptr(d.Get("delete_commit_message").(string))
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		// NOTE: we also re-read the file to obtain an eventually changed `LastCommitID` for which we needed the refresh

		existingRepositoryFile, _, err := client.RepositoryFiles.GetFile(project, filePath, readOptions, gitlab.WithContext(ctx))
		if err != nil {
			return retry.NonRetryableError(err)
		}

		deleteOptions.LastCommitID = gitlab.Ptr(existingRepositoryFile.LastCommitID)
		resp, err := client.RepositoryFiles.DeleteFile(project, filePath, deleteOptions, gitlab.WithContext(ctx))
		if err != nil {
			if isRefreshError(err) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(fmt.Errorf("%s failed to delete repository file: (%s) %v", d.Id(), resp.Status, err))
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitLabRepositoryFileParseId(id string) (string, string, string, error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected ID format (%q). Expected project:branch:repository_file_path", id)
	}

	return parts[0], parts[1], parts[2], nil
}

func resourceGitLabRepositoryFileBuildId(project string, branch string, filePath string) string {
	return fmt.Sprintf("%s:%s:%s", project, branch, filePath)
}

func isRefreshError(err error) bool {
	var httpErr *gitlab.ErrorResponse
	return errors.As(err, &httpErr) &&
		httpErr.Response.StatusCode == http.StatusBadRequest &&
		strings.Contains(httpErr.Message, "Please refresh and try again")
}
