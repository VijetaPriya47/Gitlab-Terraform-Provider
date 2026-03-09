package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &gitlabReleaseResource{}
	_ resource.ResourceWithConfigure   = &gitlabReleaseResource{}
	_ resource.ResourceWithImportState = &gitlabReleaseResource{}
)

func init() {
	registerResource(NewGitLabReleaseResource)
}

// NewGitLabReleaseResource is a helper function to simplify the provider implementation.
func NewGitLabReleaseResource() resource.Resource {
	return &gitlabReleaseResource{}
}

func (r *gitlabReleaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release"
}

// gitlabReleaseResource defines the resource implementation.
type gitlabReleaseResource struct {
	client *gitlab.Client
}

// gitlabReleaseResourceModel describes the resource data model
type gitlabReleaseResourceModel struct {
	ID              types.String      `tfsdk:"id"`
	Project         types.String      `tfsdk:"project"`
	Name            types.String      `tfsdk:"name"`
	TagName         types.String      `tfsdk:"tag_name"`
	TagMessage      types.String      `tfsdk:"tag_message"`
	TagPath         types.String      `tfsdk:"tag_path"`
	Description     types.String      `tfsdk:"description"`
	DescriptionHTML types.String      `tfsdk:"description_html"`
	Ref             types.String      `tfsdk:"ref"`
	Milestones      types.Set         `tfsdk:"milestones"`
	CreatedAt       types.String      `tfsdk:"created_at"`
	ReleasedAt      timetypes.RFC3339 `tfsdk:"released_at"`
	Author          types.Object      `tfsdk:"author"`
	Commit          types.Object      `tfsdk:"commit"`
	UpcomingRelease types.Bool        `tfsdk:"upcoming_release"`
	CommitPath      types.String      `tfsdk:"commit_path"`
	Assets          types.Object      `tfsdk:"assets"`
	Links           types.Object      `tfsdk:"links"`
}

type gitlabReleaseAuthor struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Username  types.String `tfsdk:"username"`
	State     types.String `tfsdk:"state"`
	AvatarURL types.String `tfsdk:"avatar_url"`
	WebURL    types.String `tfsdk:"web_url"`
}

type gitlabReleaseCommit struct {
	ID             types.String `tfsdk:"id"`
	ShortID        types.String `tfsdk:"short_id"`
	Title          types.String `tfsdk:"title"`
	CreatedAt      types.String `tfsdk:"created_at"`
	ParentIDs      types.Set    `tfsdk:"parent_ids"`
	Message        types.String `tfsdk:"message"`
	AuthorName     types.String `tfsdk:"author_name"`
	AuthorEmail    types.String `tfsdk:"author_email"`
	AuthoredDate   types.String `tfsdk:"authored_date"`
	CommitterName  types.String `tfsdk:"committer_name"`
	CommitterEmail types.String `tfsdk:"committer_email"`
	CommittedDate  types.String `tfsdk:"committed_date"`
}

type gitlabReleaseAsset struct {
	Count types.Int64 `tfsdk:"count"`
}

type gitlabReleaseLinks struct {
	ClosedIssuesURL        types.String `tfsdk:"closed_issues_url"`
	ClosedMergeRequestsURL types.String `tfsdk:"closed_merge_requests_url"`
	EditURL                types.String `tfsdk:"edit_url"`
	MergedMergeRequestsURL types.String `tfsdk:"merged_merge_requests_url"`
	OpenedIssuesURL        types.String `tfsdk:"opened_issues_url"`
	OpenedMergeRequestsURL types.String `tfsdk:"opened_merge_requests_url"`
	Self                   types.String `tfsdk:"self"`
}

