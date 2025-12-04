package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabDeployKeyEnableResource{}
	_ resource.ResourceWithConfigure   = &gitlabDeployKeyEnableResource{}
	_ resource.ResourceWithImportState = &gitlabDeployKeyEnableResource{}
)

func init() {
	registerResource(NewGitlabDeployKeyEnableResource)
}

func NewGitlabDeployKeyEnableResource() resource.Resource {
	return &gitlabDeployKeyEnableResource{}
}

type gitlabDeployKeyEnableResource struct {
	client *gitlab.Client
}

type gitlabDeployKeyEnableResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Project types.String `tfsdk:"project"`
	KeyID   types.String `tfsdk:"key_id"`
	Title   types.String `tfsdk:"title"`
	Key     types.String `tfsdk:"key"`
	CanPush types.Bool   `tfsdk:"can_push"`
}

func (r *gitlabDeployKeyEnableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deploy_key_enable"
}

func (r *gitlabDeployKeyEnableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_deploy_key_enable`" + ` resource allows to enable an already existing deploy key (see ` + "`gitlab_deploy_key resource`" + `) for a specific project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/deploy_keys/#enable-a-deploy-key)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this terraform resource. In the format `<project:key-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project to add the deploy key to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key_id": schema.StringAttribute{
				MarkdownDescription: "The Gitlab key id for the pre-existing deploy key",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Deploy key's title.",
				Computed:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Deploy key.",
				Computed:            true,
			},
			"can_push": schema.BoolAttribute{
				MarkdownDescription: "Can deploy key push to the project's repository.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *gitlabDeployKeyEnableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabDeployKeyEnableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabDeployKeyEnableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabDeployKeyEnableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()

	keyID, err := strconv.ParseInt(data.KeyID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing key ID", fmt.Sprintf("Could not parse key ID %q to int: %s", data.KeyID.ValueString(), err))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("enable gitlab deploy key %s/%d", project, keyID))

	_, _, err = r.client.DeployKeys.EnableDeployKey(project, keyID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to enable deploy key: %s", err.Error()))
		return
	}

	options := &gitlab.UpdateDeployKeyOptions{
		CanPush: data.CanPush.ValueBoolPointer(),
	}

	tflog.Debug(ctx, fmt.Sprintf("update gitlab deploy key %s/%d", project, keyID))

	deployKey, _, err := r.client.DeployKeys.UpdateDeployKey(project, keyID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update deploy key: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%d", project, deployKey.ID))
	data.Title = types.StringValue(deployKey.Title)
	data.KeyID = types.StringValue(strconv.FormatInt(deployKey.ID, 10))
	data.Key = types.StringValue(deployKey.Key)
	data.CanPush = types.BoolValue(deployKey.CanPush)
	data.Project = types.StringValue(project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabDeployKeyEnableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabDeployKeyEnableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, deployKeyID, err := resourceGitLabDeployKeyEnableParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Read. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read gitlab deploy key %s/%d", project, deployKeyID))

	deployKey, _, err := r.client.DeployKeys.GetDeployKey(project, deployKeyID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("gitlab deploy key not found %s/%d", project, deployKeyID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read deploy key: %s", err.Error()))
		return
	}

	data.Title = types.StringValue(deployKey.Title)
	data.KeyID = types.StringValue(strconv.FormatInt(deployKey.ID, 10))
	data.Key = types.StringValue(deployKey.Key)
	data.CanPush = types.BoolValue(deployKey.CanPush)
	data.Project = types.StringValue(project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabDeployKeyEnableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Provider Error, report upstream",
		"Somehow the resource was requested to perform an in-place upgrade which is not possible.",
	)
}

func (r *gitlabDeployKeyEnableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabDeployKeyEnableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, deployKeyID, err := resourceGitLabDeployKeyEnableParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Delete. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Delete gitlab deploy key %s/%d", project, deployKeyID))

	_, err = r.client.DeployKeys.DeleteDeployKey(project, deployKeyID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete deploy key: %s", err.Error()))
		return
	}
}

func resourceGitLabDeployKeyEnableParseId(id string) (string, int64, error) {
	projectID, deployTokenID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	deployTokenIID, err := strconv.ParseInt(deployTokenID, 10, 64)
	if err != nil {
		return "", 0, err
	}

	return projectID, deployTokenIID, nil
}
