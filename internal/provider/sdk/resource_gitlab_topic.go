package sdk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var _ = registerResource("gitlab_topic", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_topic`" + ` resource allows to manage the lifecycle of topics that are then assignable to projects.

-> Topics are the successors for project tags. Aside from avoiding terminology collisions with Git tags, they are more descriptive and better searchable.

**Upstream API**: [GitLab REST API docs for topics](https://docs.gitlab.com/api/topics/)
`,

		CreateContext: resourceGitlabTopicCreate,
		ReadContext:   resourceGitlabTopicRead,
		UpdateContext: resourceGitlabTopicUpdate,
		DeleteContext: resourceGitlabTopicDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: constructSchema(map[string]*schema.Schema{
			"name": {
				Description: "The topic's name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"title": {
				Description: "The topic's description.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description: "A text describing the topic.",
				Type:        schema.TypeString,
				Optional:    true,
			},
		}, avatarableSchema()),
		CustomizeDiff: avatarableDiff,
	}
})

func resourceGitlabTopicCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	options := &gitlab.CreateTopicOptions{
		Name:  gitlab.Ptr(d.Get("name").(string)),
		Title: gitlab.Ptr(d.Get("title").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		options.Description = gitlab.Ptr(v.(string))
	}

	avatar, err := handleAvatarOnCreate(d)
	if err != nil {
		return diag.FromErr(err)
	}
	if avatar != nil {
		options.Avatar = &gitlab.TopicAvatar{
			Filename: avatar.Filename,
			Image:    avatar.Image,
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab topic %s", *options.Name))

	topic, _, err := client.Topics.CreateTopic(options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to create topic %q: %s", *options.Name, err)
	}

	d.SetId(fmt.Sprintf("%d", topic.ID))
	return resourceGitlabTopicRead(ctx, d, meta)
}

func resourceGitlabTopicRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	topicID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.Errorf("Failed to convert topic id %s to int: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab topic %d", topicID))

	topic, _, err := client.Topics.GetTopic(topicID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab group %s not found so removing from state", d.Id()))
			d.SetId("")
			return nil
		}
		return diag.Errorf("Failed to read topic %d: %s", topicID, err)
	}

	d.SetId(fmt.Sprintf("%d", topic.ID))
	d.Set("name", topic.Name)
	d.Set("title", topic.Title)
	d.Set("description", topic.Description)
	d.Set("avatar_url", topic.AvatarURL)
	return nil
}

func resourceGitlabTopicUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	options := &gitlab.UpdateTopicOptions{}

	if d.HasChange("name") {
		options.Name = gitlab.Ptr(d.Get("name").(string))
	}

	if d.HasChange("title") {
		options.Title = gitlab.Ptr(d.Get("title").(string))
	}

	if d.HasChange("description") {
		options.Description = gitlab.Ptr(d.Get("description").(string))
	}

	avatar, err := handleAvatarOnUpdate(d)
	if err != nil {
		return diag.FromErr(err)
	}
	if avatar != nil {
		options.Avatar = &gitlab.TopicAvatar{
			Filename: avatar.Filename,
			Image:    avatar.Image,
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab topic %s", d.Id()))

	topicID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.Errorf("Failed to convert topic id %s to int: %s", d.Id(), err)
	}
	_, _, err = client.Topics.UpdateTopic(topicID, options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to update topic %d: %s", topicID, err)
	}
	return resourceGitlabTopicRead(ctx, d, meta)
}

func resourceGitlabTopicDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	topicID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.Errorf("Failed to convert topic id %s to int: %s", d.Id(), err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] delete gitlab topic %s", d.Id()))

	if _, err = client.Topics.DeleteTopic(topicID, gitlab.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to delete topic %d: %s", topicID, err)
	}

	return nil
}
