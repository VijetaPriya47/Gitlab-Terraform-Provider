package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                   = &gitlabUserRunnerResource{}
	_ resource.ResourceWithConfigure      = &gitlabUserRunnerResource{}
	_ resource.ResourceWithImportState    = &gitlabUserRunnerResource{}
	_ resource.ResourceWithValidateConfig = &gitlabUserRunnerResource{}
	_ resource.ResourceWithModifyPlan     = &gitlabUserRunnerResource{}
)

func init() {
	registerResource(NewGitLabUserRunnerResource)
}

func NewGitLabUserRunnerResource() resource.Resource {
	return &gitlabUserRunnerResource{}
}

type gitlabUserRunnerResource struct {
	client *gitlab.Client
}

// Metadata returns the resource name
func (d *gitlabUserRunnerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_runner"
}

// Struct for the schema
type gitlabUserRunnerModel struct {
	ID              types.String `tfsdk:"id"`
	RunnerType      types.String `tfsdk:"runner_type"`
	GroupID         types.Int64  `tfsdk:"group_id"`
	ProjectID       types.Int64  `tfsdk:"project_id"`
	Description     types.String `tfsdk:"description"`
	Paused          types.Bool   `tfsdk:"paused"`
	Locked          types.Bool   `tfsdk:"locked"`
	Untagged        types.Bool   `tfsdk:"untagged"`
	TagList         types.Set    `tfsdk:"tag_list"`
	AccessLevel     types.String `tfsdk:"access_level"`
	MaximumTimeout  types.Int64  `tfsdk:"maximum_timeout"`
	Token           types.String `tfsdk:"token"`
	MaintenanceNote types.String `tfsdk:"maintenance_note"`
}

func (d *gitlabUserRunnerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Valid values for the schema:
	validRunnerTypes := []string{"instance_type", "group_type", "project_type"}
	validAccessLevels := []string{"not_protected", "ref_protected"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_user_runner`" + ` resource allows creating a GitLab runner using the new [GitLab Runner Registration Flow](https://docs.gitlab.com/ci/runners/new_creation_workflow/).

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/users/#create-a-runner-linked-to-a-user)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the gitlab runner.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"runner_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: fmt.Sprintf("The scope of the runner. Valid values are: %s.", utils.RenderValueListForDocs(validRunnerTypes)),
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(validRunnerTypes...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"group_id": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The ID of the group that the runner is created in. Required if runner_type is group_type.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace(), int64planmodifier.UseStateForUnknown()},
			},
			"project_id": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The ID of the project that the runner is created in. Required if runner_type is project_type.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace(), int64planmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Description of the runner.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"paused": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Specifies if the runner should ignore new jobs.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"locked": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Specifies if the runner should be locked for the current project.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"untagged": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Specifies if the runner should handle untagged jobs.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"tag_list": schema.SetAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A list of runner tags.",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"access_level": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: fmt.Sprintf("The access level of the runner. Valid values are: %s.", utils.RenderValueListForDocs(validAccessLevels)),
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(validAccessLevels...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"maximum_timeout": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Maximum timeout that limits the amount of time (in seconds) that runners can run jobs. Must be at least 600 (10 minutes).",
				Validators: []validator.Int64{
					int64validator.AtLeast(600), // api enforced {maximum_timeout: [needs to be at least 10 minutes]}}
				},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The authentication token to use when setting up a new runner with this configuration. This value cannot be imported.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"maintenance_note": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Free-form maintenance notes for the runner (1024 characters) ",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtMost(1024),
				},
			},
		},
	}
}

// ModifyPlan is used to ignore and updates to `Token` outside the create.
func (d *gitlabUserRunnerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var stateData, planData *gitlabUserRunnerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the `ID` is nil or unknown in state, we're doing a "Create" operation, otherwise we're doing an "Update" operation.
	// If we're doing an update operation, then we need to set Token to `null` if it's currently `null` in state
	// to prevent an "Unknown" error when using it after import.
	if stateData == nil || stateData.ID.IsNull() || stateData.ID.IsUnknown() {
		// We're doing a "Create" operation
		return
	}

	// We're performing an update, so check if the token is null or unknown in state, and set it to null in the plan
	// if it is (because we're updating, or importing, etc)
	if (stateData.Token.IsNull() || stateData.Token.IsUnknown()) && planData != nil {
		planData.Token = types.StringNull()
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planData)...)
}

