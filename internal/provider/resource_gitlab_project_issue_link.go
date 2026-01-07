package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectIssueLinkResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIssueLinkResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIssueLinkResource{}

	allowedLinkTypes = []string{"relates_to", "blocks", "is_blocked_by"}
)

func init() {
	registerResource(NewGitlabProjectIssueLinkResource)
}

func NewGitlabProjectIssueLinkResource() resource.Resource {
	return &gitlabProjectIssueLinkResource{}
}

type gitlabProjectIssueLinkResource struct {
	client *gitlab.Client
}

type gitlabProjectIssueLinkResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Project         types.String `tfsdk:"project"`
	IssueIID        types.Int64  `tfsdk:"issue_iid"`
	TargetProjectID types.String `tfsdk:"target_project_id"`
	TargetIssueIID  types.Int64  `tfsdk:"target_issue_iid"`
	LinkType        types.String `tfsdk:"link_type"`
	IssueLinkID     types.Int64  `tfsdk:"issue_link_id"`
}

func (r *gitlabProjectIssueLinkResourceModel) issueLinkServiceToStateModel(projectId string, issueIID int64, issueLinkID int64, issueLink *gitlab.IssueLink) {
	r.ID = types.StringValue(fmt.Sprintf("%s:%d:%d", projectId, issueIID, issueLinkID))
	r.Project = types.StringValue(projectId)
	r.IssueIID = types.Int64Value(issueIID)

	if issueLink.TargetIssue != nil {
		r.TargetProjectID = types.StringValue(strconv.FormatInt(issueLink.TargetIssue.ProjectID, 10))
		r.TargetIssueIID = types.Int64Value(issueLink.TargetIssue.IID)
	}

	r.LinkType = types.StringValue(issueLink.LinkType)
	r.IssueLinkID = types.Int64Value(issueLinkID)
}

func (r *gitlabProjectIssueLinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_issue_link"
}

func (r *gitlabProjectIssueLinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_issue_link`" + ` resource allows to manage the lifecycle of project issue links.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/issue_links/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<issue_iid>:<issue_link_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"issue_iid": schema.Int64Attribute{
				MarkdownDescription: "The internal ID of a project's issue.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"target_project_id": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the target project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"target_issue_iid": schema.Int64Attribute{
				MarkdownDescription: "The internal ID of the target issue.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"link_type": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Type of the relationship. Valid values are %s.", utils.RenderValueListForDocs(allowedLinkTypes)),
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf(allowedLinkTypes...)},
			},
			"issue_link_id": schema.Int64Attribute{
				MarkdownDescription: "ID of an issue relationship.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabProjectIssueLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIssueLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectIssueLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectIssueLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.Project.ValueString()
	issueIID := data.IssueIID.ValueInt64()
	targetProjectID := data.TargetProjectID.ValueString()
	targetIssueIID := data.TargetIssueIID.ValueInt64()
	linkType := data.LinkType.ValueString()

	options := &gitlab.CreateIssueLinkOptions{
		TargetProjectID: gitlab.Ptr(targetProjectID),
		TargetIssueIID:  gitlab.Ptr(strconv.FormatInt(targetIssueIID, 10)),
		LinkType:        gitlab.Ptr(linkType),
	}

	issueLink, _, err := r.client.IssueLinks.CreateIssueLink(projectId, issueIID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create issue link: %s", err.Error()))
		return
	}

	var issueLinkID int64
	found := false

	relations, hasErr := gitlab.Scan(func(p gitlab.PaginationOptionFunc) ([]*gitlab.IssueRelation, *gitlab.Response, error) {
		return r.client.IssueLinks.ListIssueRelations(projectId, issueIID, gitlab.WithContext(ctx), p)
	})

	// Search for issue link ID, stopping early when found
	for relation := range relations {
		if relation.IID != targetIssueIID {
			continue
		}

		// Verify target project ID matches
		// If parsing fails (e.g., targetProjectID is a path), we'll still match on IID only
		parsedTargetProjectID, err := strconv.ParseInt(targetProjectID, 10, 64)
		if err == nil && relation.ProjectID != parsedTargetProjectID {
			continue
		}

		issueLinkID = relation.IssueLinkID
		found = true
		break
	}

	// Check for pagination errors after iteration completes
	if err := hasErr(); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to list issue relations: %s", err.Error()))
		return
	}

	if !found {
		resp.Diagnostics.AddError("Failed to find issue link", fmt.Sprintf("Unable to find issue link ID after creation for target project %s and issue %d", targetProjectID, targetIssueIID))
		return
	}

	data.issueLinkServiceToStateModel(projectId, issueIID, issueLinkID, issueLink)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIssueLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectIssueLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, issueIID, issueLinkID, err := parseIssueLinkID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", fmt.Sprintf("Unable to parse resource ID %s: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, "Reading GitLab project issue link", map[string]any{
		"project":       project,
		"issue_iid":     issueIID,
		"issue_link_id": issueLinkID,
	})

	issueLink, _, err := r.client.IssueLinks.GetIssueLink(project, issueIID, issueLinkID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, fmt.Sprintf("Received 404 for issue link %s. Removing from state", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read issue link: %s", err.Error()))
		return
	}

	if issueLink.SourceIssue == nil || issueLink.TargetIssue == nil {
		resp.Diagnostics.AddError("Invalid API response", "Issue link read returned invalid data")
		return
	}

	data.issueLinkServiceToStateModel(project, issueIID, issueLinkID, issueLink)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *gitlabProjectIssueLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Cannot update issue link", "Issue links are immutable. To change any attribute, please delete and recreate the resource.")
}

func (r *gitlabProjectIssueLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectIssueLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, issueIID, issueLinkID, err := parseIssueLinkID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", fmt.Sprintf("Unable to parse resource ID %s: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, "Deleting GitLab project issue link", map[string]any{
		"project":       project,
		"issue_iid":     issueIID,
		"issue_link_id": issueLinkID,
	})

	_, _, err = r.client.IssueLinks.DeleteIssueLink(project, issueIID, issueLinkID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Issue link already deleted", map[string]any{
				"id": data.ID.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete issue link: %s", err.Error()))
		return
	}

}

func parseIssueLinkID(id string) (projectId string, issueIID int64, issueLinkID int64, err error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return "", 0, 0, fmt.Errorf("unexpected ID format (%q).  Expected <project>:<issue_iid>:<issue_link_id>", id)
	}

	project := parts[0]

	issueIID, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, 0, fmt.Errorf("unable to parse issue_iid in ID: %w", err)
	}

	issueLinkID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return "", 0, 0, fmt.Errorf("unable to parse issue_link_id in ID: %w", err)
	}

	return project, issueIID, issueLinkID, nil
}
