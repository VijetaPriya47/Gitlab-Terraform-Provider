package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
)

var (
	_ resource.Resource                = &gitlabApplicationAppearanceResource{}
	_ resource.ResourceWithConfigure   = &gitlabApplicationAppearanceResource{}
	_ resource.ResourceWithImportState = &gitlabApplicationAppearanceResource{}
)

func init() {
	registerResource(NewGitlabApplicationAppearanceResource)
}

func NewGitlabApplicationAppearanceResource() resource.Resource {
	return &gitlabApplicationAppearanceResource{}
}

type gitlabApplicationAppearanceResource struct {
	client *gitlab.Client
}

func (r *gitlabApplicationAppearanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_appearance"
}

func (r *gitlabApplicationAppearanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabApplicationAppearanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type gitlabApplicationAppearanceResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	KeepSettingsOnDestroy       types.Bool   `tfsdk:"keep_settings_on_destroy"`
	Title                       types.String `tfsdk:"title"`
	Description                 types.String `tfsdk:"description"`
	PWAName                     types.String `tfsdk:"pwa_name"`
	PWAShortName                types.String `tfsdk:"pwa_short_name"`
	PWADescription              types.String `tfsdk:"pwa_description"`
	MemberGuidelines            types.String `tfsdk:"member_guidelines"`
	NewProjectGuidelines        types.String `tfsdk:"new_project_guidelines"`
	ProfileImageGuidelines      types.String `tfsdk:"profile_image_guidelines"`
	HeaderMessage               types.String `tfsdk:"header_message"`
	FooterMessage               types.String `tfsdk:"footer_message"`
	MessageBackgroundColor      types.String `tfsdk:"message_background_color"`
	MessageFontColor            types.String `tfsdk:"message_font_color"`
	EmailHeaderAndFooterEnabled types.Bool   `tfsdk:"email_header_and_footer_enabled"`
}

func (r *gitlabApplicationAppearanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf(`The ` + "`" + `gitlab_application_appearance` + "`" + ` resource manages the GitLab application appearance.
		
~> This is an **experimental resource**. By nature it doesn't properly fit into how Terraform resources are meant to work.

~> All ` + "`" + `gitlab_application_appearance` + "`" + ` resources use the same ID ` + "`" + `gitlab` + "`" + `.

~> When you destroy the resource, you can control if appearance settings are saved or not. Set ` + "`" + `keep_settings_on_destroy` + "`" + ` to ` + "`" + `true` + "`" + ` (default) to save changes to appearance settings. Set ` + "`" + `keep_settings_on_destroy` + "`" + ` to ` + "`" + `false` + "`" + ` to reset the appearance to its original values.
The original values are saved in state when you create the resource. You can change the ` + "`" + `keep_settings_on_destroy` + "`" + ` value before destroying the resource to control this behavior.

-> Requires administrative privileges on GitLab.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/appearance/)`),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. Hard-coded to `gitlab`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"keep_settings_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Set to true if the appearance settings should not be reset to their pre-terraform defaults on destroy.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Application title on the sign-in and sign-up page.",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Markdown text shown on the sign-in and sign-up page.",
				Optional:            true,
				Computed:            true,
			},
			"pwa_name": schema.StringAttribute{
				MarkdownDescription: "Full name of the Progressive Web App. Used for the attribute `name` in `manifest.json`.",
				Optional:            true,
				Computed:            true,
			},
			"pwa_short_name": schema.StringAttribute{
				MarkdownDescription: "Short name for Progressive Web App.",
				Optional:            true,
				Computed:            true,
			},
			"pwa_description": schema.StringAttribute{
				MarkdownDescription: "An explanation of what the Progressive Web App does. Used for the attribute `description` in `manifest.json`.",
				Optional:            true,
				Computed:            true,
			},
			"member_guidelines": schema.StringAttribute{
				MarkdownDescription: "Markdown text shown on the group or project member page for users with permission to change members.",
				Optional:            true,
				Computed:            true,
			},
			"new_project_guidelines": schema.StringAttribute{
				MarkdownDescription: "Markdown text shown on the new project page.",
				Optional:            true,
				Computed:            true,
			},
			"profile_image_guidelines": schema.StringAttribute{
				MarkdownDescription: "Markdown text shown on the profile page below the Public Avatar.",
				Optional:            true,
				Computed:            true,
			},
			"header_message": schema.StringAttribute{
				MarkdownDescription: "Message in the system header bar.",
				Optional:            true,
				Computed:            true,
			},
			"footer_message": schema.StringAttribute{
				MarkdownDescription: "Message in the system footer bar.",
				Optional:            true,
				Computed:            true,
			},
			"message_background_color": schema.StringAttribute{
				MarkdownDescription: "Background color for the system header or footer bar, in CSS hex notation.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`), "value must be a valid color code"),
				},
			},
			"message_font_color": schema.StringAttribute{
				MarkdownDescription: "Font color for the system header or footer bar, in CSS hex notation.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`), "value must be a valid color code"),
				},
			},
			"email_header_and_footer_enabled": schema.BoolAttribute{
				MarkdownDescription: "Add header and footer to all outgoing emails if enabled.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *gitlabApplicationAppearanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabApplicationAppearanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.storeOriginalAppearance(ctx, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	appearance, err := r.changeAppearance(data, ctx)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to change application appearance: %s", err.Error()))
		return
	}

	data.appearanceToStateModel(appearance)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type gitlabApplicationAppearancePrivateStateModel struct {
	Title                       string `json:"title"`
	Description                 string `json:"description"`
	PWAName                     string `json:"pwa_name"`
	PWAShortName                string `json:"pwa_short_name"`
	PWADescription              string `json:"pwa_description"`
	MemberGuidelines            string `json:"member_guidelines"`
	NewProjectGuidelines        string `json:"new_project_guidelines"`
	ProfileImageGuidelines      string `json:"profile_image_guidelines"`
	HeaderMessage               string `json:"header_message"`
	FooterMessage               string `json:"footer_message"`
	MessageBackgroundColor      string `json:"message_background_color"`
	MessageFontColor            string `json:"message_font_color"`
	EmailHeaderAndFooterEnabled bool   `json:"email_header_and_footer_enabled"`
}

