package provider

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
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

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                   = &gitlabGroupVariableResource{}
	_ resource.ResourceWithConfigure      = &gitlabGroupVariableResource{}
	_ resource.ResourceWithImportState    = &gitlabGroupVariableResource{}
	_ resource.ResourceWithValidateConfig = &gitlabGroupVariableResource{}
)

var (
	gitlabVariableTypeValues   = []string{"env_var", "file"}
	stringToVariableTypelookup = map[string]gitlab.VariableTypeValue{"env_var": gitlab.EnvVariableType, "file": gitlab.FileVariableType}
	regexpGitlabVariableName   = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	invalidMaskedValueSummary  = "Invalid value for a masked variable. Check the masked variable requirements: https://docs.gitlab.com/ci/variables/#mask-a-cicd-variable"
)

func init() {
	registerResource(NewGitlabGroupGroupVariableResource)
}

// NewGitlabGroupGroupVariableResource is a helper function to simplify the provider implementation.
func NewGitlabGroupGroupVariableResource() resource.Resource {
	return &gitlabGroupVariableResource{}
}

// gitlabGroupVariableResource defines the resource implementation.
type gitlabGroupVariableResource struct {
	client *gitlab.Client
}

func (r *gitlabGroupVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_variable"
}

// Struct for the schema
type gitlabGroupVariableResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Group            types.String `tfsdk:"group"`
	Key              types.String `tfsdk:"key"`
	Value            types.String `tfsdk:"value"`
	VariableType     types.String `tfsdk:"variable_type"`
	Protected        types.Bool   `tfsdk:"protected"`
	Masked           types.Bool   `tfsdk:"masked"`
	Hidden           types.Bool   `tfsdk:"hidden"`
	EnvironmentScope types.String `tfsdk:"environment_scope"`
	Raw              types.Bool   `tfsdk:"raw"`
	Description      types.String `tfsdk:"description"`
}

