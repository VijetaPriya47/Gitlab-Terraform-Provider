package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectEnvironmentResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectEnvironmentResource{}
	_ resource.ResourceWithImportState = &gitlabProjectEnvironmentResource{}
)

func init() {
	registerResource(NewGitlabProjectEnvironmentResource)
}

func NewGitlabProjectEnvironmentResource() resource.Resource {
	return &gitlabProjectEnvironmentResource{}
}

type gitlabProjectEnvironmentResourceModel struct {
	ID                  types.String      `tfsdk:"id"`
	Project             types.String      `tfsdk:"project"`
	Name                types.String      `tfsdk:"name"`
	Description         types.String      `tfsdk:"description"`
	ExternalURL         types.String      `tfsdk:"external_url"`
	Tier                types.String      `tfsdk:"tier"`
	ClusterAgentID      types.Int64       `tfsdk:"cluster_agent_id"`
	KubernetesNamespace types.String      `tfsdk:"kubernetes_namespace"`
	FluxResourcePath    types.String      `tfsdk:"flux_resource_path"`
	Slug                types.String      `tfsdk:"slug"`
	CreatedAt           timetypes.RFC3339 `tfsdk:"created_at"`
	UpdatedAt           timetypes.RFC3339 `tfsdk:"updated_at"`
	State               types.String      `tfsdk:"state"`
	StopBeforeDestroy   types.Bool        `tfsdk:"stop_before_destroy"`
	AutoStopAt          timetypes.RFC3339 `tfsdk:"auto_stop_at"`
	AutoStopSetting     types.String      `tfsdk:"auto_stop_setting"`
}

type gitlabProjectEnvironmentResource struct {
	client *gitlab.Client
}

func (r *gitlabProjectEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment"
}

func (r *gitlabProjectEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	allowedEnvironmentTiers := []string{"production", "staging", "testing", "development", "other"}
	allowedEnvironmentAutoStopSettings := []string{"always", "with_action"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_environment`" + ` resource manages the lifecycle of an environment in a project.

-> During a terraform destroy this resource by default will not attempt to stop the environment first.
An environment is required to be in a stopped state before a deletion of the environment can occur.
Set the ` + "`stop_before_destroy`" + ` flag to attempt to automatically stop the environment before deletion. If the 
environment's ` + "`auto_stop_setting` " + `is set to ` + "`with_action`" + `, the environment will be force-stopped. 

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/environments/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource. In the format of `<project-id:environment-id>`",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project to environment is created for.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the environment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the environment.",
				Optional:            true,
				Computed:            true,
			},
			"external_url": schema.StringAttribute{
				MarkdownDescription: "Place to link to for this environment.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\S+$`), `The URL may not contain whitespace`),
				},
			},
			"tier": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The tier of the new environment. Valid values are %s.", utils.RenderValueListForDocs(allowedEnvironmentTiers)),
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedEnvironmentTiers...),
				},
			},
			"cluster_agent_id": schema.Int64Attribute{
				MarkdownDescription: "The cluster agent to associate with this environment.",
				Optional:            true,
			},
			"kubernetes_namespace": schema.StringAttribute{
				MarkdownDescription: "The Kubernetes namespace to associate with this environment.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("cluster_agent_id")),
				},
			},
			"flux_resource_path": schema.StringAttribute{
				MarkdownDescription: "The Flux resource path to associate with this environment.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("cluster_agent_id"), path.MatchRoot("kubernetes_namespace")),
				},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "The name of the environment in lowercase, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this environment was created at in UTC.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this environment was last updated at in UTC.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("State the environment is in. Valid values are %s.", utils.RenderValueListForDocs(api.ValidProjectEnvironmentStates)),
				Computed:            true,
			},
			"stop_before_destroy": schema.BoolAttribute{
				MarkdownDescription: "Determines whether the environment is attempted to be stopped before the environment is deleted. If `auto_stop_setting` is set to `with_action`, this will perform a force stop.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"auto_stop_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this environment will be automatically stopped at in UTC.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"auto_stop_setting": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The auto stop setting for the environment. Allowed values are %s. If this is set to `with_action` and `stop_before_destroy` is `true`, the environment will be force-stopped.", utils.RenderValueListForDocs(allowedEnvironmentAutoStopSettings)),
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedEnvironmentAutoStopSettings...),
				},
			},
		},
	}
}

