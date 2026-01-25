package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabBranchResource{}
	_ resource.ResourceWithConfigure   = &gitlabBranchResource{}
	_ resource.ResourceWithImportState = &gitlabBranchResource{}
)

func init() {
	registerResource(NewGitlabBranchResource)
}

func NewGitlabBranchResource() resource.Resource {
	return &gitlabBranchResource{}
}

type gitlabBranchResource struct {
	client *gitlab.Client
}

func (r *gitlabBranchResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch"
}

type gitlabBranchResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Project           types.String `tfsdk:"project"`
	Ref               types.String `tfsdk:"ref"`
	KeepOnDestroy     types.Bool   `tfsdk:"keep_on_destroy"`
	WebURL            types.String `tfsdk:"web_url"`
	Default           types.Bool   `tfsdk:"default"`
	CanPush           types.Bool   `tfsdk:"can_push"`
	Merged            types.Bool   `tfsdk:"merged"`
	Protected         types.Bool   `tfsdk:"protected"`
	DeveloperCanMerge types.Bool   `tfsdk:"developer_can_merge"`
	DeveloperCanPush  types.Bool   `tfsdk:"developer_can_push"`
	Commit            types.Set    `tfsdk:"commit"`
}

type gitlabBranchCommitResourceModel struct {
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

func (r *gitlabBranchResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_branch`" + ` resource manages the lifecycle of a repository branch.

!> The ` + "`ref`" + ` attribute is only set in state on resource creation. Imports or divergent branches can lead Terraform to destroy and recreate the resource. Use the lifecycle meta-argument to ignore changes to avoid this behavior.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/branches/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource. In the format `<project:name>`",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name for this branch.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project which the branch is created against.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ref": schema.StringAttribute{
				MarkdownDescription: "The ref which the branch is created from.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"keep_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the branch is kept once the resource destroyed (must be applied before a destroy).",
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Computed:            true,
			},
			"web_url": schema.StringAttribute{
				MarkdownDescription: "The url of the created branch (https).",
				Computed:            true,
			},
			"default": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if branch is the default branch for the project.",
				Computed:            true,
			},
			"can_push": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if you can push to the branch.",
				Computed:            true,
			},
			"merged": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if the branch has been merged into its parent.",
				Computed:            true,
			},
			"protected": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if branch has branch protection.",
				Computed:            true,
			},
			"developer_can_merge": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if developer level access allows to merge branch.",
				Computed:            true,
			},
			"developer_can_push": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if developer level access allows git push.",
				Computed:            true,
			},
			"commit": commitSchema(),
		},
	}
}

func commitSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		MarkdownDescription: "The commit associated with the branch ref.",
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

func (r *gitlabBranchResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabBranchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabBranchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabBranchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	project := data.Project.ValueString()
	ref := data.Ref.ValueString()
	branchOptions := &gitlab.CreateBranchOptions{
		Branch: &name,
		Ref:    &ref,
	}

	tflog.Debug(ctx, fmt.Sprintf("create gitlab branch %s for project %s with ref %s", name, project, ref))
	branch, _, err := r.client.Branches.CreateBranch(project, branchOptions, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read branch: %s", err.Error()))
		return
	}
	data.Ref = types.StringValue(ref)
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &name))
	resp.Diagnostics.Append(data.modelToStateModel(ctx, project, branch)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabBranchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabBranchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project, name, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Read. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read gitlab branch %s", name))
	branch, _, err := r.client.Branches.GetBranch(project, name, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("received 404 for gitlab branch %s, removing from state", name))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read branch: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &name))
	resp.Diagnostics.Append(data.modelToStateModel(ctx, project, branch)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// This function exists only to update the `keep_on_destroy` in state. No action is necessary, because all important attributes
// force re-creation of the resource.
func (r *gitlabBranchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabBranchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project, name, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Read. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read gitlab branch %s", name))
	branch, _, err := r.client.Branches.GetBranch(project, name, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("received 404 for gitlab branch %s, removing from state", name))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read branch: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &name))
	resp.Diagnostics.Append(data.modelToStateModel(ctx, project, branch)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabBranchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabBranchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, name, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Delete. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("delete gitlab branch %s", name))

	if data.KeepOnDestroy.ValueBool() {
		tflog.Info(ctx, fmt.Sprintf("skipping deletion of branch %s, 'keep_on_destroy' enabled", name))
		resp.State.RemoveResource(ctx)
		return
	}

	_, err = r.client.Branches.DeleteBranch(project, name, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete branch: %s", err.Error()))
		return
	}
}

func (d *gitlabBranchResourceModel) modelToStateModel(ctx context.Context, project string, branch *gitlab.Branch) diag.Diagnostics {
	d.Name = types.StringValue(branch.Name)
	d.Project = types.StringValue(project)
	d.WebURL = types.StringValue(branch.WebURL)
	d.Default = types.BoolValue(branch.Default)
	d.CanPush = types.BoolValue(branch.CanPush)
	d.Merged = types.BoolValue(branch.Merged)
	d.DeveloperCanMerge = types.BoolValue(branch.DevelopersCanMerge)
	d.DeveloperCanPush = types.BoolValue(branch.DevelopersCanPush)
	d.Protected = types.BoolValue(branch.Protected)

	if branch.Commit == nil {
		d.Commit = types.SetNull(commitSchema().NestedObject.Type())
	} else {
		parentIDs, diags := types.SetValueFrom(ctx, types.StringType, branch.Commit.ParentIDs)
		if diags.HasError() {
			return diags
		}
		modelCommit := gitlabBranchCommitResourceModel{
			ID:             types.StringValue(branch.Commit.ID),
			ShortID:        types.StringValue(branch.Commit.ShortID),
			Title:          types.StringValue(branch.Commit.Title),
			AuthorName:     types.StringValue(branch.Commit.AuthorName),
			AuthorEmail:    types.StringValue(branch.Commit.AuthorEmail),
			AuthoredDate:   types.StringValue(branch.Commit.AuthoredDate.Format(time.RFC3339)),
			CommittedDate:  types.StringValue(branch.Commit.CommittedDate.Format(time.RFC3339)),
			CommitterEmail: types.StringValue(branch.Commit.CommitterEmail),
			CommitterName:  types.StringValue(branch.Commit.CommitterName),
			Message:        types.StringValue(branch.Commit.Message),
			ParentIDs:      parentIDs,
		}
		commitSet, diags := types.SetValueFrom(ctx, commitSchema().NestedObject.Type(), []gitlabBranchCommitResourceModel{modelCommit})
		if diags.HasError() {
			return diags
		}
		d.Commit = commitSet
	}
	return nil
}