func (r *gitlabApplicationAppearanceResource) storeOriginalAppearance(ctx context.Context, resp *resource.CreateResponse) {
	oa, _, err := r.client.Appearance.GetAppearance(gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get current application appearance: %s", err.Error()))
		return
	}

	data := gitlabApplicationAppearancePrivateStateModel{
		Title:                       oa.Title,
		Description:                 oa.Description,
		PWAName:                     oa.PWAName,
		PWAShortName:                oa.PWAShortName,
		PWADescription:              oa.PWADescription,
		MemberGuidelines:            oa.MemberGuidelines,
		NewProjectGuidelines:        oa.NewProjectGuidelines,
		ProfileImageGuidelines:      oa.ProfileImageGuidelines,
		HeaderMessage:               oa.HeaderMessage,
		FooterMessage:               oa.FooterMessage,
		MessageBackgroundColor:      oa.MessageBackgroundColor,
		MessageFontColor:            oa.MessageFontColor,
		EmailHeaderAndFooterEnabled: oa.EmailHeaderAndFooterEnabled,
	}
	b, err := json.Marshal(data)
	if err != nil {
		resp.Diagnostics.AddError("Error marshalling appearance into json", fmt.Sprintf("Unable to marshall resource model into json: %s", err.Error()))
		return
	}

	diags := resp.Private.SetKey(ctx, "gitlab_application_appearance_original", b)
	resp.Diagnostics.Append(diags...)
}

func (r *gitlabApplicationAppearanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabApplicationAppearanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appearance, _, err := r.client.Appearance.GetAppearance(gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get application appearance: %s", err.Error()))
		return
	}

	data.appearanceToStateModel(appearance)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabApplicationAppearanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabApplicationAppearanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appearance, err := r.changeAppearance(data, ctx)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to change application appearance: %s", err.Error()))
		return
	}

	data.appearanceToStateModel(appearance)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabApplicationAppearanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabApplicationAppearanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.KeepSettingsOnDestroy.ValueBool() {
		original, diags := req.Private.GetKey(ctx, "gitlab_application_appearance_original")
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if original == nil {
			resp.Diagnostics.AddWarning("Could not reset the application appearance", "No appearance found to reset to")
			resp.State.RemoveResource(ctx)
			return
		}

		var a *gitlabApplicationAppearancePrivateStateModel
		err := json.Unmarshal(original, &a)
		if err != nil {
			resp.Diagnostics.AddError("Could not unmarshal the original appearance", err.Error())
			return
		}
		options := gitlab.ChangeAppearanceOptions{
			Title:                       gitlab.Ptr(a.Title),
			Description:                 gitlab.Ptr(a.Description),
			PWAName:                     gitlab.Ptr(a.PWAName),
			PWAShortName:                gitlab.Ptr(a.PWAShortName),
			PWADescription:              gitlab.Ptr(a.PWADescription),
			MemberGuidelines:            gitlab.Ptr(a.MemberGuidelines),
			NewProjectGuidelines:        gitlab.Ptr(a.NewProjectGuidelines),
			ProfileImageGuidelines:      gitlab.Ptr(a.ProfileImageGuidelines),
			HeaderMessage:               gitlab.Ptr(a.HeaderMessage),
			FooterMessage:               gitlab.Ptr(a.FooterMessage),
			MessageBackgroundColor:      gitlab.Ptr(a.MessageBackgroundColor),
			MessageFontColor:            gitlab.Ptr(a.MessageFontColor),
			EmailHeaderAndFooterEnabled: gitlab.Ptr(a.EmailHeaderAndFooterEnabled),
		}
		_, _, err = r.client.Appearance.ChangeAppearance(&options, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to reset application appearance: %s", err.Error()))
			return
		}
	}
	tflog.Debug(ctx, "[DEBUG] destroying the application appearance does not do anything.")
	resp.State.RemoveResource(ctx)
}