func (r *gitlabGroupVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_variable`" + ` resource allows creating a GitLab group level variables.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_level_variables/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group>:<key>:<environment_scope>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The name or id of the group.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The name of the variable.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(regexpGitlabVariableName, "is an invalid, only A-Z, a-z, 0-9, and _ are allowed"),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the variable.",
				Required:            true,
			},
			"variable_type": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The type of a variable. Valid values are: %s.", utils.RenderValueListForDocs(gitlabVariableTypeValues)),
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(gitlabVariableTypeValues...)},
			},
			"protected": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the variable will be passed only to pipelines running on protected branches and tags.",
				Optional:            true,
				Computed:            true,
			},
			"masked": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the value of the variable will be masked in job logs. The value must meet the [masking requirements](https://docs.gitlab.com/ci/variables/#mask-a-cicd-variable).",
				Optional:            true,
				Computed:            true,
			},
			"hidden": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the value of the variable will be hidden in the CI/CD User Interface. The value must meet the [hidden requirements](https://docs.gitlab.com/ci/variables/#hide-a-cicd-variable).",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplaceIfConfigured()},
				Optional:            true,
				Computed:            true,
				Validators: []validator.Bool{
					boolvalidator.AlsoRequires(path.MatchRoot("masked")),
				},
			},
			"environment_scope": schema.StringAttribute{
				MarkdownDescription: "The environment scope of the variable. Defaults to all environment (`*`). Note that in Community Editions of Gitlab, values other than `*` will cause inconsistent plans.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("*"),
			},
			"raw": schema.BoolAttribute{
				MarkdownDescription: "Whether the variable is treated as a raw string. When true, variables in the value are not expanded.",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the variable.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabGroupVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (d *gitlabGroupVariableResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data gitlabGroupVariableResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if !data.Hidden.IsNull() && !data.Hidden.IsUnknown() && data.Hidden.ValueBool() {
		if data.Masked.IsNull() || data.Masked.IsUnknown() || !data.Masked.ValueBool() {
			resp.Diagnostics.AddAttributeError(path.Root("hidden"),
				`Invalid value for a masked variable`,
				`A variable cannot be hidden without being masked. Please set the masked attribute to true`)
			return
		}
	}
}

// Create implements resource.Resource.
func (r *gitlabGroupVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupVariableResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	group := data.Group.ValueString()
	key := data.Key.ValueString()

	options := gitlab.CreateGroupVariableOptions{
		Key:         gitlab.Ptr(data.Key.ValueString()),
		Value:       gitlab.Ptr(data.Value.ValueString()),
		Masked:      gitlab.Ptr(data.Masked.ValueBool()),
		Raw:         gitlab.Ptr(data.Raw.ValueBool()),
		Description: gitlab.Ptr(data.Description.ValueString()),

		// Technically optional, but set here since it has a default.
		EnvironmentScope: gitlab.Ptr(data.EnvironmentScope.ValueString()),
	}
	if !data.Protected.IsNull() && !data.Protected.IsUnknown() {
		options.Protected = gitlab.Ptr(data.Protected.ValueBool())
	}
	if !data.VariableType.IsNull() && !data.VariableType.IsUnknown() {
		variableType, ok := stringToVariableTypelookup[data.VariableType.ValueString()]
		if !ok {
			resp.Diagnostics.AddError("Invalid variable type", fmt.Sprintf("The variable type '%s' is invalid", data.VariableType.ValueString()))
			return
		}
		options.VariableType = &variableType
	}
	if !data.Masked.IsNull() && !data.Masked.IsUnknown() {
		options.Masked = gitlab.Ptr(data.Masked.ValueBool())
	}

	if !data.Hidden.IsNull() && !data.Hidden.IsUnknown() && data.Hidden.ValueBool() {
		if data.Masked.IsNull() || data.Masked.IsUnknown() || !data.Masked.ValueBool() {
			resp.Diagnostics.AddError("Invalid value for a masked variable", "Error: this is not expected to happen")
			return
		}
		options.Masked = gitlab.Ptr(false)
		options.MaskedAndHidden = gitlab.Ptr(data.Hidden.ValueBool())
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab group variable %s/%s", group, key))

	variable, _, err := r.client.GroupVariables.CreateVariable(group, &options, gitlab.WithContext(ctx))
	if err != nil {
		if notOk, err := utils.AugmentVariableClientError(ctx, true, err); notOk {
			resp.Diagnostics.AddError(invalidMaskedValueSummary, err.Error())
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create group variable: %s", err.Error()))
		return
	}

	keyScope := fmt.Sprintf("%s:%s", key, data.EnvironmentScope.ValueString())
	data.ID = types.StringValue(utils.BuildTwoPartID(&group, &keyScope))
	data.groupVariableToStateModel(variable, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read implements resource.Resource.
func (r *gitlabGroupVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupVariableResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	group, key, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Read. It should be '<group>:<key>:<environment_scope>'. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	keyScope := strings.SplitN(key, ":", 2)
	scope := "*"
	if len(keyScope) == 2 {
		key = keyScope[0]
		scope = keyScope[1]
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab group variable %s/%s/%s", group, key, scope))

	variable, _, err := r.client.GroupVariables.GetVariable(
		group,
		key,
		nil,
		gitlab.WithContext(ctx),
		utils.WithEnvironmentScopeFilter(ctx, scope),
	)
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab group variable not found %s/%s, removing from state", group, key))
			resp.State.RemoveResource(ctx)
			return
		}
		if notOk, err := utils.AugmentVariableClientError(ctx, true, err); notOk {
			resp.Diagnostics.AddError(invalidMaskedValueSummary, err.Error())
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read group variable: %s", err.Error()))
		return
	}

	data.groupVariableToStateModel(variable, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update implements resource.Resource.
func (r *gitlabGroupVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabGroupVariableResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	group := data.Group.ValueString()
	key := data.Key.ValueString()
	options := &gitlab.UpdateGroupVariableOptions{
		Value:       gitlab.Ptr(data.Value.ValueString()),
		Masked:      gitlab.Ptr(data.Masked.ValueBool()),
		Raw:         gitlab.Ptr(data.Raw.ValueBool()),
		Description: gitlab.Ptr(data.Description.ValueString()),

		// Technically optional, but set here since it has a default.
		EnvironmentScope: gitlab.Ptr(data.EnvironmentScope.ValueString()),
	}
	if !data.Protected.IsNull() && !data.Protected.IsUnknown() {
		options.Protected = gitlab.Ptr(data.Protected.ValueBool())
	}
	if !data.VariableType.IsNull() && !data.VariableType.IsUnknown() {
		variableType, ok := stringToVariableTypelookup[data.VariableType.ValueString()]
		if !ok {
			resp.Diagnostics.AddError("Invalid variable type", fmt.Sprintf("The variable type '%s' is invalid", data.VariableType.ValueString()))
			return
		}
		options.VariableType = &variableType
	}
	if !data.Masked.IsNull() && !data.Masked.IsUnknown() {
		options.Masked = gitlab.Ptr(data.Masked.ValueBool())
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab group variable %s/%s", group, key))

	variable, _, err := r.client.GroupVariables.UpdateVariable(
		group,
		key,
		options,
		gitlab.WithContext(ctx),
		utils.WithEnvironmentScopeFilter(ctx, data.EnvironmentScope.ValueString()),
	)
	if err != nil {
		if notOk, err := utils.AugmentVariableClientError(ctx, true, err); notOk {
			resp.Diagnostics.AddError(invalidMaskedValueSummary, err.Error())
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update group variable: %s", err.Error()))
		return
	}
	data.groupVariableToStateModel(variable, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (r *gitlabGroupVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupVariableResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	group := data.Group.ValueString()
	key := data.Key.ValueString()
	environmentScope := data.EnvironmentScope.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab group variable %s/%s/%s", group, key, environmentScope))

	_, err := r.client.GroupVariables.RemoveVariable(
		group,
		key,
		&gitlab.RemoveGroupVariableOptions{
			Filter: &gitlab.VariableFilter{
				EnvironmentScope: environmentScope,
			},
		},
		gitlab.WithContext(ctx),
	)
	if err != nil {
		if api.Is404(err) {
			slog.Debug("The variable was not found, assuming deleted. If the access token doesn't have permissions to view the resource, re-importing will be required to delete the resource.")
			return
		}

		if notOk, err := utils.AugmentVariableClientError(ctx, true, err); notOk {
			resp.Diagnostics.AddError(invalidMaskedValueSummary, err.Error())
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete group variable: %s", err.Error()))
		return
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabGroupVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *gitlabGroupVariableResourceModel) groupVariableToStateModel(variable *gitlab.GroupVariable, group string) {
	// attributes from api response
	keyScope := fmt.Sprintf("%s:%s", variable.Key, variable.EnvironmentScope)
	m.ID = types.StringValue(utils.BuildTwoPartID(&group, &keyScope))
	m.Group = types.StringValue(group)
	m.Key = types.StringValue(variable.Key)
	if !variable.Hidden {
		// API response for hidden group variables is always null
		// this condition allows the value to be maintained in terraform state
		m.Value = types.StringValue(variable.Value)
	}
	m.VariableType = types.StringValue(string(variable.VariableType))
	m.Protected = types.BoolValue(variable.Protected)
	m.Masked = types.BoolValue(variable.Masked)
	m.Hidden = types.BoolValue(variable.Hidden)
	m.EnvironmentScope = types.StringValue(variable.EnvironmentScope)
	m.Raw = types.BoolValue(variable.Raw)
	m.Description = types.StringValue(variable.Description)
}
