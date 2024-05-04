package sdk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var _ = registerResource("gitlab_topic", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_topic`" + ` resource allows to manage the lifecycle of topics that are then assignable to projects.

-> Topics are the successors for project tags. Aside from avoiding terminology collisions with Git tags, they are more descriptive and better searchable.

~> Deleting a topic was implemented in GitLab 14.9. For older versions of GitLab set ` + "`soft_destroy = true`" + ` to empty out a topic instead of deleting it.

**Upstream API**: [GitLab REST API docs for topics](https://docs.gitlab.com/ee/api/topics.html)
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
				Description: "The topic's description. Requires at least GitLab 15.0 for which it's a required argument.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"soft_destroy": {
				Description: "Empty the topics fields instead of deleting it.",
				Type:        schema.TypeBool,
				Optional:    true,
				Deprecated:  "GitLab 14.9 introduced the proper deletion of topics. This field is no longer needed.",
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

func resourceGitlabTopicCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	if err := resourceGitlabTopicEnsureTitleSupport(ctx, client, d); err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.CreateTopicOptions{
		Name: gitlab.Ptr(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("title"); ok {
		options.Title = gitlab.Ptr(v.(string))
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

func resourceGitlabTopicRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

func resourceGitlabTopicUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	options := &gitlab.UpdateTopicOptions{}
	if err := resourceGitlabTopicEnsureTitleSupport(ctx, client, d); err != nil {
		return diag.FromErr(err)
	}

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

func resourceGitlabTopicDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	topicID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.Errorf("Failed to convert topic id %s to int: %s", d.Id(), err)
	}
	softDestroy := d.Get("soft_destroy").(bool)

	deleteNotSupported, err := api.IsGitLabVersionLessThan(ctx, client, "14.9")()
	if err != nil {
		return diag.FromErr(err)
	}
	if !softDestroy && deleteNotSupported {
		return diag.Errorf("GitLab 14.9 introduced the proper deletion of topics. Set `soft_destroy = true` to empty out a topic instead of deleting it.")
	}

	// NOTE: the `soft_destroy` field is deprecated and will be removed in a future version.
	//       It was only introduced because GitLab prior to 14.9 didn't support topic deletion.
	if softDestroy {
		tflog.Warn(ctx, fmt.Sprintf("[WARN] Not deleting gitlab topic %s. Instead emptying its description", d.Id()))

		options := &gitlab.UpdateTopicOptions{
			Description: gitlab.Ptr(""),
		}

		_, _, err = client.Topics.UpdateTopic(topicID, options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.Errorf("Failed to update topic %d: %s", topicID, err)
		}

		return nil
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] delete gitlab topic %s", d.Id()))

	if _, err = client.Topics.DeleteTopic(topicID, gitlab.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to delete topic %d: %s", topicID, err)
	}

	return nil
}

func resourceGitlabTopicEnsureTitleSupport(ctx context.Context, client *gitlab.Client, d *schema.ResourceData) error {
	isTitleSupported, err := api.IsGitLabVersionAtLeast(ctx, client, "15.0")()
	if err != nil {
		return err
	}

	if _, ok := d.GetOk("title"); isTitleSupported && !ok {
		return fmt.Errorf("title is a required attribute for GitLab 15.0 and newer. Please specify it in the configuration")
	} else if !isTitleSupported && ok {
		return fmt.Errorf("title is not supported by your version of GitLab. At least GitLab 15.0 is required")
	}

	return nil
}
