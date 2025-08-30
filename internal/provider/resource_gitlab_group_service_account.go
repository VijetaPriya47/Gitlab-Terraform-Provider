package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &gitlabGroupServiceAccountResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupServiceAccountResource{}
	_ resource.ResourceWithImportState = &gitlabGroupServiceAccountResource{}
)

func init() {
	registerResource(NewGitlabGroupServiceAccountResource)
}

// NewGitlabGroupServiceAccountResource is a helper function to simplify the provider implementation.
func NewGitlabGroupServiceAccountResource() resource.Resource {
	return &gitlabGroupServiceAccountResource{}
}

// gitlabGroupServiceAccountResource defines the resource implementation.
type gitlabGroupServiceAccountResource struct {
	client *gitlab.Client
}

func (r *gitlabGroupServiceAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_service_account"
}

// Struct for the schema
type gitlabGroupServiceAccountResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
	Group            types.String `tfsdk:"group"`
	Name             types.String `tfsdk:"name"`
	Username         types.String `tfsdk:"username"`
	Email            types.String `tfsdk:"email"`
}

func (r *gitlabGroupServiceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_service_account`" + ` resource allows creating a GitLab group service account.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_service_accounts/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group>:<service_account_id>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_account_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The service account id.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID or URL-encoded path of the group that the service account is created in. Must be a top level group.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
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
			"email": schema.StringAttribute{
				MarkdownDescription: "User account email. If not specified, generates an email prepended with `service_account_group_`. Custom email addresses require confirmation before the account is active, unless the group has a matching verified domain.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabGroupServiceAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Create creates a new upstream resources and adds it into the Terraform state.
func (r *gitlabGroupServiceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupServiceAccountResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Local variables for easier reference
	group := data.Group.ValueString()

	// configure GitLab API call
	options := &gitlab.CreateServiceAccountOptions{
		Name:     gitlab.Ptr(data.Name.ValueString()),
		Username: gitlab.Ptr(data.Username.ValueString()),
	}

	if !data.Email.IsNull() && !data.Email.IsUnknown() {
		options.Email = gitlab.Ptr(data.Email.ValueString())
	}

	// Create service account
	serviceAccount, _, err := r.client.Groups.CreateServiceAccount(group, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create service account: %s", err.Error()))
		return
	}

	data.serviceAccountToStateModel(serviceAccount, group)
	// Log the creation of the resource
	tflog.Debug(ctx, "created a service account", map[string]any{
		"id":       data.ServiceAccountID.ValueString(),
		"name":     data.Name.ValueString(),
		"username": data.Username.ValueString(),
		"email":    data.Email.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabGroupServiceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupServiceAccountResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	group, serviceAccountID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Read. It should be '<group>:<service_account_id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	serviceAccount, err := findGitlabServiceAccount(r.client, group, serviceAccountID)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read service account: %s", err.Error()))
		return
	}

	tflog.Trace(ctx, "found service account", map[string]any{
		"service account": gitlab.Stringify(serviceAccount),
	})

	data.serviceAccountToStateModel(serviceAccount, group)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Updates updates the resource in-place.
func (r *gitlabGroupServiceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Provider Error, report upstream",
		"Somehow the resource was requested to perform an in-place upgrade which is not possible.",
	)
}

// Deletes removes the resource.
func (r *gitlabGroupServiceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupServiceAccountResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	group, serviceAccountID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Read. It should be '<group>:<service_account_id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	serviceAccountIDInt, err := strconv.Atoi(serviceAccountID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Internal provider error",
			fmt.Sprintf("Unable to convert service account id to int: %s", err.Error()),
		)
		return
	}

	if _, err = r.client.Groups.DeleteServiceAccount(group, serviceAccountIDInt, nil, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete service account: %s", err.Error()),
		)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	done := ctx.Done()

	for {
		select {
		case <-done:
			resp.Diagnostics.AddError(
				"GitLab API Error occurred",
				"Timed out waiting for group service account deletion to complete",
			)
			return
		case <-ticker.C:
			_, _, err := r.client.Users.GetUser(serviceAccountIDInt, gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
			if api.Is404(err) {
				return
			}

			if err != nil {
				tflog.Warn(ctx, fmt.Sprintf("Error checking group service account deletion status: %s", err.Error()))
			}
		}
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabGroupServiceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabGroupServiceAccountResourceModel) serviceAccountToStateModel(serviceAccount *gitlab.GroupServiceAccount, group string) {
	// attributes from api response
	serviceAccountIDStr := strconv.Itoa(serviceAccount.ID)
	r.ID = types.StringValue(utils.BuildTwoPartID(&group, &serviceAccountIDStr))
	r.ServiceAccountID = types.StringValue(serviceAccountIDStr)
	r.Group = types.StringValue(group)
	r.Name = types.StringValue(serviceAccount.Name)
	r.Username = types.StringValue(serviceAccount.UserName)
	r.Email = types.StringValue(serviceAccount.Email)
}

func findGitlabServiceAccount(client *gitlab.Client, group, desiredId string) (*gitlab.GroupServiceAccount, error) {
	options := gitlab.ListServiceAccountsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}

	for options.Page != 0 {
		paginatedServiceAccounts, resp, err := client.Groups.ListServiceAccounts(group, &options)
		if err != nil {
			return nil, fmt.Errorf("unable to list service accounts. %s", err)
		}

		for i := range paginatedServiceAccounts {
			if strconv.Itoa(paginatedServiceAccounts[i].ID) == desiredId {
				return paginatedServiceAccounts[i], nil
			}
		}

		options.Page = resp.NextPage
	}

	// if we loop through the pages and haven't found it, we should error
	return nil, fmt.Errorf("unable to find service account with id: %s", desiredId)
}
