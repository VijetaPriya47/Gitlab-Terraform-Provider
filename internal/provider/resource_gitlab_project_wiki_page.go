package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabWikiPageResource{}
	_ resource.ResourceWithConfigure   = &gitlabWikiPageResource{}
	_ resource.ResourceWithImportState = &gitlabWikiPageResource{}
)

func init() {
	registerResource(NewGitLabWikiPageResource)
}

// NewGitLabWikiPageResource is a helper function to create a new gitlabWikiPageResource instance.
func NewGitLabWikiPageResource() resource.Resource {
	return &gitlabWikiPageResource{}
}

type gitlabWikiPageResource struct {
	client *gitlab.Client
}

// Metadata sets the resource type name.
func (r *gitlabWikiPageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_wiki_page"
}

// gitlabWikiPageResourceModel represents the schema for the wiki page.
type gitlabWikiPageResourceModel struct {
	Id       types.String `tfsdk:"id"`
	Project  types.String `tfsdk:"project"`
	Slug     types.String `tfsdk:"slug"`
	Title    types.String `tfsdk:"title"`
	Content  types.String `tfsdk:"content"`
	Format   types.String `tfsdk:"format"`
	Encoding types.String `tfsdk:"encoding"`
}

// Schema defines the structure of the resource.
func (r *gitlabWikiPageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Valid values for the schema
	var validFormats = []string{"markdown", "rdoc", "asciidoc", "org"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_wiki_page`" + ` resource allows managing the lifecycle of a project wiki page.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/wikis/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Title of the wiki page.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[^:]*$`),
						"Title must not contain ':' as it generates an invalid slug",
					),
				},
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "Content of the wiki page. Must be at least 1 character long.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"format": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf(
					"Format of the wiki page (auto-generated if not provided). Valid values are: %s.",
					utils.RenderValueListForDocs(validFormats),
				),
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf(validFormats...),
				},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "Slug of the wiki page.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"encoding": schema.StringAttribute{
				MarkdownDescription: "The encoding used for the wiki page content.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure initializes the GitLab client for the resource.
func (r *gitlabWikiPageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Create creates a new GitLab wiki page.
func (r *gitlabWikiPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabWikiPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Project.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab project wiki page for project %q", projectID))

	options := &gitlab.CreateWikiPageOptions{
		Title:   gitlab.Ptr(data.Title.ValueString()),
		Content: gitlab.Ptr(data.Content.ValueString()),
		Format:  gitlab.Ptr(gitlab.WikiFormatValue(data.Format.ValueString())),
	}

	createdPage, _, err := r.client.Wikis.CreateWikiPage(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API Error", fmt.Sprintf("Failed to create wiki page: %s", err.Error()))
		return
	}

	data.Id = types.StringValue(utils.BuildTwoPartID(&projectID, &createdPage.Slug))
	data.Slug = types.StringValue(createdPage.Slug)
	data.Project = types.StringValue(projectID)
	data.Format = types.StringValue(string(createdPage.Format))
	data.Content = types.StringValue(createdPage.Content)
	data.Title = types.StringValue(createdPage.Title)
	data.Encoding = types.StringValue(createdPage.Encoding)

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read retrieves the latest state of the GitLab wiki page.
func (r *gitlabWikiPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabWikiPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	//`slug` (Required): The unique identifier for the wiki page. It must not contain the `:` character!
	projectID, slug, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project-id>:<slug>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	wikiPage, _, err := r.client.Wikis.GetWikiPage(projectID, slug, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API Error", fmt.Sprintf("Failed to read wiki page: %s", err.Error()))
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(utils.BuildTwoPartID(&projectID, &wikiPage.Slug))
	data.Project = types.StringValue(projectID)
	data.Slug = types.StringValue(wikiPage.Slug)
	data.Title = types.StringValue(wikiPage.Title)
	data.Content = types.StringValue(wikiPage.Content)
	data.Format = types.StringValue(string(wikiPage.Format))
	data.Encoding = types.StringValue(wikiPage.Encoding)

	// Save the updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the GitLab wiki page with new information.
func (r *gitlabWikiPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabWikiPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID := data.Project.ValueString()
	options := &gitlab.EditWikiPageOptions{
		Title:   gitlab.Ptr(data.Title.ValueString()),
		Content: gitlab.Ptr(data.Content.ValueString()),
	}

	_, _, err := r.client.Wikis.EditWikiPage(projectID, data.Slug.ValueString(), options)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API Error", fmt.Sprintf("Failed to update wiki page: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Info(ctx, "Wiki page updated", map[string]any{"id": data.Id.ValueString()})
}

// Delete removes the GitLab wiki page.
func (r *gitlabWikiPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabWikiPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Project.ValueString()
	_, err := r.client.Wikis.DeleteWikiPage(projectID, data.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("GitLab API Error", fmt.Sprintf("Failed to delete wiki page: %s", err.Error()))
		return
	}
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "Wiki page deleted", map[string]any{"id": data.Id.ValueString()})
}

func (r *gitlabWikiPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
