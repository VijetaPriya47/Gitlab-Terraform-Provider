package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"gitlab.com/gitlab-org/api/client-go"
)

var (
	_ resource.Resource                = &gitlabServiceCustomIssueTrackerResource{}
	_ resource.ResourceWithConfigure   = &gitlabServiceCustomIssueTrackerResource{}
	_ resource.ResourceWithImportState = &gitlabServiceCustomIssueTrackerResource{}
)

func init() {
	registerResource(NewGitlabServiceCustomIssueTrackerResource)
}

func NewGitlabServiceCustomIssueTrackerResource() resource.Resource {
	return &gitlabServiceCustomIssueTrackerResource{}
}

type gitlabServiceCustomIssueTrackerResource struct {
	client         *gitlab.Client
	parentResource *gitlabIntegrationCustomIssueTrackerResource
}

func (r *gitlabServiceCustomIssueTrackerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_custom_issue_tracker"
}

func (r *gitlabServiceCustomIssueTrackerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resource := &gitlabIntegrationCustomIssueTrackerResource{
		client: r.client,
	}
	resource.Schema(ctx, req, resp)
	resp.Schema.MarkdownDescription = `The ` + "`gitlab_service_custom_issue_tracker`" + ` resource allows to manage the lifecycle of a project integration with Custom Issue Tracker.

~> This resource is deprecated. use ` + "`gitlab_integration_custom_issue_tracker`" + `instead!

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/integrations.html#custom-issue-tracker)`
}

/////////////////
// All of the below methods essentially just delegate logic to `gitlabIntegrationCustomIssueTrackerResource`
// This resource is a mirror of that with a different resource name, so the below should not be modified
/////////////////

func (r *gitlabServiceCustomIssueTrackerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resource := &gitlabIntegrationCustomIssueTrackerResource{
		client: r.client,
	}
	resource.Configure(ctx, req, resp)
	r.parentResource = resource
}

func (r *gitlabServiceCustomIssueTrackerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.parentResource.Create(ctx, req, resp)
}

func (r *gitlabServiceCustomIssueTrackerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.parentResource.Read(ctx, req, resp)
}

func (r *gitlabServiceCustomIssueTrackerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.parentResource.Update(ctx, req, resp)
}

func (r *gitlabServiceCustomIssueTrackerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.parentResource.Delete(ctx, req, resp)
}

func (r *gitlabServiceCustomIssueTrackerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
