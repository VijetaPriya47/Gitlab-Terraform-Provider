package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectTagResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectTagResource{}
	_ resource.ResourceWithImportState = &gitlabProjectTagResource{}
)

func init() {
	registerResource(NewGitlabProjectTagResource)
}

func NewGitlabProjectTagResource() resource.Resource {
	return &gitlabProjectTagResource{}
}

type gitlabProjectTagResource struct {
	client *gitlab.Client
}

type gitlabProjectTagResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Project   types.String `tfsdk:"project"`
	Ref       types.String `tfsdk:"ref"`
	Message   types.String `tfsdk:"message"`
	Protected types.Bool   `tfsdk:"protected"`
	Target    types.String `tfsdk:"target"`
	Release   types.Set    `tfsdk:"release"`
	Commit    types.Set    `tfsdk:"commit"`
}

type gitlabProjectTagReleaseResourceModel struct {
	TagName     types.String `tfsdk:"tag_name"`
	Description types.String `tfsdk:"description"`
}

type gitlabProjectTagCommitResourceModel struct {
	ID             types.String `tfsdk:"id"`
	AuthorEmail    types.String `tfsdk:"author_email"`
	AuthorName     types.String `tfsdk:"author_name"`
	AuthoredDate   types.String `tfsdk:"authored_date"`
	CommittedDate  types.String `tfsdk:"committed_date"`
	CommitterEmail types.String `tfsdk:"committer_email"`
	CommitterName  types.String `tfsdk:"committer_name"`
	ShortID        types.String `tfsdk:"short_id"`
	Title          types.String `tfsdk:"title"`
	Message        types.String `tfsdk:"message"`
	ParentIDs      types.Set    `tfsdk:"parent_ids"`
}

func (r *gitlabProjectTagResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_tag"
}

func (r *gitlabProjectTagResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_tag`" + ` resource allows users to manage the lifecycle of a tag in a project.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/tags/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource. In the format `<project>:<name>`",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of a tag.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project owned by the authenticated user.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ref": schema.StringAttribute{
				MarkdownDescription: "Create tag using commit SHA, another tag name, or branch name. This attribute is not available for imported resources.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"message": schema.StringAttribute{
				MarkdownDescription: "The message of the annotated tag.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
			"protected": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if tag has tag protection.",
				Computed:            true,
			},
			"target": schema.StringAttribute{
				MarkdownDescription: "The unique id assigned to the commit by Gitlab.",
				Computed:            true,
			},
			"release": schema.SetNestedAttribute{
				MarkdownDescription: "The release associated with the tag.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"tag_name": schema.StringAttribute{
							MarkdownDescription: "The name of the tag.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "The description of release.",
							Computed:            true,
						},
					},
				},
			},
			"commit": r.commitSchema(),
		},
	}
}

func (r *gitlabProjectTagResource) commitSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		MarkdownDescription: "The commit associated with the tag.",
		Computed:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "The unique id assigned to the commit by Gitlab.",
					Computed:            true,
				},
				"author_email": schema.StringAttribute{
					MarkdownDescription: "The email of the author.",
					Computed:            true,
				},
				"author_name": schema.StringAttribute{
					MarkdownDescription: "The name of the author.",
					Computed:            true,
				},
				"authored_date": schema.StringAttribute{
					MarkdownDescription: "The date which the commit was authored (format: yyyy-MM-ddTHH:mm:ssZ).",
					Computed:            true,
				},
				"committed_date": schema.StringAttribute{
					MarkdownDescription: "The date at which the commit was pushed (format: yyyy-MM-ddTHH:mm:ssZ).",
					Computed:            true,
				},
				"committer_email": schema.StringAttribute{
					MarkdownDescription: "The email of the user that committed.",
					Computed:            true,
				},
				"committer_name": schema.StringAttribute{
					MarkdownDescription: "The name of the user that committed.",
					Computed:            true,
				},
				"short_id": schema.StringAttribute{
					MarkdownDescription: "The short id assigned to the commit by Gitlab.",
					Computed:            true,
				},
				"title": schema.StringAttribute{
					MarkdownDescription: "The title of the commit",
					Computed:            true,
				},
				"message": schema.StringAttribute{
					MarkdownDescription: "The commit message",
					Computed:            true,
				},
				"parent_ids": schema.SetAttribute{
					MarkdownDescription: "The id of the parents of the commit",
					Computed:            true,
					ElementType:         types.StringType,
				},
			},
		},
	}
}

