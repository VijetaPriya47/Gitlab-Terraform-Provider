package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabInstanceVariableResource{}
	_ resource.ResourceWithConfigure   = &gitlabInstanceVariableResource{}
	_ resource.ResourceWithImportState = &gitlabInstanceVariableResource{}
)

func init() {
	registerResource(NewGitlabInstanceVariableResource)
}

func NewGitlabInstanceVariableResource() resource.Resource {
	return &gitlabInstanceVariableResource{}
}

type gitlabInstanceVariableResource struct {
	client *gitlab.Client
}

func (r *gitlabInstanceVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_variable"
}

type gitlabInstanceVariableResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Key          types.String `tfsdk:"key"`
	Value        types.String `tfsdk:"value"`
	Description  types.String `tfsdk:"description"`
	VariableType types.String `tfsdk:"variable_type"`
	Protected    types.Bool   `tfsdk:"protected"`
	Masked       types.Bool   `tfsdk:"masked"`
	Raw          types.Bool   `tfsdk:"raw"`
}

func (r *gitlabInstanceVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_instance_variable` + "`" + ` resource manages the lifecycle of an instance-level CI/CD variable.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/instance_level_ci_variables/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this terraform resource. In the format `<key>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The name of the variable.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(regexpGitlabVariableName, "is an invalid, only A-Z, a-z, 0-9, and _ are allowed"),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the variable.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the variable. Maximum of 255 characters.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(0, 255)},
			},
			"variable_type": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The type of a variable. Valid values are: %s. Default is `env_var`.", utils.RenderValueListForDocs(gitlabVariableTypeValues)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("env_var"),
				Validators:          []validator.String{stringvalidator.OneOf(gitlabVariableTypeValues...)},
			},
			"protected": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the variable will be passed only to pipelines running on protected branches and tags. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"masked": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the value of the variable will be hidden in job logs. The value must meet the [masking requirements](https://docs.gitlab.com/ci/variables/#mask-a-cicd-variable). Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"raw": schema.BoolAttribute{
				MarkdownDescription: "Whether the variable is treated as a raw string. Default: false. When true, variables in the value are not expanded.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *gitlabInstanceVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabInstanceVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabInstanceVariableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := data.Key.ValueString()
	variableType, ok := stringToVariableTypelookup[data.VariableType.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Invalid variable type", fmt.Sprintf("The variable type '%s' is invalid", data.VariableType.ValueString()))
		return
	}

	options := gitlab.CreateInstanceVariableOptions{
		Key:          &key,
		Value:        data.Value.ValueStringPointer(),
		Description:  data.Description.ValueStringPointer(),
		VariableType: &variableType,
		Protected:    data.Protected.ValueBoolPointer(),
		Masked:       data.Masked.ValueBoolPointer(),
		Raw:          data.Raw.ValueBoolPointer(),
	}
	tflog.Debug(ctx, fmt.Sprintf("create gitlab instance level CI variable %s", key))

	variable, _, err := r.client.InstanceVariables.CreateVariable(&options, gitlab.WithContext(ctx))
	if err != nil {
		if notOk, err := utils.AugmentVariableClientError(ctx, true, err); notOk {
			resp.Diagnostics.AddError(invalidMaskedValueSummary, err.Error())
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create instance variable: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(key)
	data.instanceVariableToStateModel(variable)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabInstanceVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabInstanceVariableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := data.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("read gitlab instance level CI variable %s", key))

	variable, _, err := r.client.InstanceVariables.GetVariable(key, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("gitlab instance level CI variable for %s not found so removing from state", key))
			resp.State.RemoveResource(ctx)
			return
		}
		if notOk, err := utils.AugmentVariableClientError(ctx, true, err); notOk {
			resp.Diagnostics.AddError(invalidMaskedValueSummary, err.Error())
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read instance variable: %s", err.Error()))
		return
	}

	data.instanceVariableToStateModel(variable)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabInstanceVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabInstanceVariableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := data.Key.ValueString()
	variableType, ok := stringToVariableTypelookup[data.VariableType.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Invalid variable type", fmt.Sprintf("The variable type '%s' is invalid", data.VariableType.ValueString()))
		return
	}

	options := &gitlab.UpdateInstanceVariableOptions{
		Value:        data.Value.ValueStringPointer(),
		Description:  data.Description.ValueStringPointer(),
		Protected:    data.Protected.ValueBoolPointer(),
		VariableType: &variableType,
		Masked:       data.Masked.ValueBoolPointer(),
		Raw:          data.Raw.ValueBoolPointer(),
	}
	tflog.Debug(ctx, fmt.Sprintf("update gitlab instance level CI variable %s", key))

	variable, _, err := r.client.InstanceVariables.UpdateVariable(key, options, gitlab.WithContext(ctx))
	if err != nil {
		if notOk, err := utils.AugmentVariableClientError(ctx, true, err); notOk {
			resp.Diagnostics.AddError(invalidMaskedValueSummary, err.Error())
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update instance variable: %s", err.Error()))
		return
	}
	data.instanceVariableToStateModel(variable)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabInstanceVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabInstanceVariableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := data.Key.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Delete gitlab instance level CI variable %s", key))

	_, err := r.client.InstanceVariables.RemoveVariable(key, gitlab.WithContext(ctx))
	if err != nil {
		if notOk, err := utils.AugmentVariableClientError(ctx, true, err); notOk {
			resp.Diagnostics.AddError(invalidMaskedValueSummary, err.Error())
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete instance variable: %s", err.Error()))
		return
	}
}

func (r *gitlabInstanceVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabInstanceVariableResourceModel) instanceVariableToStateModel(variable *gitlab.InstanceVariable) {
	d.Key = types.StringValue(variable.Key)
	d.Value = types.StringValue(variable.Value)
	d.Description = types.StringValue(variable.Description)
	d.VariableType = types.StringValue(string(variable.VariableType))
	d.Protected = types.BoolValue(variable.Protected)
	d.Masked = types.BoolValue(variable.Masked)
	d.Raw = types.BoolValue(variable.Raw)
}