func (r *gitlabProjectEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.CreateEnvironmentOptions{
		Name: data.Name.ValueStringPointer(),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = data.Description.ValueStringPointer()
	}

	if !data.ExternalURL.IsNull() && !data.ExternalURL.IsUnknown() {
		options.ExternalURL = data.ExternalURL.ValueStringPointer()
	}

	if !data.Tier.IsNull() && !data.Tier.IsUnknown() {
		options.Tier = data.Tier.ValueStringPointer()
	}

	if !data.ClusterAgentID.IsNull() && !data.ClusterAgentID.IsUnknown() {
		options.ClusterAgentID = gitlab.Ptr(data.ClusterAgentID.ValueInt64())
	}

	if !data.KubernetesNamespace.IsNull() && !data.KubernetesNamespace.IsUnknown() {
		options.KubernetesNamespace = data.KubernetesNamespace.ValueStringPointer()
	}

	if !data.FluxResourcePath.IsNull() && !data.FluxResourcePath.IsUnknown() {
		options.FluxResourcePath = data.FluxResourcePath.ValueStringPointer()
	}

	if !data.AutoStopSetting.IsNull() && !data.AutoStopSetting.IsUnknown() {
		options.AutoStopSetting = data.AutoStopSetting.ValueStringPointer()
	}

	project := data.Project.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Project %s create gitlab environment %q", project, *options.Name))

	environment, _, err := r.client.Environments.CreateEnvironment(project, options, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddError("Feature Environments is not available", "The Environments feature is not available for this project.")
			return
		}
		resp.Diagnostics.AddError("Error creating GitLab project environment", err.Error())
		return
	}

	environmentID := strconv.FormatInt(environment.ID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &environmentID))
	resp.Diagnostics.Append(data.modelToStateModel(project, environment)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectEnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read gitlab environment %s", data.ID.ValueString()))

	project, environmentID, err := resourceGitlabProjectEnvironmentParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing GitLab project environment ID", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Project %s read gitlab environment %d", project, environmentID))

	environment, _, err := r.client.Environments.GetEnvironment(project, environmentID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("Project %s gitlab environment %d not found, removing from state", project, environmentID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading GitLab project environment", err.Error())
		return
	}

	resp.Diagnostics.Append(data.modelToStateModel(project, environment)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("update gitlab environment %s", data.ID.ValueString()))

	project, environmentID, err := resourceGitlabProjectEnvironmentParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing GitLab project environment ID", err.Error())
		return
	}

	options := &gitlab.EditEnvironmentOptions{
		Name: data.Name.ValueStringPointer(),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = data.Description.ValueStringPointer()
	}

	if data.ExternalURL.IsNull() || data.ExternalURL.IsUnknown() {
		options.ExternalURL = gitlab.Ptr("")
	} else {
		options.ExternalURL = data.ExternalURL.ValueStringPointer()
	}

	if !data.Tier.IsNull() && !data.Tier.IsUnknown() {
		options.Tier = data.Tier.ValueStringPointer()
	}

	if !data.ClusterAgentID.IsNull() && !data.ClusterAgentID.IsUnknown() {
		options.ClusterAgentID = gitlab.Ptr(data.ClusterAgentID.ValueInt64())
	}

	if !data.KubernetesNamespace.IsNull() && !data.KubernetesNamespace.IsUnknown() {
		options.KubernetesNamespace = data.KubernetesNamespace.ValueStringPointer()
	}

	if !data.FluxResourcePath.IsNull() && !data.FluxResourcePath.IsUnknown() {
		options.FluxResourcePath = data.FluxResourcePath.ValueStringPointer()
	}

	if !data.AutoStopSetting.IsNull() && !data.AutoStopSetting.IsUnknown() {
		options.AutoStopSetting = data.AutoStopSetting.ValueStringPointer()
	}

	tflog.Debug(ctx, fmt.Sprintf("Project %s update gitlab environment %d", project, environmentID))

	environment, _, err := r.client.Environments.EditEnvironment(project, environmentID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error updating GitLab project environment", err.Error())
		return
	}

	if data.ClusterAgentID.IsNull() || data.ClusterAgentID.IsUnknown() {
		environment, err = r.updateNullableClusterAgentID(ctx, project, environmentID)
		if err != nil {
			resp.Diagnostics.AddError("Error updating nullable cluster agent ID", err.Error())
			return
		}
	}

	environmentIDStr := strconv.FormatInt(environment.ID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &environmentIDStr))
	resp.Diagnostics.Append(data.modelToStateModel(project, environment)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectEnvironmentResource) updateNullableClusterAgentID(ctx context.Context, project string, environmentID int64) (*gitlab.Environment, error) {
	options := &gitlab.EditEnvironmentOptions{}
	environment, _, err := r.client.Environments.EditEnvironment(project, environmentID, options, gitlab.WithContext(ctx), func(request *retryablehttp.Request) error {
		optionsStruct := struct {
			ClusterAgentID      *int64  `url:"cluster_agent_id" json:"cluster_agent_id"`
			KubernetesNamespace *string `url:"kubernetes_namespace" json:"kubernetes_namespace"`
			FluxResourcePath    *string `url:"flux_resource_path" json:"flux_resource_path"`
		}{
			ClusterAgentID:      nil,
			KubernetesNamespace: nil,
			FluxResourcePath:    nil,
		}

		body, err := json.Marshal(optionsStruct)
		if err != nil {
			return err
		}

		err = request.SetBody(body)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error editing gitlab project %s environment %d: %v", project, environmentID, err)
	}

	return environment, nil
}

func (r *gitlabProjectEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectEnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, environmentID, err := resourceGitlabProjectEnvironmentParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing GitLab project environment ID", err.Error())
		return
	}

	stopBeforeDestroy := data.StopBeforeDestroy.ValueBool()
	if stopBeforeDestroy {
		// To stop an environment with an on_stop action, we need to force stop.
		// https://docs.gitlab.com/ci/environments/#stop-an-environment-without-running-the-on_stop-action
		forceStop := false
		if data.AutoStopSetting.ValueString() == "with_action" {
			tflog.Debug(ctx, fmt.Sprintf("Force-stopping environment %d for Project %s with on_stop action", environmentID, project))
			forceStop = true
		}
		// resourceGitlabProjectEnvironmentStop waits for the environment to actually be stopped
		if err := r.resourceGitlabProjectEnvironmentStop(ctx, data, forceStop); err != nil {
			resp.Diagnostics.AddError("Error stopping GitLab project environment", err.Error())
			return
		}
	}

	environment, _, err := r.client.Environments.GetEnvironment(project, environmentID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("Project %s gitlab environment %d not found, removing from state", project, environmentID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading GitLab project environment", err.Error())
		return
	}

	if environment.State != "stopped" {
		resp.Diagnostics.AddError(fmt.Sprintf("cannot destroy GitLab project %s environment %d", project, environmentID), "Environment must be in a stopped state before deletion. Set stop_before_destroy flag to attempt to auto stop the environment on destruction")
		return
	}

	_, err = r.client.Environments.DeleteEnvironment(project, environmentID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting GitLab project environment", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *gitlabProjectEnvironmentResource) resourceGitlabProjectEnvironmentStop(ctx context.Context, data *gitlabProjectEnvironmentResourceModel, force bool) error {
	project, environmentID, err := resourceGitlabProjectEnvironmentParseID(data.ID.ValueString())
	if err != nil {
		return err
	}

	options := &gitlab.StopEnvironmentOptions{
		Force: gitlab.Ptr(force),
	}
	tflog.Debug(ctx, fmt.Sprintf("Stopping environment %d for Project %s", environmentID, project))
	if _, _, err = r.client.Environments.StopEnvironment(project, environmentID, options, gitlab.WithContext(ctx)); err != nil {
		return err
	}

	// Wait for the environment to be stopped, before we destroy it
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()

	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	done := ctx.Done()

	err = func() error {
		for {
			select {
			case <-done:
				if ctx.Err() == context.DeadlineExceeded {
					return fmt.Errorf("Project environment not stopped. Retrying.")
				}
				return ctx.Err()
			case <-t.C:
				_, resp, err := r.client.Environments.StopEnvironment(project, environmentID, nil, gitlab.WithContext(ctx))

				// If we get a `400` http status code with "failed to change the status", get the current status to determine
				// if the record is already stopped, and exit the state change loop if it has
				if resp.StatusCode == http.StatusBadRequest {
					// Get the current environment status
					currentEnv, _, getErr := r.client.Environments.GetEnvironment(project, environmentID, gitlab.WithContext(ctx))
					if getErr != nil {
						tflog.Warn(ctx, "Error retrieving status of environment for project", map[string]any{
							"project":     project,
							"environment": environmentID,
						})
						return getErr
					}
					if currentEnv.State == "stopped" {
						return nil
					}
				}

				return err
			}
		}
	}()

	return err
}

func (d *gitlabProjectEnvironmentResourceModel) modelToStateModel(project string, environment *gitlab.Environment) diag.Diagnostics {
	d.Project = types.StringValue(project)
	d.Name = types.StringValue(environment.Name)
	d.Description = types.StringValue(environment.Description)
	d.State = types.StringValue(environment.State)
	d.ExternalURL = types.StringValue(environment.ExternalURL)
	d.Tier = types.StringValue(environment.Tier)
	d.KubernetesNamespace = types.StringValue(environment.KubernetesNamespace)
	d.FluxResourcePath = types.StringValue(environment.FluxResourcePath)
	d.Slug = types.StringValue(environment.Slug)
	d.AutoStopSetting = types.StringValue(environment.AutoStopSetting)
	d.State = types.StringValue(environment.State)

	if environment.ClusterAgent == nil {
		d.ClusterAgentID = types.Int64Null()
	} else {
		d.ClusterAgentID = types.Int64Value(int64(environment.ClusterAgent.ID))
	}

	createdAt, diags := timetypes.NewRFC3339Value(environment.CreatedAt.Format(time.RFC3339))
	if diags != nil && diags.HasError() {
		return diags
	}
	d.CreatedAt = createdAt

	updatedAt, diags := timetypes.NewRFC3339Value(environment.UpdatedAt.Format(time.RFC3339))
	if diags != nil && diags.HasError() {
		return diags
	}
	d.UpdatedAt = updatedAt

	if environment.AutoStopAt == nil {
		d.AutoStopAt = timetypes.NewRFC3339Null()
	} else {
		autoStopAt, diags := timetypes.NewRFC3339Value(environment.AutoStopAt.Format(time.RFC3339))
		if diags != nil && diags.HasError() {
			return diags
		}
		d.AutoStopAt = autoStopAt
	}

	return nil
}

func resourceGitlabProjectEnvironmentParseID(id string) (string, int64, error) {
	project, rawEnvironmentID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}
	environmentID, err := strconv.ParseInt(rawEnvironmentID, 10, 64)
	if err != nil {
		return "", 0, err
	}
	return project, environmentID, nil
}