func (r *gitlabProjectTagResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectTagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	project := data.Project.ValueString()
	ref := data.Ref.ValueString()
	message := data.Message.ValueString()

	tagOptions := &gitlab.CreateTagOptions{
		TagName: &name,
		Ref:     &ref,
		Message: &message,
	}

	tflog.Debug(ctx, "create gitlab tag", map[string]any{
		"project": project,
		"name":    name,
		"ref":     ref,
	})
	tag, _, err := r.client.Tags.CreateTag(project, tagOptions, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create tag: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &name))
	resp.Diagnostics.Append(data.modelToStateModel(ctx, project, tag)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectTagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, name, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Read. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, "read gitlab tag", map[string]any{
		"project": project,
		"name":    name,
	})
	tag, _, err := r.client.Tags.GetTag(project, name, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "received 404 for gitlab tag, removing from state", map[string]any{
				"project": project,
				"name":    name,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read tag: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(data.modelToStateModel(ctx, project, tag)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Tags are immutable in GitLab API - all attributes use RequiresReplace
	// This function should never be called, but is required by the interface
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place update which is not possible.")
}

func (r *gitlabProjectTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectTagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, name, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Delete. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, "delete gitlab tag", map[string]any{
		"project": project,
		"name":    name,
	})
	_, err = r.client.Tags.DeleteTag(project, name, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete tag: %s", err.Error()))
		return
	}
}

func (d *gitlabProjectTagResourceModel) modelToStateModel(ctx context.Context, project string, tag *gitlab.Tag) diag.Diagnostics {
	d.Name = types.StringValue(tag.Name)
	d.Project = types.StringValue(project)
	d.Message = types.StringValue(tag.Message)
	d.Protected = types.BoolValue(tag.Protected)
	d.Target = types.StringValue(tag.Target)

	// Handle Release nested object
	if tag.Release == nil {
		d.Release = types.SetNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"tag_name":    types.StringType,
				"description": types.StringType,
			},
		})
	} else {
		modelRelease := gitlabProjectTagReleaseResourceModel{
			TagName:     types.StringValue(tag.Release.TagName),
			Description: types.StringValue(tag.Release.Description),
		}
		releaseSet, diags := types.SetValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"tag_name":    types.StringType,
				"description": types.StringType,
			},
		}, []gitlabProjectTagReleaseResourceModel{modelRelease})
		if diags.HasError() {
			return diags
		}
		d.Release = releaseSet
	}

	// Handle Commit nested object
	if tag.Commit == nil {
		d.Commit = types.SetNull(commitSchema().NestedObject.Type())
	} else {
		parentIDs, diags := types.SetValueFrom(ctx, types.StringType, tag.Commit.ParentIDs)
		if diags.HasError() {
			return diags
		}
		modelCommit := gitlabProjectTagCommitResourceModel{
			ID:             types.StringValue(tag.Commit.ID),
			AuthorEmail:    types.StringValue(tag.Commit.AuthorEmail),
			AuthorName:     types.StringValue(tag.Commit.AuthorName),
			AuthoredDate:   types.StringValue(tag.Commit.AuthoredDate.Format(time.RFC3339)),
			CommittedDate:  types.StringValue(tag.Commit.CommittedDate.Format(time.RFC3339)),
			CommitterEmail: types.StringValue(tag.Commit.CommitterEmail),
			CommitterName:  types.StringValue(tag.Commit.CommitterName),
			ShortID:        types.StringValue(tag.Commit.ShortID),
			Title:          types.StringValue(tag.Commit.Title),
			Message:        types.StringValue(tag.Commit.Message),
			ParentIDs:      parentIDs,
		}
		commitSet, diags := types.SetValueFrom(ctx, commitSchema().NestedObject.Type(), []gitlabProjectTagCommitResourceModel{modelCommit})
		if diags.HasError() {
			return diags
		}
		d.Commit = commitSet
	}

	return nil
}
