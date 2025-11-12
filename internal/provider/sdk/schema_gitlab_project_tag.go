package sdk

import (
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func gitlabProjectTagGetSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Description: "The name of a tag.",
			Type:        schema.TypeString,
			ForceNew:    true,
			Required:    true,
		},
		"message": {
			Description: "The message of the annotated tag.",
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
		},
		"protected": {
			Description: "Bool, true if tag has tag protection.",
			Type:        schema.TypeBool,
			Computed:    true,
		},
		"target": {
			Description: "The unique id assigned to the commit by Gitlab.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"release": {
			Description: "The release associated with the tag.",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashResource(releaseNoteSchema),
			Elem:        releaseNoteSchema,
		},
		"commit": {
			Description: "The commit associated with the tag.",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashResource(commitSchema),
			Elem:        commitSchema,
		},
	}
}

var releaseNoteSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"tag_name": {
			Description: "The name of the tag.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"description": {
			Description: "The description of release.",
			Type:        schema.TypeString,
			Computed:    true,
		},
	},
}

var commitSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"id": {
			Description: "The unique id assigned to the commit by Gitlab.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"author_email": {
			Description: "The email of the author.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"author_name": {
			Description: "The name of the author.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"authored_date": {
			Description: "The date which the commit was authored (format: yyyy-MM-ddTHH:mm:ssZ).",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"committed_date": {
			Description: "The date at which the commit was pushed (format: yyyy-MM-ddTHH:mm:ssZ).",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"committer_email": {
			Description: "The email of the user that committed.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"committer_name": {
			Description: "The name of the user that committed.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"short_id": {
			Description: "The short id assigned to the commit by Gitlab.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"title": {
			Description: "The title of the commit",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"message": {
			Description: "The commit message",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"parent_ids": {
			Description: "The id of the parents of the commit",
			Type:        schema.TypeSet,
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Set:         schema.HashString,
		},
	},
}

func flattenCommit(commit *gitlab.Commit) (values []map[string]any) {
	if commit == nil {
		return []map[string]any{}
	}
	return []map[string]any{
		{
			"id":              commit.ID,
			"short_id":        commit.ShortID,
			"title":           commit.Title,
			"author_name":     commit.AuthorName,
			"author_email":    commit.AuthorEmail,
			"authored_date":   commit.AuthoredDate.Format(time.RFC3339),
			"committed_date":  commit.CommittedDate.Format(time.RFC3339),
			"committer_email": commit.CommitterEmail,
			"committer_name":  commit.CommitterName,
			"message":         commit.Message,
			"parent_ids":      commit.ParentIDs,
		},
	}
}
