package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &gitlabInstanceServiceAccountResource{}
	_ resource.ResourceWithConfigure   = &gitlabInstanceServiceAccountResource{}
	_ resource.ResourceWithImportState = &gitlabInstanceServiceAccountResource{}
)

func init() {
	registerResource(NewGitlabInstanceServiceAccountResource)
}

func NewGitlabInstanceServiceAccountResource() resource.Resource {
	return &gitlabInstanceServiceAccountResource{}
}

type gitlabInstanceServiceAccountResource struct {
	client *gitlab.Client
}

func (r *gitlabInstanceServiceAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_service_account"
}

// Struct for the schema
type gitlabInstanceServiceAccountResourceModel struct {
	ID               types.String   `tfsdk:"id"`
	ServiceAccountID types.String   `tfsdk:"service_account_id"`
	Name             types.String   `tfsdk:"name"`
	Username         types.String   `tfsdk:"username"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

func (r *gitlabInstanceServiceAccountResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_instance_service_account`" + ` resource allows creating a GitLab instance service account.

~> In order for a user to create a user account, they must have admin privileges at the instance level. This makes this feature unavailable on ` + "`gitlab.com`" + `

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/user_service_accounts/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. This matches the service account id.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_account_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The service account id.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The name of the user. If not specified, the default Service account user name is used.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The username of the user. If not specified, it’s automatically generated.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Delete: true,
			}),
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabInstanceServiceAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabInstanceServiceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabInstanceServiceAccountResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create service account
	serviceAccount, _, err := r.client.Users.CreateServiceAccountUser(&gitlab.CreateServiceAccountUserOptions{
		Name:     gitlab.Ptr(data.Name.ValueString()),
		Username: gitlab.Ptr(data.Username.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create service account: %s", err.Error()))
		return
	}

	data.userToStateModel(serviceAccount)
	// Log the creation of the resource
	tflog.Debug(ctx, "created a service account", map[string]interface{}{
		"id":       data.ServiceAccountID.ValueString(),
		"name":     data.Name.ValueString(),
		"username": data.Username.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabInstanceServiceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabInstanceServiceAccountResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	serviceAccountID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to convert resource ID: %s", err.Error()))
		return
	}

	serviceAccount, _, err := r.client.Users.GetUser(serviceAccountID, gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read service account: %s", err.Error()))
		return
	}

	tflog.Trace(ctx, "found service account", map[string]interface{}{
		"service account": gitlab.Stringify(serviceAccount),
	})

	data.userToStateModel(serviceAccount)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Updates updates the resource in-place.
func (r *gitlabInstanceServiceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Provider Error, report upstream",
		"Somehow the resource was requested to perform an in-place upgrade which is not possible.",
	)
}

// Deletes removes the resource.
func (r *gitlabInstanceServiceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabInstanceServiceAccountResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	serviceAccountID := data.ID.ValueString()

	serviceAccountIDInt, err := strconv.Atoi(serviceAccountID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Internal provider error",
			fmt.Sprintf("Unable to convert service account id to int: %s", err.Error()),
		)
		return
	}

	if _, err = r.client.Users.DeleteUser(serviceAccountIDInt, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete service account: %s", err.Error()),
		)
	}

	tick_rate := 10 * time.Second
	delete_ticker := time.NewTicker(tick_rate)
	timeout, diags := data.Timeouts.Delete(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	tries := int(timeout / tick_rate)

	for i := 0; i < tries; i++ {
		_, gitlab_resp, err := r.client.Users.GetUser(serviceAccountIDInt, gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
		if gitlab_resp != nil && gitlab_resp.StatusCode == 404 {
			return
		}
		if err != nil {
			resp.Diagnostics.AddWarning(
				"GitLab API Error occurred",
				fmt.Sprintf("Unable to query service account: %s", err.Error()),
			)
		}
		<-delete_ticker.C
	}

	resp.Diagnostics.AddError(
		"Gitlab API Error occured",
		fmt.Sprintf("Deletion context exceeded: %s", err.Error()),
	)
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabInstanceServiceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabInstanceServiceAccountResourceModel) userToStateModel(serviceAccount *gitlab.User) {
	serviceAccountIDStr := strconv.Itoa(serviceAccount.ID)
	r.ID = types.StringValue(serviceAccountIDStr)
	r.ServiceAccountID = types.StringValue(serviceAccountIDStr)
	r.Name = types.StringValue(serviceAccount.Name)
	r.Username = types.StringValue(serviceAccount.Username)
}
