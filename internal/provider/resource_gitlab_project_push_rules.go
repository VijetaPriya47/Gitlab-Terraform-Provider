package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabProjectPushRulesResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectPushRulesResource{}
	_ resource.ResourceWithImportState = &gitlabProjectPushRulesResource{}
)

func init() {
	registerResource(NewGitLabProjectPushRulesResource)
}

func NewGitLabProjectPushRulesResource() resource.Resource {
	return &gitlabProjectPushRulesResource{}
}

type gitlabProjectPushRulesResource struct {
	client *gitlab.Client
}

type gitlabProjectPushRulesResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	Project                    types.String `tfsdk:"project"`
	CommitMessageRegex         types.String `tfsdk:"commit_message_regex"`
	CommitMessageNegativeRegex types.String `tfsdk:"commit_message_negative_regex"`
	BranchNameRegex            types.String `tfsdk:"branch_name_regex"`
	DenyDeleteTag              types.Bool   `tfsdk:"deny_delete_tag"`
	MemberCheck                types.Bool   `tfsdk:"member_check"`
	PreventSecrets             types.Bool   `tfsdk:"prevent_secrets"`
	AuthorEmailRegex           types.String `tfsdk:"author_email_regex"`
	FileNameRegex              types.String `tfsdk:"file_name_regex"`
	MaxFileSize                types.Int64  `tfsdk:"max_file_size"`
	CommitCommitterCheck       types.Bool   `tfsdk:"commit_committer_check"`
	CommitCommitterNameCheck   types.Bool   `tfsdk:"commit_committer_name_check"`
	RejectUnsignedCommits      types.Bool   `tfsdk:"reject_unsigned_commits"`
	RejectNonDCOCommits        types.Bool   `tfsdk:"reject_non_dco_commits"`
}

// Metadata returns the resource name
func (d *gitlabProjectPushRulesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_push_rules"
}

