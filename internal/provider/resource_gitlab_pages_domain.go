package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabPagesDomainResource{}
	_ resource.ResourceWithConfigure   = &gitlabPagesDomainResource{}
	_ resource.ResourceWithImportState = &gitlabPagesDomainResource{}
)

func init() {
	registerResource(NewGitLabPagesDomainResource)
}

func NewGitLabPagesDomainResource() resource.Resource {
	return &gitlabPagesDomainResource{}
}

type gitlabPagesDomainResource struct {
	client *gitlab.Client
}

type gitLabPagesDomainResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Domain             types.String `tfsdk:"domain"`
	Project            types.String `tfsdk:"project"`
	AutoSslEnabled     types.Bool   `tfsdk:"auto_ssl_enabled"`
	Key                types.String `tfsdk:"key"`
	URL                types.String `tfsdk:"url"`
	Verified           types.Bool   `tfsdk:"verified"`
	VerificationString types.String `tfsdk:"verification_code"`
	Certificate        types.String `tfsdk:"certificate"`
	Expired            types.Bool   `tfsdk:"expired"`
}

// Metadata returns the resource name
func (d *gitlabPagesDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pages_domain"
}

// Schema defines the schema for the resource
func (d *gitlabPagesDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_pages_domain`" + ` resource allows connecting custom domains and TLS certificates in GitLab Pages.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/pages_domains.html)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<domain>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The custom domain indicated by the user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or [URL-encoded path of the project](https://docs.gitlab.com/ee/api/index.html#namespaced-path-encoding) owned by the authenticated user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"auto_ssl_enabled": schema.BoolAttribute{
				MarkdownDescription: `Enables [automatic generation](https://docs.gitlab.com/ee/user/project/pages/custom_domains_ssl_tls_certification/lets_encrypt_integration.html) of SSL certificates issued by Let’s Encrypt for custom domains. When this is set to "true", certificate can't be provided.`,
				Optional:            true,
				Computed:            true,
				Validators: []validator.Bool{
					autoSslEnabledValidator{},
				},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The certificate key in PEM format.",
				Optional:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL for the given domain.",
				Computed:            true,
			},
			"verified": schema.BoolAttribute{
				MarkdownDescription: "The certificate data.",
				Computed:            true,
			},
			"verification_code": schema.StringAttribute{
				MarkdownDescription: "The verification code for the domain.",
				Computed:            true,
				Sensitive:           true,
			},
			"certificate": schema.StringAttribute{
				MarkdownDescription: "The certificate in PEM format with intermediates following in most specific to least specific order.",
				Optional:            true,
				Computed:            true,
			},
			"expired": schema.BoolAttribute{
				MarkdownDescription: "Whether the certificate is expired.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

// Configure adds the client implementation to the resource
func (r *gitlabPagesDomainResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (d *gitlabPagesDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Get plan information into our struct
	var data gitLabPagesDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Local variables for easier reference
	projectID := data.Project.ValueString()

	// Create our resource
	options := &gitlab.CreatePagesDomainOptions{
		Domain: gitlab.Ptr(data.Domain.ValueString()),
	}
	if !data.AutoSslEnabled.IsNull() && !data.AutoSslEnabled.IsUnknown() {
		options.AutoSslEnabled = gitlab.Ptr(data.AutoSslEnabled.ValueBool())
	}
	if !data.Certificate.IsNull() && !data.Certificate.IsUnknown() {
		options.Certificate = gitlab.Ptr(data.Certificate.ValueString())
	}
	if !data.Key.IsNull() && !data.Key.IsUnknown() {
		options.Key = gitlab.Ptr(data.Key.ValueString())
	}

	pagesDomain, _, err := d.client.PagesDomains.CreatePagesDomain(projectID, options)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating pages domain for project %s", data.Project),
			err.Error(),
		)
		return
	}

	data.pagesDomainToStateModel(pagesDomain, projectID)

	// Create the ID attribute (used for imports, among other things)
	data.ID = types.StringValue(utils.BuildTwoPartID(&projectID, gitlab.Ptr(data.Domain.ValueString())))

	tflog.Debug(ctx, "created pages domain", map[string]interface{}{
		"url": data.URL, "project": data.Project,
	})

	// Set our plan object into state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabPagesDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitLabPagesDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, domain, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<domain>'. Error: %s", data.ID, err.Error()),
		)
		return
	}

	pagesDomain, _, err := d.client.PagesDomains.GetPagesDomain(projectID, domain)
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "pages domain doesn't exist, removing from state", map[string]interface{}{
				"url": data.URL, "project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read pages domain details: %s", err.Error()))
		return
	}

	data.pagesDomainToStateModel(pagesDomain, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Updates updates the resource in-place.
func (d *gitlabPagesDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Get data information into our struct
	var data gitLabPagesDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update our resource
	options := &gitlab.UpdatePagesDomainOptions{}
	if !data.AutoSslEnabled.IsNull() && !data.AutoSslEnabled.IsUnknown() {
		options.AutoSslEnabled = gitlab.Ptr(data.AutoSslEnabled.ValueBool())
	}
	if !data.Certificate.IsNull() && !data.Certificate.IsUnknown() {
		options.Certificate = gitlab.Ptr(data.Certificate.ValueString())
	}
	if !data.Key.IsNull() && !data.Key.IsUnknown() {
		options.Key = gitlab.Ptr(data.Key.ValueString())
	}

	projectID := data.Project.ValueString()
	pagesDomain, _, err := d.client.PagesDomains.UpdatePagesDomain(projectID, data.Domain.ValueString(), options)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating pages domain for project %s", data.Project),
			err.Error(),
		)
		return
	}

	data.pagesDomainToStateModel(pagesDomain, projectID)

	// Create the ID attribute (used for imports, among other things)
	data.ID = types.StringValue(utils.BuildTwoPartID(gitlab.Ptr(data.Project.ValueString()), gitlab.Ptr(data.Domain.ValueString())))

	tflog.Debug(ctx, "updated pages domain", map[string]interface{}{
		"url": data.URL, "project": data.Project,
	})

	// Set our plan object into state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Deletes removes the resource.
