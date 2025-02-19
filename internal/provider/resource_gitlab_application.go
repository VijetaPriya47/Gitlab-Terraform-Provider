package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	conv "github.com/dcarbone/terraform-plugin-framework-utils/v3/conv"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &gitlabApplicationResource{}
	_ resource.ResourceWithConfigure   = &gitlabApplicationResource{}
	_ resource.ResourceWithImportState = &gitlabApplicationResource{}
)

func init() {
	registerResource(NewGitLabApplicationResource)
}

// NewGitLabApplicationResource is a helper function to simplify the provider implementation.
func NewGitLabApplicationResource() resource.Resource {
	return &gitlabApplicationResource{}
}

func (r *gitlabApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

// gitlabApplicationResource defines the resource implementation.
type gitlabApplicationResource struct {
	client *gitlab.Client
}

// gitlabProjectProtectedEnvironmentResourceModel describes the resource data model.
type gitlabApplicationResourceModel struct {
	Name         types.String `tfsdk:"name"`
	RedirectURL  types.String `tfsdk:"redirect_url"`
	Scopes       types.Set    `tfsdk:"scopes"`
	Confidential types.Bool   `tfsdk:"confidential"`

	Id            types.String `tfsdk:"id"`
	Secret        types.String `tfsdk:"secret"`
	ApplicationId types.String `tfsdk:"application_id"`
}

func (r *gitlabApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	allowedScopes := []string{"api", "read_api", "read_user", "read_repository", "write_repository", "read_registry",
		"write_registry", "sudo", "admin_mode", "openid", "profile", "email"}
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_application`" + ` resource allows to manage the lifecycle of applications in gitlab.

~> In order to use a user for a user to create an application, they must have admin privileges at the instance level.
To create an OIDC application, a scope of "openid".

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/applications/)`),

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<application_id>`.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the application.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"redirect_url": schema.StringAttribute{
				MarkdownDescription: "The URL gitlab should send the user to after authentication.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"scopes": schema.SetAttribute{
				MarkdownDescription: fmt.Sprintf(`
					Scopes of the application. Use "openid" if you plan to use this as an oidc authentication application. Valid options are: %s.
This is only populated when creating a new application. This attribute is not available for imported resources
					`,
					utils.RenderValueListForDocs(allowedScopes),
				),
				ElementType:   types.StringType,
				Required:      true,
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace(), setplanmodifier.UseStateForUnknown()},
				Validators:    []validator.Set{setvalidator.ValueStringsAre(stringvalidator.OneOf(allowedScopes...))},
			},
			"confidential": schema.BoolAttribute{
				MarkdownDescription: "The application is used where the client secret can be kept confidential. Native mobile apps and Single Page Apps are considered non-confidential. Defaults to true if not supplied",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace(), boolplanmodifier.UseStateForUnknown()},
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Application secret. Sensitive and must be kept secret. This is only populated when creating a new application. This attribute is not available for imported resources.",
				Computed:            true,
				Sensitive:           true,
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "Internal name of the application.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Create creates a new upstream resources and adds it into the Terraform state.
func (r *gitlabApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabApplicationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating application", map[string]interface{}{
		"scopes": data.Scopes.String(),
	})
	scopes := conv.StringSetToStrings(data.Scopes)
	if resp.Diagnostics.HasError() {
		return
	}

	formatted_scopes := strings.Join(scopes, " ")

	// configure GitLab API call
	options := &gitlab.CreateApplicationOptions{
		Name:        gitlab.Ptr(data.Name.ValueString()),
		RedirectURI: gitlab.Ptr(data.RedirectURL.ValueString()),
		Scopes:      gitlab.Ptr(formatted_scopes),
	}

	if !data.Confidential.IsNull() {
		options.Confidential = gitlab.Ptr(data.Confidential.ValueBool())
	}

	// Create application
	application, _, err := r.client.Applications.CreateApplication(options)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create application: %s", err.Error()))
		return
	}

	r.applicationModelToState(application, data)
	// Log the creation of the resource
	tflog.Debug(ctx, "created an application", map[string]interface{}{
		"name": data.Name.ValueString(), "id": data.Id.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabApplicationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	application, err := findGitlabApplication(r.client, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create application: %s", err.Error()))
		return
	}

	tflog.Trace(ctx, "found application", map[string]interface{}{
		"application": gitlab.Stringify(application),
	})

	r.applicationModelToState(application, data)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Updates updates the resource in-place.
func (r *gitlabApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Provider Error, report upstream",
		"Somehow the resource was requested to perform an in-place upgrade which is not possible.",
	)
}

// Deletes removes the resource.
func (r *gitlabApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabApplicationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Internal provider error",
			fmt.Sprintf("Unable to convert application id to int: %s", err.Error()),
		)
		return
	}

	if _, err = r.client.Applications.DeleteApplication(id); err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete application: %s", err.Error()),
		)
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabApplicationResource) applicationModelToState(application *gitlab.Application, data *gitlabApplicationResourceModel) {
	// need to check this
	// For reads, the secret will be empty, in which case we shouldn't set the state
	if application.Secret != "" {
		data.Secret = types.StringValue(application.Secret)
	}
	data.Id = types.StringValue(strconv.Itoa(application.ID))
	data.Confidential = types.BoolValue(application.Confidential)
	data.Name = types.StringValue(application.ApplicationName)
	data.RedirectURL = types.StringValue(application.CallbackURL)
	data.ApplicationId = types.StringValue(application.ApplicationID)
}

func findGitlabApplication(client *gitlab.Client, desiredId string) (*gitlab.Application, error) {

	options := gitlab.ListApplicationsOptions{
		PerPage: 20,
		Page:    1,
	}

	for options.Page != 0 {
		paginatedApplications, resp, err := client.Applications.ListApplications(&options)
		if err != nil {
			return nil, fmt.Errorf("unable to list applications. %s", err)
		}

		for i := range paginatedApplications {
			if strconv.Itoa(paginatedApplications[i].ID) == desiredId {
				return paginatedApplications[i], nil
			}
		}

		options.Page = resp.NextPage
	}

	// if we loop through the pages and haven't found it, we should error
	return nil, fmt.Errorf("unable to find application with id: %s", desiredId)
}