// Configure adds the provider configured client to the resource.
func (r *gitlabUserRunnerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// The `create` method of this resource is special, because it doesn't use the `Runners` client, it uses the
// `users` client instead since it needs to create a user-authorized runner. Other functions will use normal
// runner client interactions.
func (r *gitlabUserRunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabUserRunnerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.CreateUserRunnerOptions{
		RunnerType: data.RunnerType.ValueStringPointer(),
	}

	if !data.GroupID.IsNull() {
		options.GroupID = gitlab.Ptr(data.GroupID.ValueInt64())
	}
	if !data.ProjectID.IsNull() {
		options.ProjectID = gitlab.Ptr(data.ProjectID.ValueInt64())
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}
	if !data.Paused.IsNull() && !data.Paused.IsUnknown() {
		options.Paused = data.Paused.ValueBoolPointer()
	}
	if !data.Locked.IsNull() && !data.Locked.IsUnknown() {
		options.Locked = data.Locked.ValueBoolPointer()
	}
	if !data.Untagged.IsNull() && !data.Untagged.IsUnknown() {
		options.RunUntagged = data.Untagged.ValueBoolPointer()
	}
	if !data.TagList.IsNull() && !data.TagList.IsUnknown() {
		// convert the Set to a []string and pass it in
		var tags []string
		data.TagList.ElementsAs(ctx, &tags, true)
		options.TagList = &tags

	}
	if !data.AccessLevel.IsNull() && !data.AccessLevel.IsUnknown() {
		options.AccessLevel = data.AccessLevel.ValueStringPointer()
	}

	// Attempting to create with a timeout of 0 causes an error, so we validate that the value is
	// greater than 0 before including it within create.
	if !data.MaximumTimeout.IsNull() && !data.MaximumTimeout.IsUnknown() && data.MaximumTimeout.ValueInt64() > 0 {
		options.MaximumTimeout = gitlab.Ptr(data.MaximumTimeout.ValueInt64())
	}
	if !data.MaintenanceNote.IsNull() && !data.MaintenanceNote.IsUnknown() {
		options.MaintenanceNote = gitlab.Ptr(data.MaintenanceNote.ValueString())
	}

	tflog.Debug(ctx, "Creating new GitLab Runner", map[string]any{
		"options": options,
	})
	userRunner, _, err := r.client.Users.CreateUserRunner(options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error creating new GitLab Runner", fmt.Sprintf("couldn't create new GitLab Runner: %v", err)))
		return
	}

	// Set the ID
	data.ID = types.StringValue(strconv.FormatInt(userRunner.ID, 10))

	// Save the token, since that is only available from the `create` function
	data.Token = types.StringValue(userRunner.Token)

	// Save this data to state so it's not lost if we fail to read the runner for whatever reason
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	// call "getRunner" so we get a Runner Details model from our user runner response. This
	// is required because the userRunner response doesn't have all the attribute values, it only has the ID and token info.
	runner, _, err := r.client.Runners.GetRunnerDetails(userRunner.ID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error reading new GitLab runner after creation", fmt.Sprintf("Error reading new GitLab runner after creation: %v", err)))
		return
	}

	resp.Diagnostics.Append(data.modelToStateModel(runner, ctx)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabUserRunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabUserRunnerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	runnerId := data.ID.ValueString()

	runner, _, err := r.client.Runners.GetRunnerDetails(runnerId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "[DEBUG] gitlab runner not found", map[string]any{
				"runnerId": runnerId,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"couldn't read runner details due to API error",
			fmt.Sprintf("couldn't read runner details due to API error: %v", err),
		))
		return
	}

	resp.Diagnostics.Append(data.modelToStateModel(runner, ctx)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabUserRunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabUserRunnerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runnerId := data.ID.ValueString()
	options := &gitlab.UpdateRunnerDetailsOptions{}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}
	if !data.Paused.IsNull() && !data.Paused.IsUnknown() {
		options.Paused = data.Paused.ValueBoolPointer()
	}
	if !data.Locked.IsNull() && !data.Locked.IsUnknown() {
		options.Locked = data.Locked.ValueBoolPointer()
	}
	if !data.Untagged.IsNull() && !data.Untagged.IsUnknown() {
		options.RunUntagged = data.Untagged.ValueBoolPointer()
	}
	if !data.TagList.IsNull() && !data.TagList.IsUnknown() {
		// convert the Set to a []string and pass it in
		var tags []string
		data.TagList.ElementsAs(ctx, &tags, true)
		options.TagList = &tags

	}
	if !data.AccessLevel.IsNull() && !data.AccessLevel.IsUnknown() {
		options.AccessLevel = data.AccessLevel.ValueStringPointer()
	}
	if !data.MaximumTimeout.IsNull() && !data.MaximumTimeout.IsUnknown() && data.MaximumTimeout.ValueInt64() > 0 {
		options.MaximumTimeout = gitlab.Ptr(data.MaximumTimeout.ValueInt64())
	}
	if !data.MaintenanceNote.IsNull() && !data.MaintenanceNote.IsUnknown() {
		options.MaintenanceNote = gitlab.Ptr(data.MaintenanceNote.ValueString())
	}

	tflog.Debug(ctx, "Updating GitLab Runner ID", map[string]any{
		"runnerId": runnerId,
		"options":  options,
	})
	runner, _, err := r.client.Runners.UpdateRunnerDetails(runnerId, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Error updating GitLab runner", fmt.Sprintf("Error updating GitLab runner %s: %v", runnerId, err)))
		return
	}

	resp.Diagnostics.Append(data.modelToStateModel(runner, ctx)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabUserRunnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabUserRunnerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runnerId, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		tflog.Debug(ctx, "[DEBUG] gitlab runner ID in state is not a number.", map[string]any{
			"id": data.ID.ValueString(),
		})
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"Attempted to delete a runner with an invalid non-integer ID",
			"The terraform provider attempted to run `delete` on a GitLab runner, but the ID was a non-integer value. This can be caused by importing an invalid ID. You may need to manually remove the runner from state.",
		))
		return
	}

	tflog.Debug(ctx, "Deleting GitLab Runner by ID", map[string]any{
		"runnerId": runnerId,
	})
	if _, err = r.client.Runners.DeleteRegisteredRunnerByID(runnerId, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete runner: %s", err.Error()),
		)
	}
}