func (r *gitlabProjectPushRulesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_push_rules`" + ` resource allows to manage the lifecycle of push rules on a project.

~> This resource will compete with the ` + "`gitlab_project`" + ` resource if push rules are also defined as 
   part of that resource, since this resource will take over ownership of the project push rules created for the referenced project.
   It is recommended to define push rules using this resource OR in the ` + "`gitlab_project`" + ` resource, 
   but not in both as it may result in terraform identifying changes with every "plan" operation.

-> This resource requires a GitLab Enterprise instance with a Premium license to set the push rules on a project.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/project_push_rules/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"author_email_regex": schema.StringAttribute{
				MarkdownDescription: "All commit author emails must match this regex, e.g. `@my-company.com$`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"branch_name_regex": schema.StringAttribute{
				MarkdownDescription: "All branch names must match this regex, e.g. `(feature|hotfix)\\/*`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"commit_committer_check": schema.BoolAttribute{
				MarkdownDescription: "Users can only push commits to this repository that were committed with one of their own verified emails.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"commit_committer_name_check": schema.BoolAttribute{
				MarkdownDescription: "Users can only push commits to this repository if the commit author name is consistent with their GitLab account name.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"commit_message_negative_regex": schema.StringAttribute{
				MarkdownDescription: "No commit message is allowed to match this regex, e.g. `ssh\\:\\/\\/`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"commit_message_regex": schema.StringAttribute{
				MarkdownDescription: "All commit messages must match this regex, e.g. `Fixed \\d+\\..*`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deny_delete_tag": schema.BoolAttribute{
				MarkdownDescription: "Deny deleting a tag.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"file_name_regex": schema.StringAttribute{
				MarkdownDescription: "All committed filenames must not match this regex, e.g. `(jar|exe)$`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"max_file_size": schema.Int64Attribute{
				MarkdownDescription: "Maximum file size (MB).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{int64validator.AtLeast(0)},
			},
			"member_check": schema.BoolAttribute{
				MarkdownDescription: "Restrict commits by author (email) to existing GitLab users.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"prevent_secrets": schema.BoolAttribute{
				MarkdownDescription: "GitLab will reject any files that are likely to contain secrets.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"reject_unsigned_commits": schema.BoolAttribute{
				MarkdownDescription: "Reject commit when it’s not signed.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"reject_non_dco_commits": schema.BoolAttribute{
				MarkdownDescription: "Reject commit when it’s not DCO certified.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectPushRulesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectPushRulesResource) projectPushRulesToStateModel(projectID string, pushRules *gitlab.ProjectPushRules, data *gitlabProjectPushRulesResourceModel) {
	data.ID = types.StringValue(projectID)
	data.Project = types.StringValue(projectID)
	data.AuthorEmailRegex = types.StringValue(pushRules.AuthorEmailRegex)
	data.BranchNameRegex = types.StringValue(pushRules.BranchNameRegex)
	data.CommitCommitterCheck = types.BoolValue(pushRules.CommitCommitterCheck)
	data.CommitCommitterNameCheck = types.BoolValue(pushRules.CommitCommitterNameCheck)
	data.CommitMessageNegativeRegex = types.StringValue(pushRules.CommitMessageNegativeRegex)
	data.CommitMessageRegex = types.StringValue(pushRules.CommitMessageRegex)
	data.DenyDeleteTag = types.BoolValue(pushRules.DenyDeleteTag)
	data.FileNameRegex = types.StringValue(pushRules.FileNameRegex)
	data.MaxFileSize = types.Int64Value(int64(pushRules.MaxFileSize))
	data.MemberCheck = types.BoolValue(pushRules.MemberCheck)
	data.PreventSecrets = types.BoolValue(pushRules.PreventSecrets)
	data.RejectUnsignedCommits = types.BoolValue(pushRules.RejectUnsignedCommits)
	data.RejectNonDCOCommits = types.BoolValue(pushRules.RejectNonDCOCommits)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabProjectPushRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectPushRulesResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab project %q push rules", projectID))

	pushRules, _, err := r.client.Projects.GetProjectPushRules(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "push rules for project do not exist, removing resource from state", map[string]any{
				"project": projectID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project push rules details: %s", err.Error()))
		return
	}

	// persist API response in state model
	r.projectPushRulesToStateModel(projectID, pushRules, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create creates a new upstream resource and adds it into the Terraform state.
func (r *gitlabProjectPushRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectPushRulesResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Project.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab project %q push rules", projectID))

	// check for existing push rules; if found then update, otherwise add new
	existingPushRules, _, err := r.client.Projects.GetProjectPushRules(projectID, gitlab.WithContext(ctx))
	if err == nil && existingPushRules.ID != 0 {
		// push rules exist, update them
		err := r.update(ctx, data, &resp.Diagnostics)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update project push rules", err.Error())
			return
		}
	} else {
		// add new
		options := gitlab.AddProjectPushRuleOptions{}

		if !data.AuthorEmailRegex.IsNull() && !data.AuthorEmailRegex.IsUnknown() {
			options.AuthorEmailRegex = data.AuthorEmailRegex.ValueStringPointer()
		}

		if !data.BranchNameRegex.IsNull() && !data.BranchNameRegex.IsUnknown() {
			options.BranchNameRegex = data.BranchNameRegex.ValueStringPointer()
		}

		if !data.CommitCommitterCheck.IsNull() && !data.CommitCommitterCheck.IsUnknown() {
			options.CommitCommitterCheck = data.CommitCommitterCheck.ValueBoolPointer()
		}

		if !data.CommitCommitterNameCheck.IsNull() && !data.CommitCommitterNameCheck.IsUnknown() {
			options.CommitCommitterNameCheck = data.CommitCommitterNameCheck.ValueBoolPointer()
		}

		if !data.CommitMessageNegativeRegex.IsNull() && !data.CommitMessageNegativeRegex.IsUnknown() {
			options.CommitMessageNegativeRegex = data.CommitMessageNegativeRegex.ValueStringPointer()
		}

		if !data.CommitMessageRegex.IsNull() && !data.CommitMessageRegex.IsUnknown() {
			options.CommitMessageRegex = data.CommitMessageRegex.ValueStringPointer()
		}

		if !data.DenyDeleteTag.IsNull() && !data.DenyDeleteTag.IsUnknown() {
			options.DenyDeleteTag = data.DenyDeleteTag.ValueBoolPointer()
		}

		if !data.FileNameRegex.IsNull() && !data.FileNameRegex.IsUnknown() {
			options.FileNameRegex = data.FileNameRegex.ValueStringPointer()
		}

		if !data.MaxFileSize.IsNull() && !data.MaxFileSize.IsUnknown() {
			options.MaxFileSize = gitlab.Ptr(data.MaxFileSize.ValueInt64())
		}

		if !data.MemberCheck.IsNull() && !data.MemberCheck.IsUnknown() {
			options.MemberCheck = data.MemberCheck.ValueBoolPointer()
		}

		if !data.PreventSecrets.IsNull() && !data.PreventSecrets.IsUnknown() {
			options.PreventSecrets = data.PreventSecrets.ValueBoolPointer()
		}

		if !data.RejectUnsignedCommits.IsNull() && !data.RejectUnsignedCommits.IsUnknown() {
			options.RejectUnsignedCommits = data.RejectUnsignedCommits.ValueBoolPointer()
		}

		if !data.RejectNonDCOCommits.IsNull() && !data.RejectNonDCOCommits.IsUnknown() {
			options.RejectNonDCOCommits = data.RejectNonDCOCommits.ValueBoolPointer()
		}

		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Creating new push rules for project %q", projectID))
		pushRules, _, err := r.client.Projects.AddProjectPushRule(projectID, &options, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to add project push rules details: %s", err.Error()))
			return
		}

		// persist API response in state model
		r.projectPushRulesToStateModel(projectID, pushRules, data)

	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabProjectPushRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectPushRulesResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.update(ctx, data, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update project push rules", err.Error())
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the resource.
func (r *gitlabProjectPushRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectPushRulesResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Deleting push rules for project %q", projectID))
	_, err := r.client.Projects.DeleteProjectPushRule(projectID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete project push rules details: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectPushRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Update existing push rules on a project
func (r *gitlabProjectPushRulesResource) update(ctx context.Context, data *gitlabProjectPushRulesResourceModel, diags *diag.Diagnostics) error {
	projectID := data.Project.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab project %q push rules", projectID))

	options := gitlab.EditProjectPushRuleOptions{}

	if !data.AuthorEmailRegex.IsNull() && !data.AuthorEmailRegex.IsUnknown() {
		options.AuthorEmailRegex = data.AuthorEmailRegex.ValueStringPointer()
	}

	if !data.BranchNameRegex.IsNull() && !data.BranchNameRegex.IsUnknown() {
		options.BranchNameRegex = data.BranchNameRegex.ValueStringPointer()
	}

	if !data.CommitCommitterCheck.IsNull() && !data.CommitCommitterCheck.IsUnknown() {
		options.CommitCommitterCheck = data.CommitCommitterCheck.ValueBoolPointer()
	}

	if !data.CommitCommitterNameCheck.IsNull() && !data.CommitCommitterNameCheck.IsUnknown() {
		options.CommitCommitterNameCheck = data.CommitCommitterNameCheck.ValueBoolPointer()
	}

	if !data.CommitMessageNegativeRegex.IsNull() && !data.CommitMessageNegativeRegex.IsUnknown() {
		options.CommitMessageNegativeRegex = data.CommitMessageNegativeRegex.ValueStringPointer()
	}

	if !data.CommitMessageRegex.IsNull() && !data.CommitMessageRegex.IsUnknown() {
		options.CommitMessageRegex = data.CommitMessageRegex.ValueStringPointer()
	}

	if !data.DenyDeleteTag.IsNull() && !data.DenyDeleteTag.IsUnknown() {
		options.DenyDeleteTag = data.DenyDeleteTag.ValueBoolPointer()
	}

	if !data.FileNameRegex.IsNull() && !data.FileNameRegex.IsUnknown() {
		options.FileNameRegex = data.FileNameRegex.ValueStringPointer()
	}

	if !data.MaxFileSize.IsNull() && !data.MaxFileSize.IsUnknown() {
		options.MaxFileSize = gitlab.Ptr(data.MaxFileSize.ValueInt64())
	}

	if !data.MemberCheck.IsNull() && !data.MemberCheck.IsUnknown() {
		options.MemberCheck = data.MemberCheck.ValueBoolPointer()
	}

	if !data.PreventSecrets.IsNull() && !data.PreventSecrets.IsUnknown() {
		options.PreventSecrets = data.PreventSecrets.ValueBoolPointer()
	}

	if !data.RejectUnsignedCommits.IsNull() && !data.RejectUnsignedCommits.IsUnknown() {
		options.RejectUnsignedCommits = data.RejectUnsignedCommits.ValueBoolPointer()
	}

	if !data.RejectNonDCOCommits.IsNull() && !data.RejectNonDCOCommits.IsUnknown() {
		options.RejectNonDCOCommits = data.RejectNonDCOCommits.ValueBoolPointer()
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Updating push rules for project %q", projectID))
	pushRules, _, err := r.client.Projects.EditProjectPushRule(projectID, &options, gitlab.WithContext(ctx))
	if err != nil {
		diags.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update project push rules details: %s", err.Error()))
		return err
	}

	// persist API response in state model
	r.projectPushRulesToStateModel(projectID, pushRules, data)

	return nil
}