func (d *gitlabPagesDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitLabPagesDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, domain, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<domain>'. Error: %s", data.ID, err.Error()),
		)
		return
	}

	if _, err := d.client.PagesDomains.DeletePagesDomain(projectID, domain); err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete pages domain: %s", err.Error()),
		)
	}
}

func (r *gitlabPagesDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitLabPagesDomainResourceModel) pagesDomainToStateModel(pages *gitlab.PagesDomain, projectID string) {
	// attributes from api response
	r.Domain = types.StringValue(pages.Domain)
	r.Project = types.StringValue(projectID)
	r.AutoSslEnabled = types.BoolValue(pages.AutoSslEnabled)
	r.URL = types.StringValue(pages.URL)
	r.VerificationString = types.StringValue(pages.VerificationCode)
	r.Verified = types.BoolValue(pages.Verified)
	r.Expired = types.BoolValue(pages.Certificate.Expired)
	r.Certificate = types.StringValue(pages.Certificate.Certificate)

	// r.Key will always come from state, there is no API that exposes it.
}

// Create a validator that validates that "auto_ssl_enabled" only conflicts with certificate when
// set to "true"
type autoSslEnabledValidator struct{}

func (v autoSslEnabledValidator) Description(ctx context.Context) string {
	return `"certificate" can't be included when "auto_ssl_enabled" is set to true`
}

func (v autoSslEnabledValidator) MarkdownDescription(ctx context.Context) string {
	return `"certificate" can't be included when "auto_ssl_enabled" is set to true`
}

func (v autoSslEnabledValidator) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	// If nothing is configured for auto_ssl_enabled, skip validation because it can't conflict.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	// This could be combined in the above "if" statement, but this makes it easier to read.
	// We want to skip validation if "auto_ssl_enabled" is false, because then it shouldn't conflict.
	if !req.ConfigValue.ValueBool() {
		return
	}

	// We know "auto_ssl_enabled" is set to "true" at this point, so check if certificate is present, and add a diagnostic
	// if it's present.
	var data gitLabPagesDomainResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if !data.Certificate.IsNull() {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(
			req.Path,
			`"certificate" can't be included when "auto_ssl_enabled" is set to true`,
		))
	}

}
