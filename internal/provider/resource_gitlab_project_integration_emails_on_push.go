package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectIntegrationEmailsOnPushResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationEmailsOnPushResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationEmailsOnPushResource{}
)

func init() {
	registerResource(NewGitLabProjectIntegrationEmailsOnPushResource)

	// Remove in 19.0
	registerResource(NewGitLabIntegrationEmailsOnPushResource)
}

func NewGitLabProjectIntegrationEmailsOnPushResource() resource.Resource {
	return &gitlabProjectIntegrationEmailsOnPushResource{
		ResourceName: "_project_integration_emails_on_push",
		ResourceDescription: `The ` + "`" + `gitlab_project_integration_emails_on_push` + "`" + ` resource manages the lifecycle of a project integration with the Emails on Push Service.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#emails-on-push)`,
	}
}

// Remove in 19.0
func NewGitLabIntegrationEmailsOnPushResource() resource.Resource {
	return &gitlabProjectIntegrationEmailsOnPushResource{
		ResourceName: "_integration_emails_on_push",
		ResourceDescription: `The ` + "`" + `gitlab_integration_emails_on_push` + "`" + ` resource manages the lifecycle of a project integration with the Emails on Push Service.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_integration_emails_on_push` + "`" + `instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#emails-on-push)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_emails_on_push` instead.",
	}
}

type gitlabProjectIntegrationEmailsOnPushResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_integration_emails_on_push`
	// and `gitlab_integration_emails_on_push` for backwards compatibility reasons. Should be removed in 19.0.
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

type gitlabProjectIntegrationEmailsOnPushResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Project                types.String `tfsdk:"project"`
	Recipients             types.String `tfsdk:"recipients"`
	DisableDiffs           types.Bool   `tfsdk:"disable_diffs"`
	SendFromCommitterEmail types.Bool   `tfsdk:"send_from_committer_email"`
	PushEvents             types.Bool   `tfsdk:"push_events"`
	TagPushEvents          types.Bool   `tfsdk:"tag_push_events"`
	BranchesToBeNotified   types.String `tfsdk:"branches_to_be_notified"`
	Title                  types.String `tfsdk:"title"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
	Slug                   types.String `tfsdk:"slug"`
	Active                 types.Bool   `tfsdk:"active"`
}

func (r *gitlabProjectIntegrationEmailsOnPushResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationEmailsOnPushResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: r.ResourceDescription,
		DeprecationMessage:  r.DeprecationMessage,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID or full-path of the project you want to activate integration on.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"recipients": schema.StringAttribute{
				MarkdownDescription: "Emails separated by whitespace.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"disable_diffs": schema.BoolAttribute{
				MarkdownDescription: "Disable code diffs.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"send_from_committer_email": schema.BoolAttribute{
				MarkdownDescription: "Send from committer.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for push events.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"tag_push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for tag push events.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"branches_to_be_notified": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Branches to send notifications for. Valid options are %s. Notifications are always fired for tag pushes.", utils.RenderValueListForDocs(api.ValidBranchesToBeNotified)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("all"),
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidBranchesToBeNotified...)},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Title of the integration.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this integration was activated at in UTC.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this integration was last updated at in UTC.",
				Computed:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "The name of the integration in lowercase, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the integration is active.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabProjectIntegrationEmailsOnPushResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationEmailsOnPushResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectIntegrationEmailsOnPushResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error creating emails on push integration for project %s", err.Error()))
		return
	}

	projectID := data.Project.ValueString()
	data.ID = types.StringValue(projectID)
	data.modelToStateModel(service, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationEmailsOnPushResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectIntegrationEmailsOnPushResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	service, _, err := r.client.Services.GetEmailsOnPushService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "emails on push integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading emails on push integration for project %s", err.Error()))
		return
	}

	data.modelToStateModel(service, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationEmailsOnPushResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectIntegrationEmailsOnPushResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error updating emails on push integration for project %s", err.Error()))
		return
	}

	data.modelToStateModel(service, data.Project.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationEmailsOnPushResource) update(ctx context.Context, data *gitlabProjectIntegrationEmailsOnPushResourceModel) (*gitlab.EmailsOnPushService, error) {
	projectID := data.Project.ValueString()

	options := &gitlab.SetEmailsOnPushServiceOptions{
		Recipients:             data.Recipients.ValueStringPointer(),
		DisableDiffs:           data.DisableDiffs.ValueBoolPointer(),
		SendFromCommitterEmail: data.SendFromCommitterEmail.ValueBoolPointer(),
		PushEvents:             data.PushEvents.ValueBoolPointer(),
		TagPushEvents:          data.TagPushEvents.ValueBoolPointer(),
		BranchesToBeNotified:   data.BranchesToBeNotified.ValueStringPointer(),
	}

	service, _, err := r.client.Services.SetEmailsOnPushService(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (r *gitlabProjectIntegrationEmailsOnPushResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectIntegrationEmailsOnPushResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	if _, err := r.client.Services.DeleteEmailsOnPushService(projectID, gitlab.WithContext(ctx)); err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "emails on push integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting emails on push integration for project %s", err.Error()))
		return
	}
}

func (r *gitlabProjectIntegrationEmailsOnPushResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabProjectIntegrationEmailsOnPushResourceModel) modelToStateModel(service *gitlab.EmailsOnPushService, projectID string) {
	d.Project = types.StringValue(projectID)
	d.Recipients = types.StringValue(service.Properties.Recipients)
	d.DisableDiffs = types.BoolValue(service.Properties.DisableDiffs)
	d.SendFromCommitterEmail = types.BoolValue(service.Properties.SendFromCommitterEmail)
	d.PushEvents = types.BoolValue(service.PushEvents)
	d.TagPushEvents = types.BoolValue(service.TagPushEvents)
	d.BranchesToBeNotified = types.StringValue(service.Properties.BranchesToBeNotified)
	d.Title = types.StringValue(service.Title)
	d.CreatedAt = types.StringValue(service.CreatedAt.Format(time.RFC3339))
	if service.UpdatedAt != nil {
		d.UpdatedAt = types.StringValue(service.UpdatedAt.Format(time.RFC3339))
	} else {
		d.UpdatedAt = types.StringNull()
	}
	d.Slug = types.StringValue(service.Slug)
	d.Active = types.BoolValue(service.Active)
}