func (r *gitlabApplicationAppearanceResource) changeAppearance(data *gitlabApplicationAppearanceResourceModel, ctx context.Context) (*gitlab.Appearance, error) {
	options := gitlab.ChangeAppearanceOptions{}

	if !data.Title.IsNull() && !data.Title.IsUnknown() {
		options.Title = gitlab.Ptr(data.Title.ValueString())
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	if !data.PWAName.IsNull() && !data.PWAName.IsUnknown() {
		options.PWAName = gitlab.Ptr(data.PWAName.ValueString())
	}

	if !data.PWAShortName.IsNull() && !data.PWAShortName.IsUnknown() {
		options.PWAShortName = gitlab.Ptr(data.PWAShortName.ValueString())
	}

	if !data.PWADescription.IsNull() && !data.PWADescription.IsUnknown() {
		options.PWADescription = gitlab.Ptr(data.PWADescription.ValueString())
	}

	if !data.MemberGuidelines.IsNull() && !data.MemberGuidelines.IsUnknown() {
		options.MemberGuidelines = gitlab.Ptr(data.MemberGuidelines.ValueString())
	}

	if !data.NewProjectGuidelines.IsNull() && !data.NewProjectGuidelines.IsUnknown() {
		options.NewProjectGuidelines = gitlab.Ptr(data.NewProjectGuidelines.ValueString())
	}

	if !data.ProfileImageGuidelines.IsNull() && !data.ProfileImageGuidelines.IsUnknown() {
		options.ProfileImageGuidelines = gitlab.Ptr(data.ProfileImageGuidelines.ValueString())
	}

	if !data.HeaderMessage.IsNull() && !data.HeaderMessage.IsUnknown() {
		options.HeaderMessage = gitlab.Ptr(data.HeaderMessage.ValueString())
	}

	if !data.FooterMessage.IsNull() && !data.FooterMessage.IsUnknown() {
		options.FooterMessage = gitlab.Ptr(data.FooterMessage.ValueString())
	}

	if !data.MessageBackgroundColor.IsNull() && !data.MessageBackgroundColor.IsUnknown() {
		options.MessageBackgroundColor = gitlab.Ptr(data.MessageBackgroundColor.ValueString())
	}

	if !data.MessageFontColor.IsNull() && !data.MessageFontColor.IsUnknown() {
		options.MessageFontColor = gitlab.Ptr(data.MessageFontColor.ValueString())
	}

	if !data.EmailHeaderAndFooterEnabled.IsNull() && !data.EmailHeaderAndFooterEnabled.IsUnknown() {
		options.EmailHeaderAndFooterEnabled = gitlab.Ptr(data.EmailHeaderAndFooterEnabled.ValueBool())
	}

	appearance, _, err := r.client.Appearance.ChangeAppearance(&options, gitlab.WithContext(ctx))
	return appearance, err
}

func (d *gitlabApplicationAppearanceResourceModel) appearanceToStateModel(appearance *gitlab.Appearance) {
	d.ID = types.StringValue("gitlab")
	d.Title = types.StringValue(appearance.Title)
	d.Description = types.StringValue(appearance.Description)
	d.PWAName = types.StringValue(appearance.PWAName)
	d.PWAShortName = types.StringValue(appearance.PWAShortName)
	d.PWADescription = types.StringValue(appearance.PWADescription)
	d.MemberGuidelines = types.StringValue(appearance.MemberGuidelines)
	d.NewProjectGuidelines = types.StringValue(appearance.NewProjectGuidelines)
	d.ProfileImageGuidelines = types.StringValue(appearance.ProfileImageGuidelines)
	d.HeaderMessage = types.StringValue(appearance.HeaderMessage)
	d.FooterMessage = types.StringValue(appearance.FooterMessage)
	d.MessageBackgroundColor = types.StringValue(appearance.MessageBackgroundColor)
	d.MessageFontColor = types.StringValue(appearance.MessageFontColor)
	d.EmailHeaderAndFooterEnabled = types.BoolValue(appearance.EmailHeaderAndFooterEnabled)
}