func (r *gitlabUserRunnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabUserRunnerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data gitlabUserRunnerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// get the runner type so we can validate that proper attributes are present based on the runner type
	runnerType := data.RunnerType.ValueString()
	if runnerType == "group_type" {
		// GroupId must be provided
		if data.GroupID.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("group_id"),
				`Group ID not provided when Runner Type is set to "group_type".`,
				`When creating a Group Runner, a Group ID must be provided. The Group ID was not provided, but the Runner Type value was set to "group_type". Please provide a Group ID!`,
			)
		}

		// ProjectID must NOT be provided
		if !data.ProjectID.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("project_id"),
				`Project ID must not be provided when Runner Type is set to "group_type".`,
				`When creating a Group Runner, "project_id" must not be provided.`,
			)
		}

	}

	if runnerType == "project_type" {
		// ProjectID must be provided
		if data.ProjectID.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("project_id"),
				`Project ID not provided when Runner Type is set to "project_id".`,
				`When creating a Project Runner, a Project ID must be provided. The Project ID was not provided, but the Runner Type value was set to "project_id". Please provide a Project ID!`,
			)
		}

		// GroupID must NOT be provided
		if !data.GroupID.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("group_id"),
				`Group ID must not be provided when Runner Type is set to "project_type".`,
				`When creating a Project Runner, "group_id" must not be provided.`,
			)
		}
	}

	if runnerType == "instance_type" && (!data.ProjectID.IsNull() || !data.GroupID.IsNull()) {
		resp.Diagnostics.AddAttributeError(path.Root("runner_type"),
			`Runner Type was set to "instance_type", but a project or group ID was provided`,
			`When creating an Instance Runner, a Project ID or Group ID was provided. Those attributes are invalid for an Instance Runner, and should be removed.`,
		)
	}

	if !data.MaximumTimeout.IsNull() && !data.MaximumTimeout.IsUnknown() && data.MaximumTimeout.ValueInt64() == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("maximum_timeout"),
			`"maximum_timeout" cannot have a value of 0 configured. Please configure a value greater than 0.`,
			`"maximum_timeout" cannot have a value of 0 configured. Please configure a value greater than 0.`,
		)
	}
}

func (d *gitlabUserRunnerModel) modelToStateModel(r *gitlab.RunnerDetails, ctx context.Context) diag.Diagnostics {
	d.RunnerType = types.StringValue(r.RunnerType)
	d.Description = types.StringValue(r.Description)
	d.Paused = types.BoolValue(r.Paused)
	d.Locked = types.BoolValue(r.Locked)
	d.Untagged = types.BoolValue(r.RunUntagged)
	d.AccessLevel = types.StringValue(r.AccessLevel)
	d.MaximumTimeout = types.Int64Value(int64(r.MaximumTimeout))
	d.MaintenanceNote = types.StringValue(r.MaintenanceNote)

	// uses a []string, so doesn't require a `types` package constructor.
	list, diag := types.SetValueFrom(ctx, types.StringType, r.TagList)
	if diag != nil {
		return diag
	}

	d.TagList = list
	return nil
}