func (r *gitlabReleaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_release`" + ` resource manages the lifecycle of releases in gitlab.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/releases/)`),

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project_id:tag_name>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the release.",
				Optional:            true,
				Computed:            true,
			},
			"tag_name": schema.StringAttribute{
				MarkdownDescription: "The tag where the release is created from.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"tag_message": schema.StringAttribute{
				MarkdownDescription: "Message to use if creating a new annotated tag.",
				Optional:            true,
			},
			"tag_path": schema.StringAttribute{
				MarkdownDescription: "The path to the tag.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the release. You can use Markdown.",
				Optional:            true,
				Computed:            true,
			},
			"description_html": schema.StringAttribute{
				MarkdownDescription: "HTML rendered Markdown of the release description.",
				Computed:            true,
			},
			"ref": schema.StringAttribute{
				MarkdownDescription: "If a tag specified in tag_name doesn't exist, the release is created from ref and tagged with tag_name. It can be a commit SHA, another tag name, or a branch name.",
				Optional:            true,
				Computed:            true,
			},
			"milestones": schema.SetAttribute{
				MarkdownDescription: "The title of each milestone the release is associated with. GitLab Premium customers can specify group milestones.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the release was created. In ISO 8601 format (2019-03-15T08:00:00Z).",
				Computed:            true,
			},
			"released_at": schema.StringAttribute{
				MarkdownDescription: "Date and time for the release. Defaults to the current time. Expected in ISO 8601 format (2019-03-15T08:00:00Z). Only provide this field if creating an upcoming or historical release.",
				Optional:            true,
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"author": schema.SingleNestedAttribute{
				MarkdownDescription: "The author of the release.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						MarkdownDescription: "The ID of the author's user.",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "The name of the author.",
						Computed:            true,
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "The username of the author.",
						Computed:            true,
					},
					"state": schema.StringAttribute{
						MarkdownDescription: "The state of the author's user.",
						Computed:            true,
					},
					"avatar_url": schema.StringAttribute{
						MarkdownDescription: "The url of the author's' user avatar.",
						Computed:            true,
					},
					"web_url": schema.StringAttribute{
						MarkdownDescription: "The url to the author's user profile.",
						Computed:            true,
					},
				},
			},
			"commit": schema.SingleNestedAttribute{
				MarkdownDescription: "The release commit.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "The git commit full SHA",
						Computed:            true,
					},
					"short_id": schema.StringAttribute{
						MarkdownDescription: "The git commit short SHA.",
						Computed:            true,
					},
					"title": schema.StringAttribute{
						MarkdownDescription: "The title of the commit.",
						Computed:            true,
					},
					"created_at": schema.StringAttribute{
						MarkdownDescription: "The date and time the commit was created. In ISO 8601 format (2019-03-15T08:00:00Z).",
						Computed:            true,
					},
					"parent_ids": schema.SetAttribute{
						MarkdownDescription: "The full SHA of any parent commits.",
						Computed:            true,
						ElementType:         types.StringType,
					},
					"message": schema.StringAttribute{
						MarkdownDescription: "The commit message.",
						Computed:            true,
					},
					"author_name": schema.StringAttribute{
						MarkdownDescription: "The name of the commit author.",
						Computed:            true,
					},
					"author_email": schema.StringAttribute{
						MarkdownDescription: "The email address of the commit author.",
						Computed:            true,
					},
					"authored_date": schema.StringAttribute{
						MarkdownDescription: "The date and time the commit was authored. In ISO 8601 format (2019-03-15T08:00:00Z).",
						Computed:            true,
					},
					"committer_name": schema.StringAttribute{
						MarkdownDescription: "The name of the committer.",
						Computed:            true,
					},
					"committer_email": schema.StringAttribute{
						MarkdownDescription: "The email address of the committer.",
						Computed:            true,
					},
					"committed_date": schema.StringAttribute{
						MarkdownDescription: "The date and time the commit was made. In ISO 8601 format (2019-03-15T08:00:00Z).",
						Computed:            true,
					},
				},
			},
			"upcoming_release": schema.BoolAttribute{
				MarkdownDescription: "Whether the release_at attribute is set to a future date.",
				Computed:            true,
			},
			"commit_path": schema.StringAttribute{
				MarkdownDescription: "The path to the commit",
				Computed:            true,
			},
			"assets": schema.SingleNestedAttribute{
				MarkdownDescription: "The release assets.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"count": schema.Int64Attribute{
						MarkdownDescription: "The total count of assets in this release.",
						Computed:            true,
					},
				},
			},
			"links": schema.SingleNestedAttribute{
				MarkdownDescription: "Links of the release",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"closed_issues_url": schema.StringAttribute{
						MarkdownDescription: "URL of the release's closed issues.",
						Computed:            true,
					},
					"closed_merge_requests_url": schema.StringAttribute{
						MarkdownDescription: "URL of the release's closed merge requests.",
						Computed:            true,
					},
					"edit_url": schema.StringAttribute{
						MarkdownDescription: "URL of the release's edit page.",
						Computed:            true,
					},
					"merged_merge_requests_url": schema.StringAttribute{
						MarkdownDescription: "URL of the release's merged merge requests.",
						Computed:            true,
					},
					"opened_issues_url": schema.StringAttribute{
						MarkdownDescription: "URL of the release's open issues.",
						Computed:            true,
					},
					"opened_merge_requests_url": schema.StringAttribute{
						MarkdownDescription: "URL of the release's open merge requests.",
						Computed:            true,
					},
					"self": schema.StringAttribute{
						MarkdownDescription: "URL of the release.",
						Computed:            true,
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabReleaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Create creates a new upstream resource and adds it into the Terraform state.
func (r *gitlabReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabReleaseResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Project.ValueString()

	options := &gitlab.CreateReleaseOptions{
		TagName: gitlab.Ptr(data.TagName.ValueString()),
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		options.Name = data.Name.ValueStringPointer()
	}

	if !data.TagMessage.IsNull() && !data.TagMessage.IsUnknown() {
		options.TagMessage = data.TagMessage.ValueStringPointer()
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = data.Description.ValueStringPointer()
	}

	if !data.Ref.IsNull() && !data.Ref.IsUnknown() {
		options.Ref = data.Ref.ValueStringPointer()
	}

	if !data.ReleasedAt.IsNull() && len(data.ReleasedAt.ValueString()) > 0 {
		releasedAt, diag := data.ReleasedAt.ValueRFC3339Time()
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}
		options.ReleasedAt = &releasedAt
	}

	if !data.Milestones.IsNull() && !data.Milestones.IsUnknown() {
		var milestones []string
		data.Milestones.ElementsAs(ctx, &milestones, true)
		options.Milestones = &milestones
	}

	release, _, err := r.client.Releases.CreateRelease(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create a release: %s", err.Error()))
		return
	}
	data.ID = types.StringValue(utils.BuildTwoPartID(&projectID, data.TagName.ValueStringPointer()))
	resp.Diagnostics.Append(data.releaseModelToState(release, ctx)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	// Log the creation of the resource
	tflog.Debug(ctx, "created a release", map[string]any{
		"name": data.Name.ValueString(), "id": data.ID.ValueString(),
	})
}

func (r *gitlabReleaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabReleaseResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID, tagName, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format Read",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project-id>:<tag-name>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	release, _, err := r.client.Releases.GetRelease(projectID, tagName, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get a release: %s", err.Error()))
		return
	}
	data.Project = types.StringValue(projectID)
	data.TagName = types.StringValue(tagName)
	resp.Diagnostics.Append(data.releaseModelToState(release, ctx)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabReleaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabReleaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Project.ValueString()
	tagName := data.TagName.ValueString()

	options := &gitlab.UpdateReleaseOptions{}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		options.Name = gitlab.Ptr(data.Name.ValueString())
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}
	if !data.Milestones.IsNull() && !data.Milestones.IsUnknown() {
		var milestones []string
		data.Milestones.ElementsAs(ctx, &milestones, true)
		options.Milestones = &milestones
	}
	if !data.ReleasedAt.IsNull() && !data.ReleasedAt.IsUnknown() {
		releasedAt, diag := data.ReleasedAt.ValueRFC3339Time()
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}
		options.ReleasedAt = &releasedAt
	}

	release, _, err := r.client.Releases.UpdateRelease(projectID, tagName, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update a release: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(utils.BuildTwoPartID(&projectID, &tagName))
	resp.Diagnostics.Append(data.releaseModelToState(release, ctx)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabReleaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabReleaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, tagName, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format Delete",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project-id>:<tag-name>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, _, err = r.client.Releases.DeleteRelease(projectID, tagName, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete a release: %s", err.Error()))
		return
	}
}

func (r *gitlabReleaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabReleaseResourceModel) releaseModelToState(release *gitlab.Release, ctx context.Context) diag.Diagnostics {
	r.Name = types.StringValue(release.Name)
	r.TagName = types.StringValue(release.TagName)
	r.TagPath = types.StringValue(release.TagPath)
	r.Description = types.StringValue(release.Description)
	r.DescriptionHTML = types.StringValue(release.DescriptionHTML)
	r.CreatedAt = types.StringValue(release.CreatedAt.Format(time.RFC3339))
	milestones, diag := convertMilestonesToModel(ctx, release.Milestones)
	if diag.HasError() {
		return diag
	}
	r.Milestones = milestones
	releasedAt, diag := timetypes.NewRFC3339Value(release.ReleasedAt.Format(time.RFC3339))
	if diag.HasError() {
		return diag
	}
	r.ReleasedAt = releasedAt
	author, diag := (&gitlabReleaseAuthor{
		ID:        types.Int64Value(int64(release.Author.ID)),
		Name:      types.StringValue(release.Author.Name),
		Username:  types.StringValue(release.Author.Username),
		State:     types.StringValue(release.Author.State),
		AvatarURL: types.StringValue(release.Author.AvatarURL),
		WebURL:    types.StringValue(release.Author.WebURL),
	}).toObjectType()
	r.Author = author
	if diag.HasError() {
		return diag
	}
	parentIDs, diag := types.SetValueFrom(ctx, types.StringType, release.Commit.ParentIDs)
	if diag.HasError() {
		return diag
	}
	commit, diag := (&gitlabReleaseCommit{
		ID:             types.StringValue(release.Commit.ID),
		ShortID:        types.StringValue(release.Commit.ShortID),
		Title:          types.StringValue(release.Commit.Title),
		CreatedAt:      types.StringValue(release.Commit.CreatedAt.Format(time.RFC3339)),
		ParentIDs:      parentIDs,
		AuthorName:     types.StringValue(release.Commit.AuthorName),
		AuthorEmail:    types.StringValue(release.Commit.AuthorEmail),
		AuthoredDate:   types.StringValue(release.Commit.AuthoredDate.Format(time.RFC3339)),
		CommitterName:  types.StringValue(release.Commit.CommitterName),
		CommitterEmail: types.StringValue(release.Commit.CommitterEmail),
		CommittedDate:  types.StringValue(release.Commit.CommittedDate.Format(time.RFC3339)),
		Message:        types.StringValue(release.Commit.Message),
	}).toObjectType()
	if diag.HasError() {
		return diag
	}
	r.Commit = commit
	r.UpcomingRelease = types.BoolValue(release.UpcomingRelease)
	r.CommitPath = types.StringValue(release.CommitPath)
	assets, diag := (&gitlabReleaseAsset{
		Count: types.Int64Value(int64(release.Assets.Count)),
	}).toObjectType()
	if diag.HasError() {
		return diag
	}
	r.Assets = assets
	links, diag := (&gitlabReleaseLinks{
		ClosedIssuesURL:        types.StringValue(release.Links.ClosedIssueURL),
		ClosedMergeRequestsURL: types.StringValue(release.Links.ClosedMergeRequest),
		EditURL:                types.StringValue(release.Links.EditURL),
		MergedMergeRequestsURL: types.StringValue(release.Links.MergedMergeRequest),
		OpenedIssuesURL:        types.StringValue(release.Links.OpenedIssues),
		OpenedMergeRequestsURL: types.StringValue(release.Links.OpenedMergeRequest),
		Self:                   types.StringValue(release.Links.Self),
	}).toObjectType()
	if diag.HasError() {
		return diag
	}
	r.Links = links
	return nil
}

func convertMilestonesToModel(ctx context.Context, milestones []*gitlab.ReleaseMilestone) (basetypes.SetValue, diag.Diagnostics) {
	var milestoneNames []string
	for _, milestone := range milestones {
		milestoneNames = append(milestoneNames, milestone.Title)
	}
	return types.SetValueFrom(ctx, types.StringType, milestoneNames)
}

// toObjectType converts the model struct into a `types.Object` type, making it support
// types.Unknown and types.Null values. This appears to be required since every
// attribute in the model object is Computed, causing initial computation of the state
// to mark this as `Unknown`, causing errors if we use the native model object.
func (author *gitlabReleaseAuthor) toObjectType() (types.Object, diag.Diagnostics) {
	attributeTypes := map[string]attr.Type{
		"id":         types.Int64Type,
		"name":       types.StringType,
		"username":   types.StringType,
		"state":      types.StringType,
		"avatar_url": types.StringType,
		"web_url":    types.StringType,
	}
	attributes := map[string]attr.Value{
		"id":         author.ID,
		"name":       author.Name,
		"username":   author.Username,
		"state":      author.State,
		"avatar_url": author.AvatarURL,
		"web_url":    author.WebURL,
	}
	return types.ObjectValue(attributeTypes, attributes)
}

// toObjectType converts the model struct into a `types.Object` type, making it support
// types.Unknown and types.Null values. This appears to be required since every
// attribute in the model object is Computed, causing initial computation of the state
// to mark this as `Unknown`, causing errors if we use the native model object.
func (commit *gitlabReleaseCommit) toObjectType() (types.Object, diag.Diagnostics) {
	attributeTypes := map[string]attr.Type{
		"id":              types.StringType,
		"short_id":        types.StringType,
		"title":           types.StringType,
		"created_at":      types.StringType,
		"parent_ids":      types.SetType{ElemType: types.StringType},
		"message":         types.StringType,
		"author_name":     types.StringType,
		"author_email":    types.StringType,
		"authored_date":   types.StringType,
		"committer_name":  types.StringType,
		"committer_email": types.StringType,
		"committed_date":  types.StringType,
	}
	attributes := map[string]attr.Value{
		"id":              commit.ID,
		"short_id":        commit.ShortID,
		"title":           commit.Title,
		"created_at":      commit.CreatedAt,
		"parent_ids":      commit.ParentIDs,
		"message":         commit.Message,
		"author_name":     commit.AuthorName,
		"author_email":    commit.AuthorEmail,
		"authored_date":   commit.AuthoredDate,
		"committer_name":  commit.CommitterName,
		"committer_email": commit.CommitterEmail,
		"committed_date":  commit.CommittedDate,
	}
	return types.ObjectValue(attributeTypes, attributes)
}

// toObjectType converts the model struct into a `types.Object` type, making it support
// types.Unknown and types.Null values. This appears to be required since every
// attribute in the model object is Computed, causing initial computation of the state
// to mark this as `Unknown`, causing errors if we use the native model object.
func (asset *gitlabReleaseAsset) toObjectType() (types.Object, diag.Diagnostics) {
	attributeTypes := map[string]attr.Type{
		"count": types.Int64Type,
	}
	attributes := map[string]attr.Value{
		"count": asset.Count,
	}
	return types.ObjectValue(attributeTypes, attributes)
}

// toObjectType converts the model struct into a `types.Object` type, making it support
// types.Unknown and types.Null values. This appears to be required since every
// attribute in the model object is Computed, causing initial computation of the state
// to mark this as `Unknown`, causing errors if we use the native model object.
func (links *gitlabReleaseLinks) toObjectType() (types.Object, diag.Diagnostics) {
	attributeTypes := map[string]attr.Type{
		"closed_issues_url":         types.StringType,
		"closed_merge_requests_url": types.StringType,
		"edit_url":                  types.StringType,
		"merged_merge_requests_url": types.StringType,
		"opened_issues_url":         types.StringType,
		"opened_merge_requests_url": types.StringType,
		"self":                      types.StringType,
	}
	attributes := map[string]attr.Value{
		"closed_issues_url":         links.ClosedIssuesURL,
		"closed_merge_requests_url": links.ClosedMergeRequestsURL,
		"edit_url":                  links.EditURL,
		"merged_merge_requests_url": links.MergedMergeRequestsURL,
		"opened_issues_url":         links.OpenedIssuesURL,
		"opened_merge_requests_url": links.OpenedMergeRequestsURL,
		"self":                      links.Self,
	}
	return types.ObjectValue(attributeTypes, attributes)
}
