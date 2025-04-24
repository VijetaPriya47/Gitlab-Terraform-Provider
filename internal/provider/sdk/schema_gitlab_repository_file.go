package sdk

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"gitlab.com/gitlab-org/api/client-go"
)

func gitlabRepositoryFileGetSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"project": {
			Description: "The name or ID of the project.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"file_path": {
			Description: "The full path of the file. It must be relative to the root of the project without a leading slash `/` or `./`.",
			Type:        schema.TypeString,
			// The regex here is checking for "/" OR "./", but looks funny due to needed escaping since both "/" and "." are regex special characters.
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringDoesNotMatch(regexp.MustCompile(`^\/|^\.\/`), "`file_path` cannot start with a `/` or `./`. See https://gitlab.com/gitlab-org/gitlab/-/issues/363112 for more information.")),
			Required:         true,
			ForceNew:         true,
		},
		"content": {
			Description: "File content.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"ref": {
			Description: "The name of branch, tag or commit.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"file_name": {
			Description: "The filename.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"size": {
			Description: "The file size.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		"content_sha256": {
			Description: "File content sha256 digest.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"execute_filemode": {
			Description: "Enables or disables the execute flag on the file.",
			Type:        schema.TypeBool,
			Optional:    true,
		},
		"overwrite_on_create": {
			Description: "Enable overwriting existing files, defaults to `false`. This attribute is only used during `create` and must be use carefully. We suggest to use `imports` whenever possible and limit the use of this attribute for when the project was imported on the same `apply`. This attribute is not supported during a resource import.",
			Type:        schema.TypeBool,
			Optional:    true,
		},
		"blob_id": {
			Description: "The blob id.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"commit_id": {
			Description: "The commit id.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"last_commit_id": {
			Description: "The last known commit id.",
			Type:        schema.TypeString,
			Computed:    true,
		},
	}
}

func gitlabRepositoryFileToStateMap(project string, repositoryFile *gitlab.File) map[string]any {
	stateMap := make(map[string]any)
	stateMap["project"] = project
	stateMap["file_name"] = repositoryFile.FileName
	stateMap["file_path"] = repositoryFile.FilePath
	stateMap["size"] = repositoryFile.Size
	stateMap["encoding"] = repositoryFile.Encoding
	stateMap["content"] = repositoryFile.Content
	stateMap["content_sha256"] = repositoryFile.SHA256
	stateMap["execute_filemode"] = repositoryFile.ExecuteFilemode
	stateMap["ref"] = repositoryFile.Ref
	stateMap["blob_id"] = repositoryFile.BlobID
	stateMap["commit_id"] = repositoryFile.CommitID
	stateMap["last_commit_id"] = repositoryFile.LastCommitID
	return stateMap
}
