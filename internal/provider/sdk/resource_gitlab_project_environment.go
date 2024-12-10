package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_project_environment", func() *schema.Resource {
	allowedEnvironmentTiers := []string{"production", "staging", "testing", "development", "other"}

	return &schema.Resource{
		Description: `The ` + "`gitlab_project_environment`" + ` resource allows to manage the lifecycle of an environment in a project.

-> During a terraform destroy this resource by default will not attempt to stop the environment first.
An environment is required to be in a stopped state before a deletetion of the environment can occur.
Set the ` + "`stop_before_destroy`" + ` flag to attempt to automatically stop the environment before deletion.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/environments.html)`,

		CreateContext: resourceGitlabProjectEnvironmentCreate,
		ReadContext:   resourceGitlabProjectEnvironmentRead,
		UpdateContext: resourceGitlabProjectEnvironmentUpdate,
		DeleteContext: resourceGitlabProjectEnvironmentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"project": {
				Description:  "The ID or full path of the project to environment is created for.",
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"name": {
				Description:  "The name of the environment.",
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"external_url": {
				Description:  "Place to link to for this environment.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"tier": {
				Description:      fmt.Sprintf("The tier of the new environment. Valid values are %s.", utils.RenderValueListForDocs(allowedEnvironmentTiers)),
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(allowedEnvironmentTiers, false)),
			},
			"cluster_agent_id": {
				Description: "The cluster agent to associate with this environment.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"kubernetes_namespace": {
				Description:  "The Kubernetes namespace to associate with this environment.",
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"cluster_agent_id"},
			},
			"flux_resource_path": {
				Description:  "The Flux resource path to associate with this environment.",
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"cluster_agent_id", "kubernetes_namespace"},
			},
			"slug": {
				Description: "The name of the environment in lowercase, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"created_at": {
				Description: "The ISO8601 date/time that this environment was created at in UTC.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"updated_at": {
				Description: "The ISO8601 date/time that this environment was last updated at in UTC.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"state": {
				Description: fmt.Sprintf("State the environment is in. Valid values are %s.", utils.RenderValueListForDocs(api.ValidProjectEnvironmentStates)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"stop_before_destroy": {
				Description: "Determines whether the environment is attempted to be stopped before the environment is deleted.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
})

func resourceGitlabProjectEnvironmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	options := gitlab.CreateEnvironmentOptions{
		Name: &name,
	}
	if externalURL, ok := d.GetOk("external_url"); ok {
		options.ExternalURL = gitlab.Ptr(externalURL.(string))
	}
	if tier, ok := d.GetOk("tier"); ok {
		options.Tier = gitlab.Ptr(tier.(string))
	}
	if clusterAgentID, ok := d.GetOk("cluster_agent_id"); ok {
		options.ClusterAgentID = gitlab.Ptr(clusterAgentID.(int))
	}
	if kubernetesNamespace, ok := d.GetOk("kubernetes_namespace"); ok {
		options.KubernetesNamespace = gitlab.Ptr(kubernetesNamespace.(string))
	}
	if fluxResourcePath, ok := d.GetOk("flux_resource_path"); ok {
		options.FluxResourcePath = gitlab.Ptr(fluxResourcePath.(string))
	}

	project := d.Get("project").(string)

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Project %s create gitlab environment %q", project, *options.Name))

	client := meta.(*gitlab.Client)

	environment, _, err := client.Environments.CreateEnvironment(project, &options, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			return diag.Errorf("feature Environments is not available")
		}
		return diag.FromErr(err)
	}

	environmentID := fmt.Sprintf("%d", environment.ID)
	d.SetId(utils.BuildTwoPartID(&project, &environmentID))
	return resourceGitlabProjectEnvironmentRead(ctx, d, meta)
}

func resourceGitlabProjectEnvironmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab environment %s", d.Id()))

	project, environmentID, err := resourceGitlabProjectEnvironmentParseID(ctx, d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Project %s read gitlab environment %d", project, environmentID))

	client := meta.(*gitlab.Client)

	environment, _, err := client.Environments.GetEnvironment(project, environmentID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Project %s gitlab environment %d not found, removing from state", project, environmentID))
			d.SetId("")
			return nil
		}
		return diag.Errorf("error getting gitlab project %s environment %d: %v", project, environmentID, err)
	}

	d.Set("project", project)
	d.Set("name", environment.Name)
	d.Set("state", environment.State)
	d.Set("external_url", environment.ExternalURL)
	d.Set("tier", environment.Tier)
	if environment.ClusterAgent != nil {
		d.Set("cluster_agent_id", environment.ClusterAgent.ID)
	} else {
		d.Set("cluster_agent_id", nil)
	}
	d.Set("kubernetes_namespace", environment.KubernetesNamespace)
	d.Set("flux_resource_path", environment.FluxResourcePath)
	d.Set("created_at", environment.CreatedAt.Format(time.RFC3339))
	if environment.UpdatedAt != nil {
		d.Set("updated_at", environment.UpdatedAt.Format(time.RFC3339))
	}

	return nil
}

func resourceGitlabProjectEnvironmentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab environment %s", d.Id()))

	project, environmentID, err := resourceGitlabProjectEnvironmentParseID(ctx, d)
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.EditEnvironmentOptions{
		Name: gitlab.Ptr(d.Get("name").(string)),
	}

	if d.HasChange("external_url") {
		options.ExternalURL = gitlab.Ptr(d.Get("external_url").(string))
	}
	if d.HasChange("tier") {
		options.Tier = gitlab.Ptr(d.Get("tier").(string))
	}
	if v, ok := d.GetOk("cluster_agent_id"); d.HasChange("cluster_agent_id") && ok {
		options.ClusterAgentID = gitlab.Ptr(v.(int))
	}
	if d.HasChange("kubernetes_namespace") {
		options.KubernetesNamespace = gitlab.Ptr(d.Get("kubernetes_namespace").(string))
	}
	if d.HasChange("flux_resource_path") {
		options.FluxResourcePath = gitlab.Ptr(d.Get("flux_resource_path").(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Project %s update gitlab environment %d", project, environmentID))

	client := meta.(*gitlab.Client)

	if _, _, err := client.Environments.EditEnvironment(project, environmentID, options, gitlab.WithContext(ctx)); err != nil {
		return diag.Errorf("error editing gitlab project %s environment %d: %v", project, environmentID, err)
	}

	if v, ok := d.GetOk("cluster_agent_id"); d.HasChange("cluster_agent_id") && !ok || v == nil {
		err := updateNullableClusterAgentID(ctx, client, project, environmentID)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceGitlabProjectEnvironmentRead(ctx, d, meta)
}

func updateNullableClusterAgentID(ctx context.Context, client *gitlab.Client, pid any, environmentID int) error {
	options := &gitlab.EditEnvironmentOptions{}

	if _, _, err := client.Environments.EditEnvironment(pid, environmentID, options, gitlab.WithContext(ctx), func(request *retryablehttp.Request) error {
		optionsStruct := struct {
			ClusterAgentID *int `url:"cluster_agent_id" json:"cluster_agent_id"`
		}{
			ClusterAgentID: nil,
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

	}); err != nil {
		return fmt.Errorf("error editing gitlab project %s environment %d: %v", pid, environmentID, err)
	}

	return nil
}

func resourceGitlabProjectEnvironmentStop(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, environmentID, err := resourceGitlabProjectEnvironmentParseID(ctx, d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Stopping environment %d for Project %s", environmentID, project))
	if _, _, err = client.Environments.StopEnvironment(project, environmentID, nil, gitlab.WithContext(ctx)); err != nil {
		return diag.Errorf("error while stopping gitlab environment %q for project %s: %v", environmentID, project, err)
	}

	// Wait for the environment to be stopped, before we destroy it
	stateConf := &retry.StateChangeConf{
		Pending:    []string{"stopping", "unknown"}, // "unknown" happens when we fail to get the state
		Target:     []string{"stopped"},
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 3 * time.Second,
		Delay:      5 * time.Second,
		Refresh: func() (interface{}, string, error) {
			env, resp, err := client.Environments.StopEnvironment(project, environmentID, nil, gitlab.WithContext(ctx))
			// ignore the error here, as we'll be doing this until we succeed or timeout
			if err != nil {
				return resp, "unknown", err
			}

			return resp, env.State, nil
		},
	}
	if _, err = stateConf.WaitForStateContext(ctx); err != nil {
		return diag.Errorf("error waiting for gitlab project %s to stop in environment %q: %v", project, environmentID, err)
	}

	return nil
}

func resourceGitlabProjectEnvironmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, environmentID, err := resourceGitlabProjectEnvironmentParseID(ctx, d)
	if err != nil {
		return diag.FromErr(err)
	}

	stopBeforeDestroy := d.Get("stop_before_destroy").(bool)
	if stopBeforeDestroy {
		// resourceGitlabProjectEnvironmentStop waits for the environment to actually be stopped
		if err := resourceGitlabProjectEnvironmentStop(ctx, d, meta); err != nil {
			return err
		}
	}

	environment, _, err := client.Environments.GetEnvironment(project, environmentID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Project %s gitlab environment %d not found, removing from state", project, environmentID))
			d.SetId("")
			return nil
		}
		return diag.Errorf("error getting gitlab project %s environment %d: %v", project, environmentID, err)
	}

	if environment.State != "stopped" {
		return diag.Errorf("[ERROR] cannot destroy gitlab project %s environment %d: Environment must be in a stopped state before deletion. Set stop_before_destroy flag to attempt to auto stop the environment on destruction", project, environmentID)
	}

	if _, err = client.Environments.DeleteEnvironment(project, environmentID, gitlab.WithContext(ctx)); err != nil {
		return diag.Errorf("error deleting gitlab project %s environment %d: %v", project, environmentID, err)
	}

	return nil
}

func resourceGitlabProjectEnvironmentParseID(ctx context.Context, d *schema.ResourceData) (string, int, error) {
	project, rawEnvironmentID, err := utils.ParseTwoPartID(d.Id())

	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("[ERROR] cannot get project and environment ID from input: %v", d.Id()))
		return "", 0, err
	}

	environmentID, err := strconv.Atoi(rawEnvironmentID)

	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("[ERROR] cannot convert environment ID to int: %v", err))
		return "", 0, err
	}
	return project, environmentID, nil
}
