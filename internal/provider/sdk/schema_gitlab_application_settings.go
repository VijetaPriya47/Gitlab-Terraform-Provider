package sdk

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var allowedImportSources = []string{
	"github",
	"bitbucket",
	"bitbucket_server",
	"fogbugz",
	"git",
	"gitlab_project",
	"gitea",
	"manifest",
}

func gitlabApplicationSettingsSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"admin_mode": {
			Description: "Require administrators to enable Admin Mode by re-authenticating for administrative tasks.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"abuse_notification_email": {
			Description: "If set, abuse reports are sent to this address. Abuse reports are always available in the Admin Area.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"after_sign_out_path": {
			Description: "Where to redirect users after logout.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"after_sign_up_text": {
			Description: "Text shown to the user after signing up.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"akismet_api_key": {
			Description: "API key for Akismet spam protection.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"akismet_enabled": {
			Description: "(If enabled, requires: akismet_api_key) Enable or disable Akismet spam protection.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"allow_account_deletion": {
			Description: "Set to true to allow users to delete their accounts. Premium and Ultimate only.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"allow_group_owners_to_manage_ldap": {
			Description: "Set to true to allow group owners to manage LDAP.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"allow_local_requests_from_system_hooks": {
			Description: "Allow requests to the local network from system hooks.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"allow_local_requests_from_web_hooks_and_services": {
			Description: "Allow requests to the local network from web hooks and services.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"allow_project_creation_for_guest_and_below": {
			Description: "Indicates whether users assigned up to the Guest role can create groups and personal projects.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"allow_runner_registration_token": {
			Description: "Allow using a registration token to create a runner.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"archive_builds_in_human_readable": {
			Description: "Set the duration for which the jobs are considered as old and expired. After that time passes, the jobs are archived and no longer able to be retried. Make it empty to never expire jobs. It has to be no less than 1 day, for example: 15 days, 1 month, 2 years.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"asciidoc_max_includes": {
			Description: "Maximum limit of AsciiDoc include directives being processed in any one document. Maximum: 64.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"asset_proxy_enabled": {
			Description: "(If enabled, requires: asset_proxy_url) Enable proxying of assets. GitLab restart is required to apply changes.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"asset_proxy_secret_key": {
			Description: "Shared secret with the asset proxy server. GitLab restart is required to apply changes.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"asset_proxy_url": {
			Description: "URL of the asset proxy server. GitLab restart is required to apply changes.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"asset_proxy_allowlist": {
			Description: "Assets that match these domains are not proxied. Wildcards allowed. Your GitLab installation URL is automatically allowlisted. GitLab restart is required to apply changes.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Computed:    true,
		},

		"authorized_keys_enabled": {
			Description: "By default, we write to the authorized_keys file to support Git over SSH without additional configuration. GitLab can be optimized to authenticate SSH keys via the database file. Only disable this if you have configured your OpenSSH server to use the AuthorizedKeysCommand.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"auto_ban_user_on_excessive_projects_download": {
			Description: "When enabled, users will get automatically banned from the application when they download more than the maximum number of unique projects in the time period specified by max_number_of_repository_downloads and max_number_of_repository_downloads_within_time_period respectively. Self-managed, Ultimate only.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"auto_devops_domain": {
			Description: "Specify a domain to use by default for every project’s Auto Review Apps and Auto Deploy stages.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"auto_devops_enabled": {
			Description: "Enable Auto DevOps for projects by default. It automatically builds, tests, and deploys applications based on a predefined CI/CD configuration.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"automatic_purchased_storage_allocation": {
			Description: "Enabling this permits automatic allocation of purchased storage in a namespace.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"bulk_import_concurrent_pipeline_batch_limit": {
			Description: "Maximum simultaneous Direct Transfer batches to process.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"bulk_import_enabled": {
			Description: "Enable migrating GitLab groups by direct transfer.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"bulk_import_max_download_file_size": {
			Description: "Maximum download file size when importing from source GitLab instances by direct transfer.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"can_create_group": {
			Description: "Indicates whether users can create top-level groups.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"check_namespace_plan": {
			Description: "Enabling this makes only licensed EE features available to projects if the project namespace’s plan includes the feature or if the project is public.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"ci_max_includes": {
			Description: "The maximum number of includes per pipeline.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"ci_max_total_yaml_size_bytes": {
			Description: "The maximum amount of memory, in bytes, that can be allocated for the pipeline configuration, with all included YAML configuration files.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"commit_email_hostname": {
			Description: "Custom hostname (for private commit emails).",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"concurrent_bitbucket_import_jobs_limit": {
			Description: "Maximum number of simultaneous import jobs for the Bitbucket Cloud importer.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"concurrent_bitbucket_server_import_jobs_limit": {
			Description: "Maximum number of simultaneous import jobs for the Bitbucket Server importer.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"concurrent_github_import_jobs_limit": {
			Description: "Maximum number of simultaneous import jobs for the GitHub importer.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"container_expiration_policies_enable_historic_entries": {
			Description: "Enable cleanup policies for all projects.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"container_registry_cleanup_tags_service_max_list_size": {
			Description: "The maximum number of tags that can be deleted in a single execution of cleanup policies.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"container_registry_delete_tags_service_timeout": {
			Description: "The maximum time, in seconds, that the cleanup process can take to delete a batch of tags for cleanup policies.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"container_registry_expiration_policies_caching": {
			Description: "Caching during the execution of cleanup policies.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"container_registry_expiration_policies_worker_capacity": {
			Description: "Number of workers for cleanup policies.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"container_registry_token_expire_delay": {
			Description: "Container Registry token duration in minutes.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"package_registry_cleanup_policies_worker_capacity": {
			Description: "Number of workers assigned to the packages cleanup policies.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"deactivate_dormant_users": {
			Description: "Enable automatic deactivation of dormant users.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"deactivate_dormant_users_period": {
			Description: "Length of time (in days) after which a user is considered dormant.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"decompress_archive_file_timeout": {
			Description: "Default timeout for decompressing archived files, in seconds. Set to 0 to disable timeouts.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"default_artifacts_expire_in": {
			Description: "Set the default expiration time for each job’s artifacts.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"default_branch_name": {
			Description: "Instance-level custom initial branch name",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"default_branch_protection": {
			Description: "Determine if developers can push to the default branch. Can take: 0 (not protected, both users with the Developer role or Maintainer role can push new commits and force push), 1 (partially protected, users with the Developer role or Maintainer role can push new commits, but cannot force push) or 2 (fully protected, users with the Developer or Maintainer role cannot push new commits, but users with the Developer or Maintainer role can; no one can force push) as a parameter. Default is 2. Use `default_branch_protection_defaults` instead. To be removed in 19.0.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Deprecated:  "Use `default_branch_protection_defaults` instead. To be removed in 19.0.",
		},

		"default_branch_protection_defaults": {
			Description: "The default_branch_protection_defaults attribute describes the default branch protection defaults. All parameters are optional.",
			Type:        schema.TypeList,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"allow_force_push": {
						Description: "Allow force push for all users with push access.",
						Type:        schema.TypeBool,
						Optional:    true,
						Computed:    true,
					},
					"allowed_to_merge": {
						Description: "An array of access levels allowed to merge. Supports Developer (30) or Maintainer (40).",
						Type:        schema.TypeList,
						Elem:        &schema.Schema{Type: schema.TypeInt},
						Optional:    true,
						Computed:    true,
					},
					"allowed_to_push": {
						Description: "An array of access levels allowed to push. Supports Developer (30) or Maintainer (40).",
						Type:        schema.TypeList,
						Elem:        &schema.Schema{Type: schema.TypeInt},
						Optional:    true,
						Computed:    true,
					},
					"developer_can_initial_push": {
						Description: "Allow developers to initial push.",
						Type:        schema.TypeBool,
						Optional:    true,
						Computed:    true,
					},
				},
			},
			Optional: true,
			Computed: true,
		},

		"default_ci_config_path": {
			Description: "Default CI/CD configuration file and path for new projects (.gitlab-ci.yml if not set).",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"default_group_visibility": {
			Description: "What visibility level new groups receive. Can take private, internal and public as a parameter.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"default_preferred_language": {
			Description: "Default preferred language for users who are not logged in.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"default_project_creation": {
			Description: "Default project creation protection. Can take: 0 (No one), 1 (Maintainers) or 2 (Developers + Maintainers).",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"default_project_visibility": {
			Description: "What visibility level new projects receive. Can take private, internal and public as a parameter.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"default_projects_limit": {
			Description: "Project limit per user.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"default_snippet_visibility": {
			Description: "What visibility level new snippets receive. Can take private, internal and public as a parameter.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"delete_inactive_projects": {
			Description: "Enable inactive project deletion feature.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"default_syntax_highlighting_theme": {
			Description: "Default syntax highlighting theme for users who are new or not signed in. See IDs of available themes (https://gitlab.com/gitlab-org/gitlab/blob/master/lib/gitlab/themes.rb#L16)",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"delete_unconfirmed_users": {
			Description: "Specifies whether users who have not confirmed their email should be deleted. When set to true, unconfirmed users are deleted after unconfirmed_users_delete_after_days days. Self-managed, Premium and Ultimate only.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"deletion_adjourned_period": {
			Description: "The number of days to wait before deleting a project or group that is marked for deletion. Value must be between 1 and 90.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"diagramsnet_enabled": {
			Description:  "(If enabled, requires diagramsnet_url) Enable Diagrams.net integration.",
			Type:         schema.TypeBool,
			Optional:     true,
			Computed:     true,
			RequiredWith: []string{"diagramsnet_url"},
		},

		"diagramsnet_url": {
			Description:  "The Diagrams.net instance URL for integration.",
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			RequiredWith: []string{"diagramsnet_enabled"},
		},

		"diff_max_patch_bytes": {
			Description: "Maximum diff patch size, in bytes.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"diff_max_files": {
			Description: "Maximum files in a diff.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"diff_max_lines": {
			Description: "Maximum lines in a diff.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"disable_admin_oauth_scopes": {
			Description: "Stops administrators from connecting their GitLab accounts to non-trusted OAuth 2.0 applications that have the api, read_api, read_repository, write_repository, read_registry, write_registry, or sudo scopes.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"disable_feed_token": {
			Description: "Disable display of RSS/Atom and calendar feed tokens.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"disable_personal_access_tokens": {
			Description: "Disable personal access tokens. Self-managed, Premium and Ultimate only. There is no method available to enable a personal access token that’s been disabled through the API. This is a known issue.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"disabled_oauth_sign_in_sources": {
			Description: "Disabled OAuth sign-in sources.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Computed:    true,
		},

		"dns_rebinding_protection_enabled": {
			Description: "Enforce DNS rebinding attack protection.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"domain_denylist_enabled": {
			Description: "(If enabled, requires: domain_denylist) Allows blocking sign-ups from emails from specific domains.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"domain_denylist": {
			Description: "Users with email addresses that match these domains cannot sign up. Wildcards allowed. Use separate lines for multiple entries. Ex: domain.com, *.domain.com.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Computed:    true,
		},

		"domain_allowlist": {
			Description: "Force people to use only corporate emails for sign-up. Null means there is no restriction.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Computed:    true,
		},

		"downstream_pipeline_trigger_limit_per_project_user_sha": {
			Description: "Maximum downstream pipeline trigger rate.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"dsa_key_restriction": {
			Description: "The minimum allowed bit length of an uploaded DSA key. 0 means no restriction. -1 disables DSA keys.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"duo_features_enabled": {
			Description: "Indicates whether GitLab Duo features are enabled for this instance. Self-managed, Premium and Ultimate only.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"ecdsa_key_restriction": {
			Description: "The minimum allowed curve size (in bits) of an uploaded ECDSA key. 0 means no restriction. -1 disables ECDSA keys.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"ecdsa_sk_key_restriction": {
			Description: "The minimum allowed curve size (in bits) of an uploaded ECDSA_SK key. 0 means no restriction. -1 disables ECDSA_SK keys.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"ed25519_key_restriction": {
			Description: "The minimum allowed curve size (in bits) of an uploaded ED25519 key. 0 means no restriction. -1 disables ED25519 keys.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"ed25519_sk_key_restriction": {
			Description: "The minimum allowed curve size (in bits) of an uploaded ED25519_SK key. 0 means no restriction. -1 disables ED25519_SK keys.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"eks_access_key_id": {
			Description: "AWS IAM access key ID.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"eks_account_id": {
			Description: "Amazon account ID.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"eks_integration_enabled": {
			Description: "Enable integration with Amazon EKS.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"eks_secret_access_key": {
			Description: "AWS IAM secret access key.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_aws_access_key": {
			Description: "AWS IAM access key.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_aws_region": {
			Description: "The AWS region the Elasticsearch domain is configured.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_aws_secret_access_key": {
			Description: "AWS IAM secret access key.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_aws": {
			Description: "Enable the use of AWS hosted Elasticsearch.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_indexed_field_length_limit": {
			Description: "Maximum size of text fields to index by Elasticsearch. 0 value means no limit. This does not apply to repository and wiki indexing.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_indexed_file_size_limit_kb": {
			Description: "Maximum size of repository and wiki files that are indexed by Elasticsearch.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_indexing": {
			Description: "Enable Elasticsearch indexing.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_limit_indexing": {
			Description: "Limit Elasticsearch to index certain namespaces and projects.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_max_bulk_concurrency": {
			Description: "Maximum concurrency of Elasticsearch bulk requests per indexing operation. This only applies to repository indexing operations.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_max_bulk_size_mb": {
			Description: "Maximum size of Elasticsearch bulk indexing requests in MB. This only applies to repository indexing operations.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_max_code_indexing_concurrency": {
			Description: "Maximum concurrency of Elasticsearch code indexing background jobs. This only applies to repository indexing operations. Premium and Ultimate only.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_namespace_ids": {
			Description: "The namespaces to index via Elasticsearch if elasticsearch_limit_indexing is enabled.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeInt},
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_requeue_workers": {
			Description: "Enable automatic requeuing of indexing workers. This improves non-code indexing throughput by enqueuing Sidekiq jobs until all documents are processed. Premium and Ultimate only.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_project_ids": {
			Description: "The projects to index via Elasticsearch if elasticsearch_limit_indexing is enabled.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeInt},
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_search": {
			Description: "Enable Elasticsearch search.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_worker_number_of_shards": {
			Description: "Number of indexing worker shards. This improves non-code indexing throughput by enqueuing more parallel Sidekiq jobs. Premium and Ultimate only.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_url": {
			Description: "The URL to use for connecting to Elasticsearch. Use a comma-separated list to support cluster (for example, http://localhost:9200, http://localhost:9201).",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_username": {
			Description: "The username of your Elasticsearch instance.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"elasticsearch_password": {
			Description: "The password of your Elasticsearch instance.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"email_additional_text": {
			Description: "Additional text added to the bottom of every email for legal/auditing/compliance reasons.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"email_author_in_body": {
			Description: "Some email servers do not support overriding the email sender name. Enable this option to include the name of the author of the issue, merge request or comment in the email body instead.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"email_confirmation_setting": {
			Description: "Specifies whether users must confirm their email before sign in. Possible values are off, soft, and hard.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"enable_artifact_external_redirect_warning_page": {
			Description: "Show the external redirect page that warns you about user-generated content in GitLab Pages.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"enabled_git_access_protocol": {
			Description: "Enabled protocols for Git access. Allowed values are: ssh, http, and nil to allow both protocols.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				// if "nil" is passed in, it will set the value to "" in the backend.
				if strings.ToLower(new) == "nil" {
					new = ""
				}
				return old == new
			},
		},

		"enforce_namespace_storage_limit": {
			Description: "Enabling this permits enforcement of namespace storage limits.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"enforce_terms": {
			Description: "(If enabled, requires: terms) Enforce application ToS to all users.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"external_auth_client_cert": {
			Description: "(If enabled, requires: external_auth_client_key) The certificate to use to authenticate with the external authorization service.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"external_auth_client_key_pass": {
			Description: "Passphrase to use for the private key when authenticating with the external service this is encrypted when stored.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"external_auth_client_key": {
			Description: "Private key for the certificate when authentication is required for the external authorization service, this is encrypted when stored.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"external_authorization_service_default_label": {
			Description: "The default classification label to use when requesting authorization and no classification label has been specified on the project.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"external_authorization_service_enabled": {
			Description: "(If enabled, requires: external_authorization_service_default_label, external_authorization_service_timeout and external_authorization_service_url) Enable using an external authorization service for accessing projects.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"external_authorization_service_timeout": {
			Description: "The timeout after which an authorization request is aborted, in seconds. When a request times out, access is denied to the user. (min: 0.001, max: 10, step: 0.001).",
			Type:        schema.TypeFloat,
			Optional:    true,
			Computed:    true,
		},

		"external_authorization_service_url": {
			Description: "URL to which authorization requests are directed.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"external_pipeline_validation_service_url": {
			Description: "URL to use for pipeline validation requests.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"external_pipeline_validation_service_token": {
			Description: "Optional. Token to include as the X-Gitlab-Token header in requests to the URL in external_pipeline_validation_service_url.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"external_pipeline_validation_service_timeout": {
			Description: "How long to wait for a response from the pipeline validation service. Assumes OK if it times out.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"failed_login_attempts_unlock_period_in_minutes": {
			Description: "Time period in minutes after which the user is unlocked when maximum number of failed sign-in attempts reached.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"file_template_project_id": {
			Description: "The ID of a project to load custom file templates from.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"first_day_of_week": {
			Description: "Start day of the week for calendar views and date pickers. Valid values are 0 for Sunday, 1 for Monday, and 6 for Saturday.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"geo_node_allowed_ips": {
			Description: "Comma-separated list of IPs and CIDRs of allowed secondary nodes. For example, 1.1.1.1, 2.2.2.0/24.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"geo_status_timeout": {
			Description: "The amount of seconds after which a request to get a secondary node status times out.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"git_rate_limit_users_alertlist": {
			Description: "List of user IDs that are emailed when the Git abuse rate limit is exceeded. Maximum: 100 user IDs. Self-managed, Ultimate only.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeInt},
			Optional:    true,
			Computed:    true,
		},

		"git_rate_limit_users_allowlist": {
			Description: "List of usernames excluded from Git anti-abuse rate limits. Maximum: 100 usernames. Self-managed, Ultimate only.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Computed:    true,
		},

		"git_two_factor_session_expiry": {
			Description: "Maximum duration (in minutes) of a session for Git operations when 2FA is enabled.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"gitaly_timeout_default": {
			Description: "Default Gitaly timeout, in seconds. This timeout is not enforced for Git fetch/push operations or Sidekiq jobs. Set to 0 to disable timeouts.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"gitaly_timeout_fast": {
			Description: "Gitaly fast operation timeout, in seconds. Some Gitaly operations are expected to be fast. If they exceed this threshold, there may be a problem with a storage shard and ‘failing fast’ can help maintain the stability of the GitLab instance. Set to 0 to disable timeouts.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"gitaly_timeout_medium": {
			Description: "Medium Gitaly timeout, in seconds. This should be a value between the Fast and the Default timeout. Set to 0 to disable timeouts.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"gitlab_dedicated_instance": {
			Description: "Indicates whether the instance was provisioned for GitLab Dedicated.",
			Type:        schema.TypeBool,
			Computed:    true,
		},

		"gitlab_environment_toolkit_instance": {
			Description: "Indicates whether the instance was provisioned with the GitLab Environment Toolkit for Service Ping reporting.",
			Type:        schema.TypeBool,
			Computed:    true,
		},

		"gitlab_shell_operation_limit": {
			Description: "Maximum number of Git operations per minute a user can perform.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"gitpod_enabled": {
			Description: "Enable Gitpod integration.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"gitpod_url": {
			Description: "The Gitpod instance URL for integration.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"globally_allowed_ips": {
			Description: "Comma-separated list of IP addresses and CIDRs always allowed for inbound traffic. For example, 1.1.1.1, 2.2.2.0/24.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"grafana_enabled": {
			Description: "Enable Grafana.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"grafana_url": {
			Description: "Grafana URL.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"gravatar_enabled": {
			Description: "Enable Gravatar.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"group_owners_can_manage_default_branch_protection": {
			Description: "Prevent overrides of default branch protection.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"help_page_hide_commercial_content": {
			Description: "Hide marketing-related entries from help.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"help_page_support_url": {
			Description: "Alternate support URL for help page and help dropdown.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"help_page_text": {
			Description: "Custom text displayed on the help page.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"help_text": {
			Description: "GitLab server administrator information.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"hide_third_party_offers": {
			Description: "Do not display offers from third parties in GitLab.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"home_page_url": {
			Description: "Redirect to this URL when not logged in.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"housekeeping_enabled": {
			Description: "Enable or disable Git housekeeping. If enabled, requires housekeeping_optimize_repository_period.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"housekeeping_optimize_repository_period": {
			Description: "Number of Git pushes after which an incremental git-repack is run.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"html_emails_enabled": {
			Description: "Enable HTML emails.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"import_sources": {
			Description: fmt.Sprintf("Sources to allow project import from. Valid values are: %s", utils.RenderValueListForDocs(allowedImportSources)),
			Type:        schema.TypeList,
			Elem: &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringInSlice(allowedImportSources, false),
			},
			Optional: true,
			Computed: true,
		},

		"in_product_marketing_emails_enabled": {
			Description: "Enable in-product marketing emails.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"inactive_projects_delete_after_months": {
			Description: "If delete_inactive_projects is true, the time (in months) to wait before deleting inactive projects.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"inactive_projects_min_size_mb": {
			Description: "If delete_inactive_projects is true, the minimum repository size for projects to be checked for inactivity.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"inactive_projects_send_warning_email_after_months": {
			Description: "If delete_inactive_projects is true, sets the time (in months) to wait before emailing maintainers that the project is scheduled be deleted because it is inactive.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"include_optional_metrics_in_service_ping": {
			Description: "Whether or not optional metrics are enabled in Service Ping.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"invisible_captcha_enabled": {
			Description: "Enable Invisible CAPTCHA spam detection during sign-up.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"issues_create_limit": {
			Description: "Max number of issue creation requests per minute per user.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"jira_connect_application_key": {
			Description: "ID of the OAuth application used to authenticate with the GitLab for Jira Cloud app.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"jira_connect_proxy_url": {
			Description: "URL of the GitLab instance used as a proxy for the GitLab for Jira Cloud app.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"jira_connect_public_key_storage_enabled": {
			Description: "Enable public key storage for the GitLab for Jira Cloud app.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"keep_latest_artifact": {
			Description: "Prevent the deletion of the artifacts from the most recent successful jobs, regardless of the expiry time.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"local_markdown_version": {
			Description: "Increase this value when any cached Markdown should be invalidated.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"lock_memberships_to_ldap": {
			Description: "Set to true to lock all memberships to LDAP. Premium and Ultimate only.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"lock_duo_features_enabled": {
			Description: "Indicates whether the GitLab Duo features enabled setting is enforced for all subgroups. Self-managed, Premium and Ultimate only.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"mailgun_signing_key": {
			Description: "The Mailgun HTTP webhook signing key for receiving events from webhook.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"mailgun_events_enabled": {
			Description: "Enable Mailgun event receiver.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"maintenance_mode_message": {
			Description: "Message displayed when instance is in maintenance mode.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"maintenance_mode": {
			Description: "When instance is in maintenance mode, non-administrative users can sign in with read-only access and make read-only API requests.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"maven_package_requests_forwarding": {
			Description: "Use repo.maven.apache.org as a default remote repository when the package is not found in the GitLab Package Registry for Maven. Premium and Ultimate only.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"max_artifacts_size": {
			Description: "Maximum artifacts size in MB.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_attachment_size": {
			Description: "Limit attachment size in MB.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_decompressed_archive_size": {
			Description: "Maximum decompressed archive size in bytes.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_export_size": {
			Description: "Maximum export size in MB. 0 for unlimited.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_import_remote_file_size": {
			Description: "Maximum remote file size for imports from external object storages.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_import_size": {
			Description: "Maximum import size in MB. 0 for unlimited.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_login_attempts": {
			Description: "Maximum number of sign-in attempts before locking out the user.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_pages_size": {
			Description: "Maximum size of pages repositories in MB.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_personal_access_token_lifetime": {
			Description: "Maximum allowable lifetime for access tokens in days.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_ssh_key_lifetime": {
			Description: "Maximum allowable lifetime for SSH keys in days.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_terraform_state_size_bytes": {
			Description: "Maximum size in bytes of the Terraform state files. Set this to 0 for unlimited file size.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"metrics_method_call_threshold": {
			Description: "A method call is only tracked when it takes longer than the given amount of milliseconds.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_number_of_repository_downloads": {
			Description: "Maximum number of unique repositories a user can download in the specified time period before they are banned. Maximum: 10,000 repositories.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"max_number_of_repository_downloads_within_time_period": {
			Description: "Reporting time period (in seconds). Maximum: 864000 seconds (10 days).",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"mirror_available": {
			Description: "Allow repository mirroring to configured by project Maintainers. If disabled, only Administrators can configure repository mirroring.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"mirror_capacity_threshold": {
			Description: "Minimum capacity to be available before scheduling more mirrors preemptively.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"mirror_max_capacity": {
			Description: "Maximum number of mirrors that can be synchronizing at the same time.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"mirror_max_delay": {
			Description: "Maximum time (in minutes) between updates that a mirror can have when scheduled to synchronize.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"notify_on_unknown_sign_in": {
			Description: "Enable sending notification if sign in from unknown IP address happens",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},
		
		"npm_package_requests_forwarding": {
			Description: "Use npmjs.org as a default remote repository when the package is not found in the GitLab Package Registry for npm.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"nuget_skip_metadata_url_validation": {
			Description: "Indicates whether to skip metadata URL validation for the NuGet package.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"outbound_local_requests_whitelist": {
			Description: "Define a list of trusted domains or IP addresses to which local requests are allowed when local requests for hooks and services are disabled.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Computed:    true,
		},

		"package_metadata_purl_types": {
			Description: "List of package registry metadata to sync. See the list of the available values (https://gitlab.com/gitlab-org/gitlab/-/blob/ace16c20d5da7c4928dd03fb139692638b557fe3/app/models/concerns/enums/package_metadata.rb#L5). Self-managed, Ultimate only.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeInt},
			Optional:    true,
			Computed:    true,
		},

		"package_registry_allow_anyone_to_pull_option": {
			Description: "Enable to allow anyone to pull from Package Registry visible and changeable.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"pages_domain_verification_enabled": {
			Description: "Require users to prove ownership of custom domains. Domain verification is an essential security measure for public GitLab sites. Users are required to demonstrate they control a domain before it is enabled.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"password_authentication_enabled_for_git": {
			Description: "Enable authentication for Git over HTTP(S) via a GitLab account password.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"password_authentication_enabled_for_web": {
			Description: "Enable authentication for the web interface via a GitLab account password.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"minimum_password_length": {
			Description: "Indicates whether passwords require a minimum length. Premium and Ultimate only.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"password_number_required": {
			Description: "Indicates whether passwords require at least one number.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"password_symbol_required": {
			Description: "Indicates whether passwords require at least one symbol character.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"password_uppercase_required": {
			Description: "Indicates whether passwords require at least one uppercase letter.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"password_lowercase_required": {
			Description: "Indicates whether passwords require at least one lowercase letter.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"performance_bar_allowed_group_path": {
			Description: "Path of the group that is allowed to toggle the performance bar.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"personal_access_token_prefix": {
			Description: "Prefix for all generated personal access tokens.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"pipeline_limit_per_project_user_sha": {
			Description: "Maximum number of pipeline creation requests per minute per user and commit.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"plantuml_enabled": {
			Description: "(If enabled, requires: plantuml_url) Enable PlantUML integration.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"plantuml_url": {
			Description: "The PlantUML instance URL for integration.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"polling_interval_multiplier": {
			Description: "Interval multiplier used by endpoints that perform polling. Set to 0 to disable polling.",
			Type:        schema.TypeFloat,
			Optional:    true,
			Computed:    true,
		},

		"project_export_enabled": {
			Description: "Enable project export.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"project_jobs_api_rate_limit": {
			Description: "Maximum authenticated requests to /project/:id/jobs per minute.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"projects_api_rate_limit_unauthenticated": {
			Description: "Max number of requests per 10 minutes per IP address for unauthenticated requests to the list all projects API. To disable throttling set to 0.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"prometheus_metrics_enabled": {
			Description: "Enable Prometheus metrics.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"protected_ci_variables": {
			Description: "CI/CD variables are protected by default.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"push_event_activities_limit": {
			Description: "Number of changes (branches or tags) in a single push to determine whether individual push events or bulk push events are created. Bulk push events are created if it surpasses that value.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"push_event_hooks_limit": {
			Description: "Number of changes (branches or tags) in a single push to determine whether webhooks and services fire or not. Webhooks and services aren’t submitted if it surpasses that value.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"pypi_package_requests_forwarding": {
			Description: "Use pypi.org as a default remote repository when the package is not found in the GitLab Package Registry for PyPI.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"rate_limiting_response_text": {
			Description: "When rate limiting is enabled via the throttle_* settings, send this plain text response when a rate limit is exceeded. ‘Retry later’ is sent if this is blank.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"raw_blob_request_limit": {
			Description: "Max number of requests per minute for each raw path. To disable throttling set to 0.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"receptive_cluster_agents_enabled": {
			Description: "Enable receptive mode for GitLab Agents for Kubernetes.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"remember_me_enabled": {
			Description: "Enable Remember me setting.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"require_admin_two_factor_authentication": {
			Description: "Allow administrators to require 2FA for all administrators on the instance.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"require_personal_access_token_expiry": {
			Description: "When enabled, users must set an expiration date when creating a group or project access token, or a personal access token owned by a non-service account.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"search_rate_limit": {
			Description: "Max number of requests per minute for performing a search while authenticated. To disable throttling set to 0.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"search_rate_limit_unauthenticated": {
			Description: "Max number of requests per minute for performing a search while unauthenticated. To disable throttling set to 0.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"recaptcha_enabled": {
			Description: "(If enabled, requires: recaptcha_private_key and recaptcha_site_key) Enable reCAPTCHA.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"recaptcha_private_key": {
			Description: "Private key for reCAPTCHA.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"recaptcha_site_key": {
			Description: "Site key for reCAPTCHA.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"receive_max_input_size": {
			Description: "Maximum push size (MB).",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"repository_checks_enabled": {
			Description: "GitLab periodically runs git fsck in all project and wiki repositories to look for silent disk corruption issues.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"repository_size_limit": {
			Description: "Size limit per repository (MB).",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"repository_storages_weighted": {
			Description: "Hash of names taken from gitlab.yml to weights. New projects are created in one of these stores, chosen by a weighted random selection.",
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeInt},
			Optional:    true,
			Computed:    true,
		},

		"require_admin_approval_after_user_signup": {
			Description: "When enabled, any user that signs up for an account using the registration form is placed under a Pending approval state and has to be explicitly approved by an administrator.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"require_two_factor_authentication": {
			Description: "(If enabled, requires: two_factor_grace_period) Require all users to set up Two-factor authentication.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"restricted_visibility_levels": {
			Description: "Selected levels cannot be used by non-Administrator users for groups, projects or snippets. Can take private, internal and public as a parameter. Null means there is no restriction.",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Computed:    true,
		},

		"rsa_key_restriction": {
			Description: "The minimum allowed bit length of an uploaded RSA key. 0 means no restriction. -1 disables RSA keys.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"security_approval_policies_limit": {
			Description: "Maximum number of active merge request approval policies per security policy project. Maximum: 20",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"security_policy_global_group_approvers_enabled": {
			Description: "Whether to look up merge request approval policy approval groups globally or within project hierarchies.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"security_txt_content": {
			Description: "Public security contact information.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"send_user_confirmation_email": {
			Description: "Send confirmation email on sign-up.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"service_access_tokens_expiration_enforced": {
			Description: "Flag to indicate if token expiry date can be optional for service account users",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"session_expire_delay": {
			Description: "Session duration in minutes. GitLab restart is required to apply changes.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"shared_runners_enabled": {
			Description: "(If enabled, requires: shared_runners_text and shared_runners_minutes) Enable shared runners for new projects.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"shared_runners_minutes": {
			Description: "Set the maximum number of CI/CD minutes that a group can use on shared runners per month.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"shared_runners_text": {
			Description: "Shared runners text.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"sidekiq_job_limiter_mode": {
			Description: "track or compress. Sets the behavior for Sidekiq job size limits.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"sidekiq_job_limiter_compression_threshold_bytes": {
			Description: "The threshold in bytes at which Sidekiq jobs are compressed before being stored in Redis.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"sidekiq_job_limiter_limit_bytes": {
			Description: "The threshold in bytes at which Sidekiq jobs are rejected. 0 means do not reject any job.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"sign_in_text": {
			Description: "Text on the login page.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"signup_enabled": {
			Description: "Enable registration.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"silent_admin_exports_enabled": {
			Description: "Enable Silent admin exports.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"silent_mode_enabled": {
			Description: "Enable Silent mode.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"slack_app_enabled": {
			Description: "(If enabled, requires: slack_app_id, slack_app_secret and slack_app_secret) Enable Slack app.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"slack_app_id": {
			Description: "The app ID of the Slack-app.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"slack_app_secret": {
			Description: "The app secret of the Slack-app.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"slack_app_signing_secret": {
			Description: "The signing secret of the Slack-app.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"slack_app_verification_token": {
			Description: "The verification token of the Slack-app.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"snippet_size_limit": {
			Description: "Max snippet content size in bytes.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"snowplow_app_id": {
			Description: "The Snowplow site name / application ID. (for example, gitlab)",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"snowplow_collector_hostname": {
			Description: "The Snowplow collector hostname. (for example, snowplow.trx.gitlab.net)",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"snowplow_cookie_domain": {
			Description: "The Snowplow cookie domain. (for example, .gitlab.com)",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"snowplow_database_collector_hostname": {
			Description: "The Snowplow collector for database events hostname. (for example, db-snowplow.trx.gitlab.net)",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"snowplow_enabled": {
			Description: "Enable snowplow tracking.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"sourcegraph_enabled": {
			Description: "Enables Sourcegraph integration. If enabled, requires sourcegraph_url.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"sourcegraph_public_only": {
			Description: "Blocks Sourcegraph from being loaded on private and internal projects.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"sourcegraph_url": {
			Description: "The Sourcegraph instance URL for integration.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"spam_check_endpoint_enabled": {
			Description: "Enables spam checking using external Spam Check API endpoint.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"spam_check_endpoint_url": {
			Description: "URL of the external Spamcheck service endpoint. Valid URI schemes are grpc or tls. Specifying tls forces communication to be encrypted.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"spam_check_api_key": {
			Description: "API key used by GitLab for accessing the Spam Check service endpoint.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Computed:    true,
		},

		"static_objects_external_storage_auth_token": {
			Description: "Authentication token for the external storage linked in static_objects_external_storage_url.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
		},

		"static_objects_external_storage_url": {
			Description: "URL to an external storage for repository static objects.",
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
		},

		"suggest_pipeline_enabled": {
			Description: "Enable pipeline suggestion banner.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"terminal_max_session_time": {
			Description: "Maximum time for web terminal websocket connection (in seconds). Set to 0 for unlimited time.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"terms": {
			Description: "(Required by: enforce_terms) Markdown content for the ToS.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"throttle_authenticated_api_enabled": {
			Description: "(If enabled, requires: throttle_authenticated_api_period_in_seconds and throttle_authenticated_api_requests_per_period) Enable authenticated API request rate limit. Helps reduce request volume (for example, from crawlers or abusive bots).",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"throttle_authenticated_api_period_in_seconds": {
			Description: "Rate limit period (in seconds).",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_authenticated_api_requests_per_period": {
			Description: "Maximum requests per period per user.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_authenticated_packages_api_enabled": {
			Description: "(If enabled, requires: throttle_authenticated_packages_api_period_in_seconds and throttle_authenticated_packages_api_requests_per_period) Enable authenticated API request rate limit. Helps reduce request volume (for example, from crawlers or abusive bots). View Package Registry rate limits for more details.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"throttle_authenticated_packages_api_period_in_seconds": {
			Description: "Rate limit period (in seconds). View Package Registry rate limits for more details.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_authenticated_packages_api_requests_per_period": {
			Description: "Maximum requests per period per user. View Package Registry rate limits for more details.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_authenticated_web_enabled": {
			Description: "(If enabled, requires: throttle_authenticated_web_period_in_seconds and throttle_authenticated_web_requests_per_period) Enable authenticated web request rate limit. Helps reduce request volume (for example, from crawlers or abusive bots).",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"throttle_authenticated_web_period_in_seconds": {
			Description: "Rate limit period (in seconds).",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_authenticated_web_requests_per_period": {
			Description: "Maximum requests per period per user.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_unauthenticated_api_enabled": {
			Description: "(If enabled, requires: throttle_unauthenticated_api_period_in_seconds and throttle_unauthenticated_api_requests_per_period) Enable unauthenticated API request rate limit. Helps reduce request volume (for example, from crawlers or abusive bots).",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"throttle_unauthenticated_api_period_in_seconds": {
			Description: "Rate limit period in seconds.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_unauthenticated_api_requests_per_period": {
			Description: "Max requests per period per IP.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_unauthenticated_packages_api_enabled": {
			Description: "(If enabled, requires: throttle_unauthenticated_packages_api_period_in_seconds and throttle_unauthenticated_packages_api_requests_per_period) Enable authenticated API request rate limit. Helps reduce request volume (for example, from crawlers or abusive bots). View Package Registry rate limits for more details.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"throttle_unauthenticated_packages_api_period_in_seconds": {
			Description: "Rate limit period (in seconds). View Package Registry rate limits for more details.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_unauthenticated_packages_api_requests_per_period": {
			Description: "Maximum requests per period per user. View Package Registry rate limits for more details.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_unauthenticated_web_enabled": {
			Description: "(If enabled, requires: throttle_unauthenticated_web_period_in_seconds and throttle_unauthenticated_web_requests_per_period) Enable unauthenticated web request rate limit. Helps reduce request volume (for example, from crawlers or abusive bots).",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"throttle_unauthenticated_web_period_in_seconds": {
			Description: "Rate limit period in seconds.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"throttle_unauthenticated_web_requests_per_period": {
			Description: "Max requests per period per IP.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"time_tracking_limit_to_hours": {
			Description: "Limit display of time tracking units to hours.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"two_factor_grace_period": {
			Description: "Amount of time (in hours) that users are allowed to skip forced configuration of two-factor authentication.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"unconfirmed_users_delete_after_days": {
			Description: "Specifies how many days after sign-up to delete users who have not confirmed their email. Only applicable if delete_unconfirmed_users is set to true. Must be 1 or greater. Self-managed, Premium and Ultimate only.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"unique_ips_limit_enabled": {
			Description: "(If enabled, requires: unique_ips_limit_per_user and unique_ips_limit_time_window) Limit sign in from multiple IPs.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"unique_ips_limit_per_user": {
			Description: "Maximum number of IPs per user.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"unique_ips_limit_time_window": {
			Description: "How many seconds an IP is counted towards the limit.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},

		"update_runner_versions_enabled": {
			Description: "Fetch GitLab Runner release version data from GitLab.com.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"usage_ping_enabled": {
			Description: "Every week GitLab reports license usage back to GitLab, Inc.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"use_clickhouse_for_analytics": {
			Description: "Enables ClickHouse as a data source for analytics reports. ClickHouse must be configured for this setting to take effect. Available on Premium and Ultimate only.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"user_deactivation_emails_enabled": {
			Description: "Send an email to users upon account deactivation.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"user_default_external": {
			Description: "Newly registered users are external by default.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"user_default_internal_regex": {
			Description: "Specify an email address regex pattern to identify default internal users.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"user_defaults_to_private_profile": {
			Description: "Newly created users have private profile by default.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"user_oauth_applications": {
			Description: "Allow users to register any application to use GitLab as an OAuth provider.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"user_show_add_ssh_key_message": {
			Description: "When set to false disable the You won't be able to pull or push project code via SSH warning shown to users with no uploaded SSH key.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"valid_runner_registrars": {
			Description: "List of types which are allowed to register a GitLab Runner. Can be [], ['group'], ['project'] or ['group', 'project'].",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Computed:    true,
		},

		"version_check_enabled": {
			Description: "Let GitLab inform you when an update is available.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"whats_new_variant": {
			Description: "What's new variant, possible values: all_tiers, current_tier, and disabled.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},

		"web_ide_clientside_preview_enabled": {
			Description: "Live Preview (allow live previews of JavaScript projects in the Web IDE using CodeSandbox Live Preview).",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},

		"wiki_page_max_content_bytes": {
			Description: "Maximum wiki page content size in bytes. The minimum value is 1024 bytes.",
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
		},
	}
}

func gitlabApplicationSettingsToStateMap(settings *gitlab.Settings) map[string]any {
	stateMap := make(map[string]any)
	stateMap["admin_mode"] = settings.AdminMode
	stateMap["abuse_notification_email"] = settings.AbuseNotificationEmail
	stateMap["after_sign_out_path"] = settings.AfterSignOutPath
	stateMap["after_sign_up_text"] = settings.AfterSignUpText
	stateMap["akismet_api_key"] = settings.AkismetAPIKey
	stateMap["akismet_enabled"] = settings.AkismetEnabled
	stateMap["allow_account_deletion"] = settings.AllowAccountDeletion
	stateMap["allow_group_owners_to_manage_ldap"] = settings.AllowGroupOwnersToManageLDAP
	stateMap["allow_local_requests_from_system_hooks"] = settings.AllowLocalRequestsFromSystemHooks
	stateMap["allow_local_requests_from_web_hooks_and_services"] = settings.AllowLocalRequestsFromWebHooksAndServices
	stateMap["allow_project_creation_for_guest_and_below"] = settings.AllowProjectCreationForGuestAndBelow
	stateMap["allow_runner_registration_token"] = settings.AllowRunnerRegistrationToken
	stateMap["archive_builds_in_human_readable"] = settings.ArchiveBuildsInHumanReadable
	stateMap["asciidoc_max_includes"] = settings.ASCIIDocMaxIncludes
	stateMap["asset_proxy_enabled"] = settings.AssetProxyEnabled
	stateMap["asset_proxy_secret_key"] = settings.AssetProxySecretKey
	stateMap["asset_proxy_url"] = settings.AssetProxyURL
	stateMap["asset_proxy_allowlist"] = settings.AssetProxyAllowlist
	stateMap["authorized_keys_enabled"] = settings.AuthorizedKeysEnabled
	stateMap["auto_ban_user_on_excessive_projects_download"] = settings.AutoBanUserOnExcessiveProjectsDownload
	stateMap["auto_devops_domain"] = settings.AutoDevOpsDomain
	stateMap["auto_devops_enabled"] = settings.AutoDevOpsEnabled
	stateMap["automatic_purchased_storage_allocation"] = settings.AutomaticPurchasedStorageAllocation
	stateMap["bulk_import_concurrent_pipeline_batch_limit"] = settings.BulkImportConcurrentPipelineBatchLimit
	stateMap["bulk_import_enabled"] = settings.BulkImportEnabled
	stateMap["can_create_group"] = settings.CanCreateGroup
	stateMap["check_namespace_plan"] = settings.CheckNamespacePlan
	stateMap["ci_max_includes"] = settings.CIMaxIncludes
	stateMap["ci_max_total_yaml_size_bytes"] = settings.CIMaxTotalYAMLSizeBytes
	stateMap["commit_email_hostname"] = settings.CommitEmailHostname
	stateMap["concurrent_bitbucket_import_jobs_limit"] = settings.ConcurrentBitbucketImportJobsLimit
	stateMap["concurrent_bitbucket_server_import_jobs_limit"] = settings.ConcurrentBitbucketServerImportJobsLimit
	stateMap["concurrent_github_import_jobs_limit"] = settings.ConcurrentGitHubImportJobsLimit
	stateMap["container_expiration_policies_enable_historic_entries"] = settings.ContainerExpirationPoliciesEnableHistoricEntries
	stateMap["container_registry_cleanup_tags_service_max_list_size"] = settings.ContainerRegistryCleanupTagsServiceMaxListSize
	stateMap["container_registry_delete_tags_service_timeout"] = settings.ContainerRegistryDeleteTagsServiceTimeout
	stateMap["container_registry_expiration_policies_caching"] = settings.ContainerRegistryExpirationPoliciesCaching
	stateMap["container_registry_expiration_policies_worker_capacity"] = settings.ContainerRegistryExpirationPoliciesWorkerCapacity
	stateMap["container_registry_token_expire_delay"] = settings.ContainerRegistryTokenExpireDelay
	stateMap["package_registry_cleanup_policies_worker_capacity"] = settings.PackageRegistryCleanupPoliciesWorkerCapacity
	stateMap["deactivate_dormant_users"] = settings.DeactivateDormantUsers
	stateMap["deactivate_dormant_users_period"] = settings.DeactivateDormantUsersPeriod
	stateMap["decompress_archive_file_timeout"] = settings.DecompressArchiveFileTimeout
	stateMap["default_artifacts_expire_in"] = settings.DefaultArtifactsExpireIn
	stateMap["default_branch_name"] = settings.DefaultBranchName
	// nolint:staticcheck // SA1019 ignore deprecated DefaultBranchProtection
	stateMap["default_branch_protection"] = settings.DefaultBranchProtection
	stateMap["default_ci_config_path"] = settings.DefaultCiConfigPath
	stateMap["default_group_visibility"] = settings.DefaultGroupVisibility
	stateMap["default_preferred_language"] = settings.DefaultPreferredLanguage
	stateMap["default_project_creation"] = settings.DefaultProjectCreation
	stateMap["default_project_visibility"] = settings.DefaultProjectVisibility
	stateMap["default_projects_limit"] = settings.DefaultProjectsLimit
	stateMap["default_snippet_visibility"] = settings.DefaultSnippetVisibility
	stateMap["default_syntax_highlighting_theme"] = settings.DefaultSyntaxHighlightingTheme
	stateMap["delete_inactive_projects"] = settings.DeleteInactiveProjects
	stateMap["delete_unconfirmed_users"] = settings.DeleteUnconfirmedUsers
	stateMap["deletion_adjourned_period"] = settings.DeletionAdjournedPeriod
	stateMap["diagramsnet_enabled"] = settings.DiagramsnetEnabled
	stateMap["diagramsnet_url"] = settings.DiagramsnetURL
	stateMap["diff_max_patch_bytes"] = settings.DiffMaxPatchBytes
	stateMap["diff_max_files"] = settings.DiffMaxFiles
	stateMap["diff_max_lines"] = settings.DiffMaxLines
	stateMap["disable_admin_oauth_scopes"] = settings.DisableAdminOAuthScopes
	stateMap["disable_feed_token"] = settings.DisableFeedToken
	stateMap["disable_personal_access_tokens"] = settings.DisablePersonalAccessTokens
	stateMap["disabled_oauth_sign_in_sources"] = settings.DisabledOauthSignInSources
	stateMap["dns_rebinding_protection_enabled"] = settings.DNSRebindingProtectionEnabled
	stateMap["domain_denylist_enabled"] = settings.DomainDenylistEnabled
	stateMap["domain_denylist"] = settings.DomainDenylist
	stateMap["domain_allowlist"] = settings.DomainAllowlist
	stateMap["downstream_pipeline_trigger_limit_per_project_user_sha"] = settings.DownstreamPipelineTriggerLimitPerProjectUserSHA
	stateMap["dsa_key_restriction"] = settings.DSAKeyRestriction
	stateMap["duo_features_enabled"] = settings.DuoFeaturesEnabled
	stateMap["ecdsa_key_restriction"] = settings.ECDSAKeyRestriction
	stateMap["ecdsa_sk_key_restriction"] = settings.ECDSASKKeyRestriction
	stateMap["ed25519_key_restriction"] = settings.Ed25519KeyRestriction
	stateMap["ed25519_sk_key_restriction"] = settings.Ed25519SKKeyRestriction
	stateMap["eks_access_key_id"] = settings.EKSAccessKeyID
	stateMap["eks_account_id"] = settings.EKSAccountID
	stateMap["eks_integration_enabled"] = settings.EKSIntegrationEnabled
	stateMap["eks_secret_access_key"] = settings.EKSSecretAccessKey
	stateMap["elasticsearch_aws_access_key"] = settings.ElasticsearchAWSAccessKey
	stateMap["elasticsearch_aws_region"] = settings.ElasticsearchAWSRegion
	stateMap["elasticsearch_aws_secret_access_key"] = settings.ElasticsearchAWSSecretAccessKey
	stateMap["elasticsearch_aws"] = settings.ElasticsearchAWS
	stateMap["elasticsearch_indexed_field_length_limit"] = settings.ElasticsearchIndexedFieldLengthLimit
	stateMap["elasticsearch_indexed_file_size_limit_kb"] = settings.ElasticsearchIndexedFileSizeLimitKB
	stateMap["elasticsearch_indexing"] = settings.ElasticsearchIndexing
	stateMap["elasticsearch_limit_indexing"] = settings.ElasticsearchLimitIndexing
	stateMap["elasticsearch_max_bulk_concurrency"] = settings.ElasticsearchMaxBulkConcurrency
	stateMap["elasticsearch_max_bulk_size_mb"] = settings.ElasticsearchMaxBulkSizeMB
	stateMap["elasticsearch_max_code_indexing_concurrency"] = settings.ElasticsearchMaxCodeIndexingConcurrency
	stateMap["elasticsearch_namespace_ids"] = settings.ElasticsearchNamespaceIDs
	stateMap["elasticsearch_project_ids"] = settings.ElasticsearchProjectIDs
	stateMap["elasticsearch_requeue_workers"] = settings.ElasticsearchRequeueWorkers
	stateMap["elasticsearch_search"] = settings.ElasticsearchSearch
	stateMap["elasticsearch_url"] = settings.ElasticsearchURL
	stateMap["elasticsearch_username"] = settings.ElasticsearchUsername
	stateMap["elasticsearch_password"] = settings.ElasticsearchPassword
	stateMap["elasticsearch_worker_number_of_shards"] = settings.ElasticsearchWorkerNumberOfShards
	stateMap["email_additional_text"] = settings.EmailAdditionalText
	stateMap["email_author_in_body"] = settings.EmailAuthorInBody
	stateMap["email_confirmation_setting"] = settings.EmailConfirmationSetting
	stateMap["enable_artifact_external_redirect_warning_page"] = settings.EnableArtifactExternalRedirectWarningPage
	stateMap["enabled_git_access_protocol"] = settings.EnabledGitAccessProtocol
	stateMap["enforce_namespace_storage_limit"] = settings.EnforceNamespaceStorageLimit
	stateMap["enforce_terms"] = settings.EnforceTerms
	stateMap["external_auth_client_cert"] = settings.ExternalAuthClientCert
	stateMap["external_auth_client_key_pass"] = settings.ExternalAuthClientKeyPass
	stateMap["external_auth_client_key"] = settings.ExternalAuthClientKey
	stateMap["external_authorization_service_default_label"] = settings.ExternalAuthorizationServiceDefaultLabel
	stateMap["external_authorization_service_enabled"] = settings.ExternalAuthorizationServiceEnabled
	stateMap["external_authorization_service_timeout"] = settings.ExternalAuthorizationServiceTimeout
	stateMap["external_authorization_service_url"] = settings.ExternalAuthorizationServiceURL
	stateMap["external_pipeline_validation_service_url"] = settings.ExternalPipelineValidationServiceURL
	stateMap["external_pipeline_validation_service_token"] = settings.ExternalPipelineValidationServiceToken
	stateMap["external_pipeline_validation_service_timeout"] = settings.ExternalPipelineValidationServiceTimeout
	stateMap["failed_login_attempts_unlock_period_in_minutes"] = settings.FailedLoginAttemptsUnlockPeriodInMinutes
	stateMap["file_template_project_id"] = settings.FileTemplateProjectID
	stateMap["first_day_of_week"] = settings.FirstDayOfWeek
	stateMap["geo_node_allowed_ips"] = settings.GeoNodeAllowedIPs
	stateMap["geo_status_timeout"] = settings.GeoStatusTimeout
	stateMap["git_rate_limit_users_alertlist"] = settings.GitRateLimitUsersAlertlist
	stateMap["git_rate_limit_users_allowlist"] = settings.GitRateLimitUsersAllowlist
	stateMap["git_two_factor_session_expiry"] = settings.GitTwoFactorSessionExpiry
	stateMap["gitaly_timeout_default"] = settings.GitalyTimeoutDefault
	stateMap["gitaly_timeout_fast"] = settings.GitalyTimeoutFast
	stateMap["gitaly_timeout_medium"] = settings.GitalyTimeoutMedium
	stateMap["gitlab_dedicated_instance"] = settings.GitlabDedicatedInstance
	stateMap["gitlab_environment_toolkit_instance"] = settings.GitlabEnvironmentToolkitInstance
	stateMap["gitlab_shell_operation_limit"] = settings.GitlabShellOperationLimit
	stateMap["gitpod_enabled"] = settings.GitpodEnabled
	stateMap["gitpod_url"] = settings.GitpodURL
	stateMap["globally_allowed_ips"] = settings.GloballyAllowedIPs
	stateMap["grafana_enabled"] = settings.GrafanaEnabled
	stateMap["grafana_url"] = settings.GrafanaURL
	stateMap["gravatar_enabled"] = settings.GravatarEnabled
	stateMap["group_owners_can_manage_default_branch_protection"] = settings.GroupOwnersCanManageDefaultBranchProtection
	stateMap["help_page_hide_commercial_content"] = settings.HelpPageHideCommercialContent
	stateMap["help_page_support_url"] = settings.HelpPageSupportURL
	stateMap["help_page_text"] = settings.HelpPageText
	stateMap["help_text"] = settings.HelpText
	stateMap["hide_third_party_offers"] = settings.HideThirdPartyOffers
	stateMap["home_page_url"] = settings.HomePageURL
	stateMap["housekeeping_enabled"] = settings.HousekeepingEnabled
	stateMap["housekeeping_optimize_repository_period"] = settings.HousekeepingOptimizeRepositoryPeriod
	stateMap["html_emails_enabled"] = settings.HTMLEmailsEnabled
	stateMap["import_sources"] = settings.ImportSources
	stateMap["in_product_marketing_emails_enabled"] = settings.InProductMarketingEmailsEnabled
	stateMap["inactive_projects_delete_after_months"] = settings.InactiveProjectsDeleteAfterMonths
	stateMap["inactive_projects_min_size_mb"] = settings.InactiveProjectsMinSizeMB
	stateMap["inactive_projects_send_warning_email_after_months"] = settings.InactiveProjectsSendWarningEmailAfterMonths
	stateMap["include_optional_metrics_in_service_ping"] = settings.IncludeOptionalMetricsInServicePing
	stateMap["invisible_captcha_enabled"] = settings.InvisibleCaptchaEnabled
	stateMap["issues_create_limit"] = settings.IssuesCreateLimit
	stateMap["jira_connect_application_key"] = settings.JiraConnectApplicationKey
	stateMap["jira_connect_proxy_url"] = settings.JiraConnectProxyURL
	stateMap["jira_connect_public_key_storage_enabled"] = settings.JiraConnectPublicKeyStorageEnabled
	stateMap["keep_latest_artifact"] = settings.KeepLatestArtifact
	stateMap["local_markdown_version"] = settings.LocalMarkdownVersion
	stateMap["lock_memberships_to_ldap"] = settings.LockMembershipsToLDAP
	stateMap["lock_duo_features_enabled"] = settings.LockDuoFeaturesEnabled
	stateMap["mailgun_signing_key"] = settings.MailgunSigningKey
	stateMap["mailgun_events_enabled"] = settings.MailgunEventsEnabled
	stateMap["maintenance_mode_message"] = settings.MaintenanceModeMessage
	stateMap["maven_package_requests_forwarding"] = settings.MavenPackageRequestsForwarding
	stateMap["maintenance_mode"] = settings.MaintenanceMode
	stateMap["max_artifacts_size"] = settings.MaxArtifactsSize
	stateMap["max_attachment_size"] = settings.MaxAttachmentSize
	stateMap["max_decompressed_archive_size"] = settings.MaxDecompressedArchiveSize
	stateMap["max_export_size"] = settings.MaxExportSize
	stateMap["max_import_remote_file_size"] = settings.MaxImportRemoteFileSize
	stateMap["max_import_size"] = settings.MaxImportSize
	stateMap["max_login_attempts"] = settings.MaxLoginAttempts
	stateMap["max_pages_size"] = settings.MaxPagesSize
	stateMap["max_personal_access_token_lifetime"] = settings.MaxPersonalAccessTokenLifetime
	stateMap["max_ssh_key_lifetime"] = settings.MaxSSHKeyLifetime
	stateMap["max_terraform_state_size_bytes"] = settings.MaxTerraformStateSizeBytes
	stateMap["metrics_method_call_threshold"] = settings.MetricsMethodCallThreshold
	stateMap["max_number_of_repository_downloads"] = settings.MaxNumberOfRepositoryDownloads
	stateMap["max_number_of_repository_downloads_within_time_period"] = settings.MaxNumberOfRepositoryDownloadsWithinTimePeriod
	stateMap["mirror_available"] = settings.MirrorAvailable
	stateMap["mirror_capacity_threshold"] = settings.MirrorCapacityThreshold
	stateMap["mirror_max_capacity"] = settings.MirrorMaxCapacity
	stateMap["mirror_max_delay"] = settings.MirrorMaxDelay
	stateMap["notify_on_unknown_sign_in"] = settings.NotifyOnUnknownSignIn
	stateMap["npm_package_requests_forwarding"] = settings.NPMPackageRequestsForwarding
	stateMap["nuget_skip_metadata_url_validation"] = settings.NugetSkipMetadataURLValidation
	stateMap["outbound_local_requests_whitelist"] = settings.OutboundLocalRequestsWhitelist
	stateMap["package_metadata_purl_types"] = settings.PackageMetadataPURLTypes
	stateMap["package_registry_allow_anyone_to_pull_option"] = settings.PackageRegistryAllowAnyoneToPullOption
	stateMap["pages_domain_verification_enabled"] = settings.PagesDomainVerificationEnabled
	stateMap["password_authentication_enabled_for_git"] = settings.PasswordAuthenticationEnabledForGit
	stateMap["password_authentication_enabled_for_web"] = settings.PasswordAuthenticationEnabledForWeb
	stateMap["minimum_password_length"] = settings.MinimumPasswordLength
	stateMap["password_number_required"] = settings.PasswordNumberRequired
	stateMap["password_symbol_required"] = settings.PasswordSymbolRequired
	stateMap["password_uppercase_required"] = settings.PasswordUppercaseRequired
	stateMap["password_lowercase_required"] = settings.PasswordLowercaseRequired
	stateMap["performance_bar_allowed_group_path"] = settings.PerformanceBarAllowedGroupPath
	stateMap["personal_access_token_prefix"] = settings.PersonalAccessTokenPrefix
	stateMap["pipeline_limit_per_project_user_sha"] = settings.PipelineLimitPerProjectUserSha
	stateMap["plantuml_enabled"] = settings.PlantumlEnabled
	stateMap["plantuml_url"] = settings.PlantumlURL
	stateMap["polling_interval_multiplier"] = settings.PollingIntervalMultiplier
	stateMap["project_export_enabled"] = settings.ProjectExportEnabled
	stateMap["project_jobs_api_rate_limit"] = settings.ProjectJobsAPIRateLimit
	stateMap["projects_api_rate_limit_unauthenticated"] = settings.ProjectsAPIRateLimitUnauthenticated
	stateMap["prometheus_metrics_enabled"] = settings.PrometheusMetricsEnabled
	stateMap["protected_ci_variables"] = settings.ProtectedCIVariables
	stateMap["push_event_activities_limit"] = settings.PushEventActivitiesLimit
	stateMap["push_event_hooks_limit"] = settings.PushEventHooksLimit
	stateMap["pypi_package_requests_forwarding"] = settings.PyPIPackageRequestsForwarding
	stateMap["rate_limiting_response_text"] = settings.RateLimitingResponseText
	stateMap["raw_blob_request_limit"] = settings.RawBlobRequestLimit
	stateMap["search_rate_limit"] = settings.SearchRateLimit
	stateMap["search_rate_limit_unauthenticated"] = settings.SearchRateLimitUnauthenticated
	stateMap["recaptcha_enabled"] = settings.RecaptchaEnabled
	stateMap["recaptcha_private_key"] = settings.RecaptchaPrivateKey
	stateMap["recaptcha_site_key"] = settings.RecaptchaSiteKey
	stateMap["receive_max_input_size"] = settings.ReceiveMaxInputSize
	stateMap["receptive_cluster_agents_enabled"] = settings.ReceptiveClusterAgentsEnabled
	stateMap["remember_me_enabled"] = settings.RememberMeEnabled
	stateMap["repository_checks_enabled"] = settings.RepositoryChecksEnabled
	stateMap["repository_size_limit"] = settings.RepositorySizeLimit
	stateMap["repository_storages_weighted"] = settings.RepositoryStoragesWeighted
	stateMap["require_admin_approval_after_user_signup"] = settings.RequireAdminApprovalAfterUserSignup
	stateMap["require_admin_two_factor_authentication"] = settings.RequireAdminTwoFactorAuthentication
	stateMap["require_personal_access_token_expiry"] = settings.RequirePersonalAccessTokenExpiry
	stateMap["require_two_factor_authentication"] = settings.RequireTwoFactorAuthentication
	stateMap["restricted_visibility_levels"] = settings.RestrictedVisibilityLevels
	stateMap["rsa_key_restriction"] = settings.RSAKeyRestriction
	stateMap["security_approval_policies_limit"] = settings.SecurityApprovalPoliciesLimit
	stateMap["security_policy_global_group_approvers_enabled"] = settings.SecurityPolicyGlobalGroupApproversEnabled
	stateMap["security_txt_content"] = settings.SecurityTXTContent
	stateMap["send_user_confirmation_email"] = settings.SendUserConfirmationEmail
	stateMap["service_access_tokens_expiration_enforced"] = settings.ServiceAccessTokensExpirationEnforced
	stateMap["session_expire_delay"] = settings.SessionExpireDelay
	stateMap["shared_runners_enabled"] = settings.SharedRunnersEnabled
	stateMap["shared_runners_minutes"] = settings.SharedRunnersMinutes
	stateMap["shared_runners_text"] = settings.SharedRunnersText
	stateMap["sidekiq_job_limiter_mode"] = settings.SidekiqJobLimiterMode
	stateMap["sidekiq_job_limiter_compression_threshold_bytes"] = settings.SidekiqJobLimiterCompressionThresholdBytes
	stateMap["sidekiq_job_limiter_limit_bytes"] = settings.SidekiqJobLimiterLimitBytes
	stateMap["sign_in_text"] = settings.SignInText
	stateMap["signup_enabled"] = settings.SignupEnabled
	stateMap["silent_admin_exports_enabled"] = settings.SilentAdminExportsEnabled
	stateMap["silent_mode_enabled"] = settings.SilentModeEnabled
	stateMap["slack_app_enabled"] = settings.SlackAppEnabled
	stateMap["slack_app_id"] = settings.SlackAppID
	stateMap["slack_app_secret"] = settings.SlackAppSecret
	stateMap["slack_app_signing_secret"] = settings.SlackAppSigningSecret
	stateMap["slack_app_verification_token"] = settings.SlackAppVerificationToken
	stateMap["snippet_size_limit"] = settings.SnippetSizeLimit
	stateMap["snowplow_app_id"] = settings.SnowplowAppID
	stateMap["snowplow_collector_hostname"] = settings.SnowplowCollectorHostname
	stateMap["snowplow_cookie_domain"] = settings.SnowplowCookieDomain
	stateMap["snowplow_database_collector_hostname"] = settings.SnowplowDatabaseCollectorHostname
	stateMap["snowplow_enabled"] = settings.SnowplowEnabled
	stateMap["sourcegraph_enabled"] = settings.SourcegraphEnabled
	stateMap["sourcegraph_public_only"] = settings.SourcegraphPublicOnly
	stateMap["sourcegraph_url"] = settings.SourcegraphURL
	stateMap["spam_check_endpoint_enabled"] = settings.SpamCheckEndpointEnabled
	stateMap["spam_check_endpoint_url"] = settings.SpamCheckEndpointURL
	stateMap["spam_check_api_key"] = settings.SpamCheckAPIKey
	stateMap["static_objects_external_storage_auth_token"] = settings.StaticObjectsExternalStorageAuthToken
	stateMap["static_objects_external_storage_url"] = settings.StaticObjectsExternalStorageURL
	stateMap["suggest_pipeline_enabled"] = settings.SuggestPipelineEnabled
	stateMap["terminal_max_session_time"] = settings.TerminalMaxSessionTime
	stateMap["terms"] = settings.Terms
	stateMap["throttle_authenticated_api_enabled"] = settings.ThrottleAuthenticatedAPIEnabled
	stateMap["throttle_authenticated_api_period_in_seconds"] = settings.ThrottleAuthenticatedAPIPeriodInSeconds
	stateMap["throttle_authenticated_api_requests_per_period"] = settings.ThrottleAuthenticatedAPIRequestsPerPeriod
	stateMap["throttle_authenticated_packages_api_enabled"] = settings.ThrottleAuthenticatedPackagesAPIEnabled
	stateMap["throttle_authenticated_packages_api_period_in_seconds"] = settings.ThrottleAuthenticatedPackagesAPIPeriodInSeconds
	stateMap["throttle_authenticated_packages_api_requests_per_period"] = settings.ThrottleAuthenticatedPackagesAPIRequestsPerPeriod
	stateMap["throttle_authenticated_web_enabled"] = settings.ThrottleAuthenticatedWebEnabled
	stateMap["throttle_authenticated_web_period_in_seconds"] = settings.ThrottleAuthenticatedWebPeriodInSeconds
	stateMap["throttle_authenticated_web_requests_per_period"] = settings.ThrottleAuthenticatedWebRequestsPerPeriod
	stateMap["throttle_unauthenticated_api_enabled"] = settings.ThrottleUnauthenticatedAPIEnabled
	stateMap["throttle_unauthenticated_api_period_in_seconds"] = settings.ThrottleUnauthenticatedAPIPeriodInSeconds
	stateMap["throttle_unauthenticated_api_requests_per_period"] = settings.ThrottleUnauthenticatedAPIRequestsPerPeriod
	stateMap["throttle_unauthenticated_packages_api_enabled"] = settings.ThrottleUnauthenticatedPackagesAPIEnabled
	stateMap["throttle_unauthenticated_packages_api_period_in_seconds"] = settings.ThrottleUnauthenticatedPackagesAPIPeriodInSeconds
	stateMap["throttle_unauthenticated_packages_api_requests_per_period"] = settings.ThrottleUnauthenticatedPackagesAPIRequestsPerPeriod
	stateMap["throttle_unauthenticated_web_enabled"] = settings.ThrottleUnauthenticatedWebEnabled
	stateMap["throttle_unauthenticated_web_period_in_seconds"] = settings.ThrottleUnauthenticatedWebPeriodInSeconds
	stateMap["throttle_unauthenticated_web_requests_per_period"] = settings.ThrottleUnauthenticatedWebRequestsPerPeriod
	stateMap["time_tracking_limit_to_hours"] = settings.TimeTrackingLimitToHours
	stateMap["two_factor_grace_period"] = settings.TwoFactorGracePeriod
	stateMap["unconfirmed_users_delete_after_days"] = settings.UnconfirmedUsersDeleteAfterDays
	stateMap["unique_ips_limit_enabled"] = settings.UniqueIPsLimitEnabled
	stateMap["unique_ips_limit_per_user"] = settings.UniqueIPsLimitPerUser
	stateMap["unique_ips_limit_time_window"] = settings.UniqueIPsLimitTimeWindow
	stateMap["update_runner_versions_enabled"] = settings.UpdateRunnerVersionsEnabled
	stateMap["usage_ping_enabled"] = settings.UsagePingEnabled
	stateMap["use_clickhouse_for_analytics"] = settings.UseClickhouseForAnalytics
	stateMap["user_deactivation_emails_enabled"] = settings.UserDeactivationEmailsEnabled
	stateMap["user_default_external"] = settings.UserDefaultExternal
	stateMap["user_default_internal_regex"] = settings.UserDefaultInternalRegex
	stateMap["user_defaults_to_private_profile"] = settings.UserDefaultsToPrivateProfile
	stateMap["user_oauth_applications"] = settings.UserOauthApplications
	stateMap["user_show_add_ssh_key_message"] = settings.UserShowAddSSHKeyMessage
	stateMap["valid_runner_registrars"] = settings.ValidRunnerRegistrars
	stateMap["version_check_enabled"] = settings.VersionCheckEnabled
	stateMap["whats_new_variant"] = settings.WhatsNewVariant
	stateMap["web_ide_clientside_preview_enabled"] = settings.WebIDEClientsidePreviewEnabled
	stateMap["wiki_page_max_content_bytes"] = settings.WikiPageMaxContentBytes

	stateMap["default_branch_protection_defaults"] = flattenDefaultBranchProtectionDefaults(settings.DefaultBranchProtectionDefaults)
	return stateMap
}

// Flattens the default branch protection into a statement for easier storage.
func flattenDefaultBranchProtectionDefaults(input *gitlab.BranchProtectionDefaults) (values []map[string]any) {
	v := map[string]any{}
	v["allow_force_push"] = input.AllowForcePush
	v["developer_can_initial_push"] = input.DeveloperCanInitialPush
	if len(input.AllowedToMerge) > 0 {
		list := []int{}
		for _, v := range input.AllowedToMerge {
			list = append(list, int(*v.AccessLevel))
		}

		v["allowed_to_merge"] = list
	}
	if len(input.AllowedToPush) > 0 {
		list := []int{}
		for _, v := range input.AllowedToPush {
			list = append(list, int(*v.AccessLevel))
		}

		v["allowed_to_push"] = list
	}

	values = append(values, v)

	return values
}

func gitlabApplicationSettingsToUpdateOptions(d *schema.ResourceData) *gitlab.UpdateSettingsOptions {
	options := gitlab.UpdateSettingsOptions{}

	if d.HasChange("admin_mode") {
		options.AdminMode = gitlab.Ptr(d.Get("admin_mode").(bool))
	}

	if d.HasChange("abuse_notification_email") {
		options.AbuseNotificationEmail = gitlab.Ptr(d.Get("abuse_notification_email").(string))
	}

	if d.HasChange("after_sign_out_path") {
		options.AfterSignOutPath = gitlab.Ptr(d.Get("after_sign_out_path").(string))
	}

	if d.HasChange("after_sign_up_text") {
		options.AfterSignUpText = gitlab.Ptr(d.Get("after_sign_up_text").(string))
	}

	if d.HasChange("akismet_api_key") {
		options.AkismetAPIKey = gitlab.Ptr(d.Get("akismet_api_key").(string))
	}

	if d.HasChange("akismet_enabled") {
		options.AkismetEnabled = gitlab.Ptr(d.Get("akismet_enabled").(bool))
	}

	if d.HasChange("allow_account_deletion") {
		options.AllowAccountDeletion = gitlab.Ptr(d.Get("allow_account_deletion").(bool))
	}

	if d.HasChange("allow_group_owners_to_manage_ldap") {
		options.AllowGroupOwnersToManageLDAP = gitlab.Ptr(d.Get("allow_group_owners_to_manage_ldap").(bool))
	}

	if d.HasChange("allow_local_requests_from_system_hooks") {
		options.AllowLocalRequestsFromSystemHooks = gitlab.Ptr(d.Get("allow_local_requests_from_system_hooks").(bool))
	}

	if d.HasChange("allow_local_requests_from_web_hooks_and_services") {
		options.AllowLocalRequestsFromWebHooksAndServices = gitlab.Ptr(d.Get("allow_local_requests_from_web_hooks_and_services").(bool))
	}

	if d.HasChange("allow_project_creation_for_guest_and_below") {
		options.AllowProjectCreationForGuestAndBelow = gitlab.Ptr(d.Get("allow_project_creation_for_guest_and_below").(bool))
	}

	if d.HasChange("allow_runner_registration_token") {
		options.AllowRunnerRegistrationToken = gitlab.Ptr(d.Get("allow_runner_registration_token").(bool))
	}

	if d.HasChange("archive_builds_in_human_readable") {
		options.ArchiveBuildsInHumanReadable = gitlab.Ptr(d.Get("archive_builds_in_human_readable").(string))
	}

	if d.HasChange("asciidoc_max_includes") {
		options.ASCIIDocMaxIncludes = gitlab.Ptr(int64(d.Get("asciidoc_max_includes").(int)))
	}

	if d.HasChange("asset_proxy_enabled") {
		options.AssetProxyEnabled = gitlab.Ptr(d.Get("asset_proxy_enabled").(bool))
	}

	if d.HasChange("asset_proxy_secret_key") {
		options.AssetProxySecretKey = gitlab.Ptr(d.Get("asset_proxy_secret_key").(string))
	}

	if d.HasChange("asset_proxy_url") {
		options.AssetProxyURL = gitlab.Ptr(d.Get("asset_proxy_url").(string))
	}

	if d.HasChange("asset_proxy_allowlist") {
		options.AssetProxyAllowlist = stringListToStringSlice(d.Get("asset_proxy_allowlist").([]any))
	}

	if d.HasChange("authorized_keys_enabled") {
		options.AuthorizedKeysEnabled = gitlab.Ptr(d.Get("authorized_keys_enabled").(bool))
	}

	if d.HasChange("auto_ban_user_on_excessive_projects_download") {
		options.AutoBanUserOnExcessiveProjectsDownload = gitlab.Ptr(d.Get("auto_ban_user_on_excessive_projects_download").(bool))
	}

	if d.HasChange("auto_devops_domain") {
		options.AutoDevOpsDomain = gitlab.Ptr(d.Get("auto_devops_domain").(string))
	}

	if d.HasChange("auto_devops_enabled") {
		options.AutoDevOpsEnabled = gitlab.Ptr(d.Get("auto_devops_enabled").(bool))
	}

	if d.HasChange("automatic_purchased_storage_allocation") {
		options.AutomaticPurchasedStorageAllocation = gitlab.Ptr(d.Get("automatic_purchased_storage_allocation").(bool))
	}

	if d.HasChange("bulk_import_concurrent_pipeline_batch_limit") {
		options.BulkImportConcurrentPipelineBatchLimit = gitlab.Ptr(int64(d.Get("bulk_import_concurrent_pipeline_batch_limit").(int)))
	}

	if d.HasChange("bulk_import_enabled") {
		options.BulkImportEnabled = gitlab.Ptr(d.Get("bulk_import_enabled").(bool))
	}

	if d.HasChange("bulk_import_max_download_file_size") {
		options.BulkImportMaxDownloadFileSize = gitlab.Ptr(int64(d.Get("bulk_import_max_download_file_size").(int)))
	}

	if d.HasChange("can_create_group") {
		options.CanCreateGroup = gitlab.Ptr(d.Get("can_create_group").(bool))
	}

	if d.HasChange("check_namespace_plan") {
		options.CheckNamespacePlan = gitlab.Ptr(d.Get("check_namespace_plan").(bool))
	}

	if d.HasChange("ci_max_includes") {
		options.CIMaxIncludes = gitlab.Ptr(int64(d.Get("ci_max_includes").(int)))
	}

	if d.HasChange("ci_max_total_yaml_size_bytes") {
		options.CIMaxTotalYAMLSizeBytes = gitlab.Ptr(int64(d.Get("ci_max_total_yaml_size_bytes").(int)))
	}

	if d.HasChange("commit_email_hostname") {
		options.CommitEmailHostname = gitlab.Ptr(d.Get("commit_email_hostname").(string))
	}

	if d.HasChange("concurrent_bitbucket_import_jobs_limit") {
		options.ConcurrentBitbucketImportJobsLimit = gitlab.Ptr(int64(d.Get("concurrent_bitbucket_import_jobs_limit").(int)))
	}

	if d.HasChange("concurrent_bitbucket_server_import_jobs_limit") {
		options.ConcurrentBitbucketServerImportJobsLimit = gitlab.Ptr(int64(d.Get("concurrent_bitbucket_server_import_jobs_limit").(int)))
	}

	if d.HasChange("concurrent_github_import_jobs_limit") {
		options.ConcurrentGitHubImportJobsLimit = gitlab.Ptr(int64(d.Get("concurrent_github_import_jobs_limit").(int)))
	}

	if d.HasChange("container_expiration_policies_enable_historic_entries") {
		options.ContainerExpirationPoliciesEnableHistoricEntries = gitlab.Ptr(d.Get("container_expiration_policies_enable_historic_entries").(bool))
	}

	if d.HasChange("container_registry_cleanup_tags_service_max_list_size") {
		options.ContainerRegistryCleanupTagsServiceMaxListSize = gitlab.Ptr(int64(d.Get("container_registry_cleanup_tags_service_max_list_size").(int)))
	}

	if d.HasChange("container_registry_delete_tags_service_timeout") {
		options.ContainerRegistryDeleteTagsServiceTimeout = gitlab.Ptr(int64(d.Get("container_registry_delete_tags_service_timeout").(int)))
	}

	if d.HasChange("container_registry_expiration_policies_caching") {
		options.ContainerRegistryExpirationPoliciesCaching = gitlab.Ptr(d.Get("container_registry_expiration_policies_caching").(bool))
	}

	if d.HasChange("container_registry_expiration_policies_worker_capacity") {
		options.ContainerRegistryExpirationPoliciesWorkerCapacity = gitlab.Ptr(int64(d.Get("container_registry_expiration_policies_worker_capacity").(int)))
	}

	if d.HasChange("container_registry_token_expire_delay") {
		options.ContainerRegistryTokenExpireDelay = gitlab.Ptr(int64(d.Get("container_registry_token_expire_delay").(int)))
	}

	if d.HasChange("package_registry_cleanup_policies_worker_capacity") {
		options.PackageRegistryCleanupPoliciesWorkerCapacity = gitlab.Ptr(int64(d.Get("package_registry_cleanup_policies_worker_capacity").(int)))
	}

	if d.HasChange("package_metadata_purl_types") {
		options.PackageMetadataPURLTypes = intListToInt64Slice(d.Get("package_metadata_purl_types").([]any))
	}

	if d.HasChange("deactivate_dormant_users") {
		options.DeactivateDormantUsers = gitlab.Ptr(d.Get("deactivate_dormant_users").(bool))
	}

	if d.HasChange("deactivate_dormant_users_period") {
		options.DeactivateDormantUsersPeriod = gitlab.Ptr(int64(d.Get("deactivate_dormant_users_period").(int)))
	}

	if d.HasChange("decompress_archive_file_timeout") {
		options.DecompressArchiveFileTimeout = gitlab.Ptr(int64(d.Get("decompress_archive_file_timeout").(int)))
	}

	if d.HasChange("default_artifacts_expire_in") {
		options.DefaultArtifactsExpireIn = gitlab.Ptr(d.Get("default_artifacts_expire_in").(string))
	}

	if d.HasChange("default_branch_name") {
		options.DefaultBranchName = gitlab.Ptr(d.Get("default_branch_name").(string))
	}

	if d.HasChange("default_branch_protection") {
		options.DefaultBranchProtection = gitlab.Ptr(int64(d.Get("default_branch_protection").(int))) //nolint:staticcheck
	}

	if d.HasChange("default_branch_protection_defaults") {
		// only one struct is allowed here, so retrieve the first one.
		values := d.Get("default_branch_protection_defaults.0").(map[string]any)

		// Read the allowed to push and convert to []*gitlab.GroupAccessLevel
		allowedToPushList := values["allowed_to_push"].([]any)
		allowedToPush := make([]*gitlab.GroupAccessLevel, len(allowedToPushList))
		for k, v := range allowedToPushList {
			allowedToPush[k] = &gitlab.GroupAccessLevel{
				AccessLevel: gitlab.Ptr(gitlab.AccessLevelValue(v.(int))),
			}
		}

		// Read the allowed to merge and convert to []*gitlab.GroupAccessLevel
		allowedToMergeList := values["allowed_to_merge"].([]any)
		allowedToMerge := make([]*gitlab.GroupAccessLevel, len(allowedToMergeList))
		for k, v := range allowedToMergeList {
			allowedToMerge[k] = &gitlab.GroupAccessLevel{
				AccessLevel: gitlab.Ptr(gitlab.AccessLevelValue(v.(int))),
			}
		}

		branchProtectionDefault := &gitlab.DefaultBranchProtectionDefaultsOptions{
			DeveloperCanInitialPush: gitlab.Ptr(values["developer_can_initial_push"].(bool)),
			AllowForcePush:          gitlab.Ptr(values["allow_force_push"].(bool)),
			AllowedToPush:           &allowedToPush,
			AllowedToMerge:          &allowedToMerge,
		}
		options.DefaultBranchProtectionDefaults = branchProtectionDefault
	}

	if d.HasChange("default_ci_config_path") {
		options.DefaultCiConfigPath = gitlab.Ptr(d.Get("default_ci_config_path").(string))
	}

	if d.HasChange("default_group_visibility") {
		options.DefaultGroupVisibility = stringToVisibilityLevel(d.Get("default_group_visibility").(string))
	}

	if d.HasChange("default_preferred_language") {
		options.DefaultPreferredLanguage = gitlab.Ptr(d.Get("default_preferred_language").(string))
	}

	if d.HasChange("default_project_creation") {
		options.DefaultProjectCreation = gitlab.Ptr(int64(d.Get("default_project_creation").(int)))
	}

	if d.HasChange("default_project_visibility") {
		options.DefaultProjectVisibility = stringToVisibilityLevel(d.Get("default_project_visibility").(string))
	}

	if d.HasChange("default_projects_limit") {
		options.DefaultProjectsLimit = gitlab.Ptr(int64(d.Get("default_projects_limit").(int)))
	}

	if d.HasChange("default_snippet_visibility") {
		options.DefaultSnippetVisibility = stringToVisibilityLevel(d.Get("default_snippet_visibility").(string))
	}

	if d.HasChange("default_syntax_highlighting_theme") {
		options.DefaultSyntaxHighlightingTheme = gitlab.Ptr(int64(d.Get("default_syntax_highlighting_theme").(int)))
	}

	if d.HasChange("delete_inactive_projects") {
		options.DeleteInactiveProjects = gitlab.Ptr(d.Get("delete_inactive_projects").(bool))
	}

	if d.HasChange("delete_unconfirmed_users") {
		options.DeleteUnconfirmedUsers = gitlab.Ptr(d.Get("delete_unconfirmed_users").(bool))
	}

	if d.HasChange("deletion_adjourned_period") {
		options.DeletionAdjournedPeriod = gitlab.Ptr(int64(d.Get("deletion_adjourned_period").(int)))
	}

	if d.HasChange("diagramsnet_enabled") {
		options.DiagramsnetEnabled = gitlab.Ptr(d.Get("diagramsnet_enabled").(bool))
	}

	if d.HasChange("diagramsnet_url") {
		options.DiagramsnetURL = gitlab.Ptr(d.Get("diagramsnet_url").(string))
	}

	if d.HasChange("diff_max_patch_bytes") {
		options.DiffMaxPatchBytes = gitlab.Ptr(int64(d.Get("diff_max_patch_bytes").(int)))
	}

	if d.HasChange("diff_max_files") {
		options.DiffMaxFiles = gitlab.Ptr(int64(d.Get("diff_max_files").(int)))
	}

	if d.HasChange("diff_max_lines") {
		options.DiffMaxLines = gitlab.Ptr(int64(d.Get("diff_max_lines").(int)))
	}

	if d.HasChange("disable_admin_oauth_scopes") {
		options.DisableAdminOAuthScopes = gitlab.Ptr(d.Get("disable_admin_oauth_scopes").(bool))
	}

	if d.HasChange("disable_feed_token") {
		options.DisableFeedToken = gitlab.Ptr(d.Get("disable_feed_token").(bool))
	}

	if d.HasChange("disable_personal_access_tokens") {
		options.DisablePersonalAccessTokens = gitlab.Ptr(d.Get("disable_personal_access_tokens").(bool))
	}

	if d.HasChange("disabled_oauth_sign_in_sources") {
		options.DisabledOauthSignInSources = stringListToStringSlice(d.Get("disabled_oauth_sign_in_sources").([]any))
	}

	if d.HasChange("dns_rebinding_protection_enabled") {
		options.DNSRebindingProtectionEnabled = gitlab.Ptr(d.Get("dns_rebinding_protection_enabled").(bool))
	}

	if d.HasChange("domain_denylist_enabled") {
		options.DomainDenylistEnabled = gitlab.Ptr(d.Get("domain_denylist_enabled").(bool))
	}

	if d.HasChange("domain_denylist") {
		options.DomainDenylist = stringListToStringSlice(d.Get("domain_denylist").([]any))
	}

	if d.HasChange("domain_allowlist") {
		options.DomainAllowlist = stringListToStringSlice(d.Get("domain_allowlist").([]any))
	}

	if d.HasChange("downstream_pipeline_trigger_limit_per_project_user_sha") {
		options.DownstreamPipelineTriggerLimitPerProjectUserSHA = gitlab.Ptr(int64(d.Get("downstream_pipeline_trigger_limit_per_project_user_sha").(int)))
	}

	if d.HasChange("duo_features_enabled") {
		options.DuoFeaturesEnabled = gitlab.Ptr(d.Get("duo_features_enabled").(bool))
	}

	if d.HasChange("dsa_key_restriction") {
		options.DSAKeyRestriction = gitlab.Ptr(int64(d.Get("dsa_key_restriction").(int)))
	}

	if d.HasChange("ecdsa_key_restriction") {
		options.ECDSAKeyRestriction = gitlab.Ptr(int64(d.Get("ecdsa_key_restriction").(int)))
	}

	if d.HasChange("ecdsa_sk_key_restriction") {
		options.ECDSASKKeyRestriction = gitlab.Ptr(int64(d.Get("ecdsa_sk_key_restriction").(int)))
	}

	if d.HasChange("ed25519_key_restriction") {
		options.Ed25519KeyRestriction = gitlab.Ptr(int64(d.Get("ed25519_key_restriction").(int)))
	}

	if d.HasChange("ed25519_sk_key_restriction") {
		options.Ed25519SKKeyRestriction = gitlab.Ptr(int64(d.Get("ed25519_sk_key_restriction").(int)))
	}

	if d.HasChange("eks_access_key_id") {
		options.EKSAccessKeyID = gitlab.Ptr(d.Get("eks_access_key_id").(string))
	}

	if d.HasChange("eks_account_id") {
		options.EKSAccountID = gitlab.Ptr(d.Get("eks_account_id").(string))
	}

	if d.HasChange("eks_integration_enabled") {
		options.EKSIntegrationEnabled = gitlab.Ptr(d.Get("eks_integration_enabled").(bool))
	}

	if d.HasChange("eks_secret_access_key") {
		options.EKSSecretAccessKey = gitlab.Ptr(d.Get("eks_secret_access_key").(string))
	}

	if d.HasChange("elasticsearch_aws_access_key") {
		options.ElasticsearchAWSAccessKey = gitlab.Ptr(d.Get("elasticsearch_aws_access_key").(string))
	}

	if d.HasChange("elasticsearch_aws_region") {
		options.ElasticsearchAWSRegion = gitlab.Ptr(d.Get("elasticsearch_aws_region").(string))
	}

	if d.HasChange("elasticsearch_aws_secret_access_key") {
		options.ElasticsearchAWSSecretAccessKey = gitlab.Ptr(d.Get("elasticsearch_aws_secret_access_key").(string))
	}

	if d.HasChange("elasticsearch_aws") {
		options.ElasticsearchAWS = gitlab.Ptr(d.Get("elasticsearch_aws").(bool))
	}

	if d.HasChange("elasticsearch_indexed_field_length_limit") {
		options.ElasticsearchIndexedFieldLengthLimit = gitlab.Ptr(int64(d.Get("elasticsearch_indexed_field_length_limit").(int)))
	}

	if d.HasChange("elasticsearch_indexed_file_size_limit_kb") {
		options.ElasticsearchIndexedFileSizeLimitKB = gitlab.Ptr(int64(d.Get("elasticsearch_indexed_file_size_limit_kb").(int)))
	}

	if d.HasChange("elasticsearch_indexing") {
		options.ElasticsearchIndexing = gitlab.Ptr(d.Get("elasticsearch_indexing").(bool))
	}

	if d.HasChange("elasticsearch_limit_indexing") {
		options.ElasticsearchLimitIndexing = gitlab.Ptr(d.Get("elasticsearch_limit_indexing").(bool))
	}

	if d.HasChange("elasticsearch_max_bulk_concurrency") {
		options.ElasticsearchMaxBulkConcurrency = gitlab.Ptr(int64(d.Get("elasticsearch_max_bulk_concurrency").(int)))
	}

	if d.HasChange("elasticsearch_max_bulk_size_mb") {
		options.ElasticsearchMaxBulkSizeMB = gitlab.Ptr(int64(d.Get("elasticsearch_max_bulk_size_mb").(int)))
	}

	if d.HasChange("elasticsearch_namespace_ids") {
		options.ElasticsearchNamespaceIDs = intListToInt64Slice(d.Get("elasticsearch_namespace_ids").([]any))
	}

	if d.HasChange("elasticsearch_project_ids") {
		options.ElasticsearchProjectIDs = intListToInt64Slice(d.Get("elasticsearch_project_ids").([]any))
	}

	if d.HasChange("elasticsearch_search") {
		options.ElasticsearchSearch = gitlab.Ptr(d.Get("elasticsearch_search").(bool))
	}

	if d.HasChange("elasticsearch_url") {
		options.ElasticsearchURL = stringListToCommaSeparatedString(d.Get("elasticsearch_url").([]any))
	}

	if d.HasChange("elasticsearch_username") {
		options.ElasticsearchUsername = gitlab.Ptr(d.Get("elasticsearch_username").(string))
	}

	if d.HasChange("elasticsearch_password") {
		options.ElasticsearchPassword = gitlab.Ptr(d.Get("elasticsearch_password").(string))
	}

	if d.HasChange("email_additional_text") {
		options.EmailAdditionalText = gitlab.Ptr(d.Get("email_additional_text").(string))
	}

	if d.HasChange("email_author_in_body") {
		options.EmailAuthorInBody = gitlab.Ptr(d.Get("email_author_in_body").(bool))
	}

	if d.HasChange("enabled_git_access_protocol") {
		options.EnabledGitAccessProtocol = gitlab.Ptr(d.Get("enabled_git_access_protocol").(string))
	}

	if d.HasChange("enforce_namespace_storage_limit") {
		options.EnforceNamespaceStorageLimit = gitlab.Ptr(d.Get("enforce_namespace_storage_limit").(bool))
	}

	if d.HasChange("enforce_terms") {
		options.EnforceTerms = gitlab.Ptr(d.Get("enforce_terms").(bool))
	}

	if d.HasChange("external_auth_client_cert") {
		options.ExternalAuthClientCert = gitlab.Ptr(d.Get("external_auth_client_cert").(string))
	}

	if d.HasChange("external_auth_client_key_pass") {
		options.ExternalAuthClientKeyPass = gitlab.Ptr(d.Get("external_auth_client_key_pass").(string))
	}

	if d.HasChange("external_auth_client_key") {
		options.ExternalAuthClientKey = gitlab.Ptr(d.Get("external_auth_client_key").(string))
	}

	if d.HasChange("external_authorization_service_default_label") {
		options.ExternalAuthorizationServiceDefaultLabel = gitlab.Ptr(d.Get("external_authorization_service_default_label").(string))
	}

	if d.HasChange("external_authorization_service_enabled") {
		options.ExternalAuthorizationServiceEnabled = gitlab.Ptr(d.Get("external_authorization_service_enabled").(bool))
	}

	if d.HasChange("external_authorization_service_timeout") {
		gv := d.Get("external_authorization_service_timeout").(float64)
		options.ExternalAuthorizationServiceTimeout = &gv
	}

	if d.HasChange("external_authorization_service_url") {
		options.ExternalAuthorizationServiceURL = gitlab.Ptr(d.Get("external_authorization_service_url").(string))
	}

	if d.HasChange("external_pipeline_validation_service_url") {
		options.ExternalPipelineValidationServiceURL = gitlab.Ptr(d.Get("external_pipeline_validation_service_url").(string))
	}

	if d.HasChange("external_pipeline_validation_service_token") {
		options.ExternalPipelineValidationServiceToken = gitlab.Ptr(d.Get("external_pipeline_validation_service_token").(string))
	}

	if d.HasChange("external_pipeline_validation_service_timeout") {
		options.ExternalPipelineValidationServiceTimeout = gitlab.Ptr(int64(d.Get("external_pipeline_validation_service_timeout").(int)))
	}

	if d.HasChange("file_template_project_id") {
		options.FileTemplateProjectID = gitlab.Ptr(int64(d.Get("file_template_project_id").(int)))
	}

	if d.HasChange("first_day_of_week") {
		options.FirstDayOfWeek = gitlab.Ptr(int64(d.Get("first_day_of_week").(int)))
	}

	if d.HasChange("geo_node_allowed_ips") {
		options.GeoNodeAllowedIPs = gitlab.Ptr(d.Get("geo_node_allowed_ips").(string))
	}

	if d.HasChange("geo_status_timeout") {
		options.GeoStatusTimeout = gitlab.Ptr(int64(d.Get("geo_status_timeout").(int)))
	}

	if d.HasChange("git_two_factor_session_expiry") {
		options.GitTwoFactorSessionExpiry = gitlab.Ptr(int64(d.Get("git_two_factor_session_expiry").(int)))
	}

	if d.HasChange("gitaly_timeout_default") {
		options.GitalyTimeoutDefault = gitlab.Ptr(int64(d.Get("gitaly_timeout_default").(int)))
	}

	if d.HasChange("gitaly_timeout_fast") {
		options.GitalyTimeoutFast = gitlab.Ptr(int64(d.Get("gitaly_timeout_fast").(int)))
	}

	if d.HasChange("gitaly_timeout_medium") {
		options.GitalyTimeoutMedium = gitlab.Ptr(int64(d.Get("gitaly_timeout_medium").(int)))
	}

	if d.HasChange("grafana_enabled") {
		options.GrafanaEnabled = gitlab.Ptr(d.Get("grafana_enabled").(bool))
	}

	if d.HasChange("grafana_url") {
		options.GrafanaURL = gitlab.Ptr(d.Get("grafana_url").(string))
	}

	if d.HasChange("gravatar_enabled") {
		options.GravatarEnabled = gitlab.Ptr(d.Get("gravatar_enabled").(bool))
	}

	if d.HasChanges("group_owners_can_manage_default_branch_protection") {
		options.GroupOwnersCanManageDefaultBranchProtection = gitlab.Ptr(d.Get("group_owners_can_manage_default_branch_protection").(bool))
	}

	if d.HasChange("help_page_hide_commercial_content") {
		options.HelpPageHideCommercialContent = gitlab.Ptr(d.Get("help_page_hide_commercial_content").(bool))
	}

	if d.HasChange("help_page_support_url") {
		options.HelpPageSupportURL = gitlab.Ptr(d.Get("help_page_support_url").(string))
	}

	if d.HasChange("help_page_text") {
		options.HelpPageText = gitlab.Ptr(d.Get("help_page_text").(string))
	}

	if d.HasChange("help_text") {
		options.HelpText = gitlab.Ptr(d.Get("help_text").(string))
	}

	if d.HasChange("hide_third_party_offers") {
		options.HideThirdPartyOffers = gitlab.Ptr(d.Get("hide_third_party_offers").(bool))
	}

	if d.HasChange("home_page_url") {
		options.HomePageURL = gitlab.Ptr(d.Get("home_page_url").(string))
	}

	if d.HasChange("housekeeping_enabled") {
		options.HousekeepingEnabled = gitlab.Ptr(d.Get("housekeeping_enabled").(bool))
	}

	if d.HasChange("housekeeping_optimize_repository_period") {
		options.HousekeepingOptimizeRepositoryPeriod = gitlab.Ptr(int64(d.Get("housekeeping_optimize_repository_period").(int)))
	}

	if d.HasChange("html_emails_enabled") {
		options.HTMLEmailsEnabled = gitlab.Ptr(d.Get("html_emails_enabled").(bool))
	}

	if d.HasChange("import_sources") {
		options.ImportSources = stringListToStringSlice(d.Get("import_sources").([]any))
	}

	if d.HasChange("in_product_marketing_emails_enabled") {
		options.InProductMarketingEmailsEnabled = gitlab.Ptr(d.Get("in_product_marketing_emails_enabled").(bool))
	}

	if d.HasChange("inactive_projects_delete_after_months") {
		options.InactiveProjectsDeleteAfterMonths = gitlab.Ptr(int64(d.Get("inactive_projects_delete_after_months").(int)))
	}

	if d.HasChange("inactive_projects_min_size_mb") {
		options.InactiveProjectsMinSizeMB = gitlab.Ptr(int64(d.Get("inactive_projects_min_size_mb").(int)))
	}

	if d.HasChange("inactive_projects_send_warning_email_after_months") {
		options.InactiveProjectsSendWarningEmailAfterMonths = gitlab.Ptr(int64(d.Get("inactive_projects_send_warning_email_after_months").(int)))
	}

	if d.HasChange("invisible_captcha_enabled") {
		options.InvisibleCaptchaEnabled = gitlab.Ptr(d.Get("invisible_captcha_enabled").(bool))
	}

	if d.HasChange("issues_create_limit") {
		options.IssuesCreateLimit = gitlab.Ptr(int64(d.Get("issues_create_limit").(int)))
	}

	if d.HasChange("keep_latest_artifact") {
		options.KeepLatestArtifact = gitlab.Ptr(d.Get("keep_latest_artifact").(bool))
	}

	if d.HasChange("local_markdown_version") {
		options.LocalMarkdownVersion = gitlab.Ptr(int64(d.Get("local_markdown_version").(int)))
	}

	if d.HasChange("lock_memberships_to_ldap") {
		options.LockMembershipsToLDAP = gitlab.Ptr(d.Get("lock_memberships_to_ldap").(bool))
	}

	if d.HasChange("mailgun_signing_key") {
		options.MailgunSigningKey = gitlab.Ptr(d.Get("mailgun_signing_key").(string))
	}

	if d.HasChange("mailgun_events_enabled") {
		options.MailgunEventsEnabled = gitlab.Ptr(d.Get("mailgun_events_enabled").(bool))
	}

	if d.HasChange("maintenance_mode_message") {
		options.MaintenanceModeMessage = gitlab.Ptr(d.Get("maintenance_mode_message").(string))
	}

	if d.HasChange("maintenance_mode") {
		options.MaintenanceMode = gitlab.Ptr(d.Get("maintenance_mode").(bool))
	}

	if d.HasChange("max_artifacts_size") {
		options.MaxArtifactsSize = gitlab.Ptr(int64(d.Get("max_artifacts_size").(int)))
	}

	if d.HasChange("max_attachment_size") {
		options.MaxAttachmentSize = gitlab.Ptr(int64(d.Get("max_attachment_size").(int)))
	}

	if d.HasChange("max_export_size") {
		options.MaxExportSize = gitlab.Ptr(int64(d.Get("max_export_size").(int)))
	}

	if d.HasChange("max_import_size") {
		options.MaxImportSize = gitlab.Ptr(int64(d.Get("max_import_size").(int)))
	}

	if d.HasChange("max_pages_size") {
		options.MaxPagesSize = gitlab.Ptr(int64(d.Get("max_pages_size").(int)))
	}

	if d.HasChange("max_personal_access_token_lifetime") {
		options.MaxPersonalAccessTokenLifetime = gitlab.Ptr(int64(d.Get("max_personal_access_token_lifetime").(int)))
	}

	if d.HasChange("max_ssh_key_lifetime") {
		options.MaxSSHKeyLifetime = gitlab.Ptr(int64(d.Get("max_ssh_key_lifetime").(int)))
	}

	if d.HasChange("max_terraform_state_size_bytes") {
		options.MaxTerraformStateSizeBytes = gitlab.Ptr(int64(d.Get("max_terraform_state_size_bytes").(int)))
	}

	if d.HasChange("metrics_method_call_threshold") {
		options.MetricsMethodCallThreshold = gitlab.Ptr(int64(d.Get("metrics_method_call_threshold").(int)))
	}

	if d.HasChange("max_number_of_repository_downloads") {
		options.MaxNumberOfRepositoryDownloads = gitlab.Ptr(int64(d.Get("max_number_of_repository_downloads").(int)))
	}

	if d.HasChange("max_number_of_repository_downloads_within_time_period") {
		options.MaxNumberOfRepositoryDownloadsWithinTimePeriod = gitlab.Ptr(int64(d.Get("max_number_of_repository_downloads_within_time_period").(int)))
	}

	if d.HasChange("git_rate_limit_users_allowlist") {
		options.GitRateLimitUsersAllowlist = stringListToStringSlice(d.Get("git_rate_limit_users_allowlist").([]any))
	}

	if d.HasChange("mirror_available") {
		options.MirrorAvailable = gitlab.Ptr(d.Get("mirror_available").(bool))
	}

	if d.HasChange("mirror_capacity_threshold") {
		options.MirrorCapacityThreshold = gitlab.Ptr(int64(d.Get("mirror_capacity_threshold").(int)))
	}

	if d.HasChange("mirror_max_capacity") {
		options.MirrorMaxCapacity = gitlab.Ptr(int64(d.Get("mirror_max_capacity").(int)))
	}

	if d.HasChange("mirror_max_delay") {
		options.MirrorMaxDelay = gitlab.Ptr(int64(d.Get("mirror_max_delay").(int)))
	}

	if d.HasChange("notify_on_unknown_sign_in") {
		options.NotifyOnUnknownSignIn = gitlab.Ptr(d.Get("notify_on_unknown_sign_in").(bool))
	}
	
	if d.HasChange("npm_package_requests_forwarding") {
		options.NPMPackageRequestsForwarding = gitlab.Ptr(d.Get("npm_package_requests_forwarding").(bool))
	}

	if d.HasChange("pypi_package_requests_forwarding") {
		options.PyPIPackageRequestsForwarding = gitlab.Ptr(d.Get("pypi_package_requests_forwarding").(bool))
	}

	if d.HasChange("outbound_local_requests_whitelist") {
		options.OutboundLocalRequestsWhitelist = stringListToStringSlice(d.Get("outbound_local_requests_whitelist").([]any))
	}

	if d.HasChange("pages_domain_verification_enabled") {
		options.PagesDomainVerificationEnabled = gitlab.Ptr(d.Get("pages_domain_verification_enabled").(bool))
	}

	if d.HasChange("password_authentication_enabled_for_git") {
		options.PasswordAuthenticationEnabledForGit = gitlab.Ptr(d.Get("password_authentication_enabled_for_git").(bool))
	}

	if d.HasChange("password_authentication_enabled_for_web") {
		options.PasswordAuthenticationEnabledForWeb = gitlab.Ptr(d.Get("password_authentication_enabled_for_web").(bool))
	}

	if d.HasChange("minimum_password_length") {
		options.MinimumPasswordLength = gitlab.Ptr(int64(d.Get("minimum_password_length").(int)))
	}

	if d.HasChange("password_number_required") {
		options.PasswordNumberRequired = gitlab.Ptr(d.Get("password_number_required").(bool))
	}

	if d.HasChange("password_symbol_required") {
		options.PasswordSymbolRequired = gitlab.Ptr(d.Get("password_symbol_required").(bool))
	}

	if d.HasChange("password_uppercase_required") {
		options.PasswordUppercaseRequired = gitlab.Ptr(d.Get("password_uppercase_required").(bool))
	}

	if d.HasChange("password_lowercase_required") {
		options.PasswordLowercaseRequired = gitlab.Ptr(d.Get("password_lowercase_required").(bool))
	}

	if d.HasChange("performance_bar_allowed_group_path") {
		options.PerformanceBarAllowedGroupPath = gitlab.Ptr(d.Get("performance_bar_allowed_group_path").(string))
	}

	if d.HasChange("personal_access_token_prefix") {
		options.PersonalAccessTokenPrefix = gitlab.Ptr(d.Get("personal_access_token_prefix").(string))
	}

	if d.HasChange("pipeline_limit_per_project_user_sha") {
		options.PipelineLimitPerProjectUserSha = gitlab.Ptr(int64(d.Get("pipeline_limit_per_project_user_sha").(int)))
	}

	if d.HasChange("plantuml_enabled") {
		options.PlantumlEnabled = gitlab.Ptr(d.Get("plantuml_enabled").(bool))
	}

	if d.HasChange("plantuml_url") {
		options.PlantumlURL = gitlab.Ptr(d.Get("plantuml_url").(string))
	}

	if d.HasChange("polling_interval_multiplier") {
		gv := d.Get("polling_interval_multiplier").(float64)
		options.PollingIntervalMultiplier = &gv
	}

	if d.HasChange("project_export_enabled") {
		options.ProjectExportEnabled = gitlab.Ptr(d.Get("project_export_enabled").(bool))
	}

	if d.HasChange("project_jobs_api_rate_limit") {
		options.ProjectJobsAPIRateLimit = gitlab.Ptr(int64(d.Get("project_jobs_api_rate_limit").(int)))
	}

	if d.HasChange("projects_api_rate_limit_unauthenticated") {
		options.ProjectsAPIRateLimitUnauthenticated = gitlab.Ptr(int64(d.Get("projects_api_rate_limit_unauthenticated").(int)))
	}

	if d.HasChange("prometheus_metrics_enabled") {
		options.PrometheusMetricsEnabled = gitlab.Ptr(d.Get("prometheus_metrics_enabled").(bool))
	}

	if d.HasChange("protected_ci_variables") {
		options.ProtectedCIVariables = gitlab.Ptr(d.Get("protected_ci_variables").(bool))
	}

	if d.HasChange("push_event_activities_limit") {
		options.PushEventActivitiesLimit = gitlab.Ptr(int64(d.Get("push_event_activities_limit").(int)))
	}

	if d.HasChange("push_event_hooks_limit") {
		options.PushEventHooksLimit = gitlab.Ptr(int64(d.Get("push_event_hooks_limit").(int)))
	}

	if d.HasChange("rate_limiting_response_text") {
		options.RateLimitingResponseText = gitlab.Ptr(d.Get("rate_limiting_response_text").(string))
	}

	if d.HasChange("raw_blob_request_limit") {
		options.RawBlobRequestLimit = gitlab.Ptr(int64(d.Get("raw_blob_request_limit").(int)))
	}

	if d.HasChange("search_rate_limit") {
		options.SearchRateLimit = gitlab.Ptr(int64(d.Get("search_rate_limit").(int)))
	}

	if d.HasChange("search_rate_limit_unauthenticated") {
		options.SearchRateLimitUnauthenticated = gitlab.Ptr(int64(d.Get("search_rate_limit_unauthenticated").(int)))
	}

	if d.HasChange("recaptcha_enabled") {
		options.RecaptchaEnabled = gitlab.Ptr(d.Get("recaptcha_enabled").(bool))
	}

	if d.HasChange("recaptcha_private_key") {
		options.RecaptchaPrivateKey = gitlab.Ptr(d.Get("recaptcha_private_key").(string))
	}

	if d.HasChange("recaptcha_site_key") {
		options.RecaptchaSiteKey = gitlab.Ptr(d.Get("recaptcha_site_key").(string))
	}

	if d.HasChange("receive_max_input_size") {
		options.ReceiveMaxInputSize = gitlab.Ptr(int64(d.Get("receive_max_input_size").(int)))
	}

	if d.HasChange("receptive_cluster_agents_enabled") {
		options.ReceptiveClusterAgentsEnabled = gitlab.Ptr(d.Get("receptive_cluster_agents_enabled").(bool))
	}

	if d.HasChange("remember_me_enabled") {
		options.RememberMeEnabled = gitlab.Ptr(d.Get("remember_me_enabled").(bool))
	}

	if d.HasChange("repository_checks_enabled") {
		options.RepositoryChecksEnabled = gitlab.Ptr(d.Get("repository_checks_enabled").(bool))
	}

	if d.HasChange("repository_size_limit") {
		options.RepositorySizeLimit = gitlab.Ptr(int64(d.Get("repository_size_limit").(int)))
	}

	if d.HasChange("repository_storages_weighted") {
		gv := fromIntegerMapToInt64(d.Get("repository_storages_weighted"))
		options.RepositoryStoragesWeighted = &gv
	}

	if d.HasChange("require_admin_approval_after_user_signup") {
		options.RequireAdminApprovalAfterUserSignup = gitlab.Ptr(d.Get("require_admin_approval_after_user_signup").(bool))
	}

	if d.HasChange("require_admin_two_factor_authentication") {
		options.RequireAdminTwoFactorAuthentication = gitlab.Ptr(d.Get("require_admin_two_factor_authentication").(bool))
	}

	if d.HasChange("require_personal_access_token_expiry") {
		options.RequirePersonalAccessTokenExpiry = gitlab.Ptr(d.Get("require_personal_access_token_expiry").(bool))
	}

	if d.HasChange("require_two_factor_authentication") {
		options.RequireTwoFactorAuthentication = gitlab.Ptr(d.Get("require_two_factor_authentication").(bool))
	}

	if d.HasChange("restricted_visibility_levels") {
		options.RestrictedVisibilityLevels = stringListToVisibilityLevelSlice(d.Get("restricted_visibility_levels").([]any))
	}

	if d.HasChange("rsa_key_restriction") {
		options.RSAKeyRestriction = gitlab.Ptr(int64(d.Get("rsa_key_restriction").(int)))
	}

	if d.HasChange("security_approval_policies_limit") {
		options.SecurityApprovalPoliciesLimit = gitlab.Ptr(int64(d.Get("security_approval_policies_limit").(int)))
	}

	if d.HasChange("security_policy_global_group_approvers_enabled") {
		options.SecurityPolicyGlobalGroupApproversEnabled = gitlab.Ptr(d.Get("security_policy_global_group_approvers_enabled").(bool))
	}

	if d.HasChange("security_txt_content") {
		options.SecurityTXTContent = gitlab.Ptr(d.Get("security_txt_content").(string))
	}

	if d.HasChange("send_user_confirmation_email") {
		options.SendUserConfirmationEmail = gitlab.Ptr(d.Get("send_user_confirmation_email").(bool))
	}

	if d.HasChange("service_access_tokens_expiration_enforced") {
		options.ServiceAccessTokensExpirationEnforced = gitlab.Ptr(d.Get("service_access_tokens_expiration_enforced").(bool))
	}

	if d.HasChange("session_expire_delay") {
		options.SessionExpireDelay = gitlab.Ptr(int64(d.Get("session_expire_delay").(int)))
	}

	if d.HasChange("shared_runners_enabled") {
		options.SharedRunnersEnabled = gitlab.Ptr(d.Get("shared_runners_enabled").(bool))
	}

	if d.HasChange("shared_runners_minutes") {
		options.SharedRunnersMinutes = gitlab.Ptr(int64(d.Get("shared_runners_minutes").(int)))
	}

	if d.HasChange("shared_runners_text") {
		options.SharedRunnersText = gitlab.Ptr(d.Get("shared_runners_text").(string))
	}

	if d.HasChange("sidekiq_job_limiter_mode") {
		options.SidekiqJobLimiterMode = gitlab.Ptr(d.Get("sidekiq_job_limiter_mode").(string))
	}

	if d.HasChange("sidekiq_job_limiter_compression_threshold_bytes") {
		options.SidekiqJobLimiterCompressionThresholdBytes = gitlab.Ptr(int64(d.Get("sidekiq_job_limiter_compression_threshold_bytes").(int)))
	}

	if d.HasChange("sidekiq_job_limiter_limit_bytes") {
		options.SidekiqJobLimiterLimitBytes = gitlab.Ptr(int64(d.Get("sidekiq_job_limiter_limit_bytes").(int)))
	}

	if d.HasChange("sign_in_text") {
		options.SignInText = gitlab.Ptr(d.Get("sign_in_text").(string))
	}

	if d.HasChange("signup_enabled") {
		options.SignupEnabled = gitlab.Ptr(d.Get("signup_enabled").(bool))
	}

	if d.HasChange("silent_admin_exports_enabled") {
		options.SilentAdminExportsEnabled = gitlab.Ptr(d.Get("silent_admin_exports_enabled").(bool))
	}

	if d.HasChange("silent_mode_enabled") {
		options.SilentModeEnabled = gitlab.Ptr(d.Get("silent_mode_enabled").(bool))
	}

	if d.HasChange("slack_app_enabled") {
		options.SlackAppEnabled = gitlab.Ptr(d.Get("slack_app_enabled").(bool))
	}

	if d.HasChange("slack_app_id") {
		options.SlackAppID = gitlab.Ptr(d.Get("slack_app_id").(string))
	}

	if d.HasChange("slack_app_secret") {
		options.SlackAppSecret = gitlab.Ptr(d.Get("slack_app_secret").(string))
	}

	if d.HasChange("slack_app_signing_secret") {
		options.SlackAppSigningSecret = gitlab.Ptr(d.Get("slack_app_signing_secret").(string))
	}

	if d.HasChange("slack_app_verification_token") {
		options.SlackAppVerificationToken = gitlab.Ptr(d.Get("slack_app_verification_token").(string))
	}

	if d.HasChange("snippet_size_limit") {
		options.SnippetSizeLimit = gitlab.Ptr(int64(d.Get("snippet_size_limit").(int)))
	}

	if d.HasChange("snowplow_app_id") {
		options.SnowplowAppID = gitlab.Ptr(d.Get("snowplow_app_id").(string))
	}

	if d.HasChange("snowplow_collector_hostname") {
		options.SnowplowCollectorHostname = gitlab.Ptr(d.Get("snowplow_collector_hostname").(string))
	}

	if d.HasChange("snowplow_cookie_domain") {
		options.SnowplowCookieDomain = gitlab.Ptr(d.Get("snowplow_cookie_domain").(string))
	}

	if d.HasChange("snowplow_database_collector_hostname") {
		options.SnowplowDatabaseCollectorHostname = gitlab.Ptr(d.Get("snowplow_database_collector_hostname").(string))
	}

	if d.HasChange("snowplow_enabled") {
		options.SnowplowEnabled = gitlab.Ptr(d.Get("snowplow_enabled").(bool))
	}

	if d.HasChange("sourcegraph_enabled") {
		options.SourcegraphEnabled = gitlab.Ptr(d.Get("sourcegraph_enabled").(bool))
	}

	if d.HasChange("sourcegraph_public_only") {
		options.SourcegraphPublicOnly = gitlab.Ptr(d.Get("sourcegraph_public_only").(bool))
	}

	if d.HasChange("sourcegraph_url") {
		options.SourcegraphURL = gitlab.Ptr(d.Get("sourcegraph_url").(string))
	}

	if d.HasChange("spam_check_endpoint_enabled") {
		options.SpamCheckEndpointEnabled = gitlab.Ptr(d.Get("spam_check_endpoint_enabled").(bool))
	}

	if d.HasChange("spam_check_endpoint_url") {
		options.SpamCheckEndpointURL = gitlab.Ptr(d.Get("spam_check_endpoint_url").(string))
	}

	if d.HasChange("spam_check_api_key") {
		options.SpamCheckAPIKey = gitlab.Ptr(d.Get("spam_check_api_key").(string))
	}

	if d.HasChange("static_objects_external_storage_auth_token") {
		options.StaticObjectsExternalStorageAuthToken = gitlab.Ptr(d.Get("static_objects_external_storage_auth_token").(string))
	}

	if d.HasChange("static_objects_external_storage_url") {
		options.StaticObjectsExternalStorageURL = gitlab.Ptr(d.Get("static_objects_external_storage_url").(string))
	}

	if d.HasChange("suggest_pipeline_enabled") {
		options.SuggestPipelineEnabled = gitlab.Ptr(d.Get("suggest_pipeline_enabled").(bool))
	}

	if d.HasChange("terminal_max_session_time") {
		options.TerminalMaxSessionTime = gitlab.Ptr(int64(d.Get("terminal_max_session_time").(int)))
	}

	if d.HasChange("terms") {
		options.Terms = gitlab.Ptr(d.Get("terms").(string))
	}

	if d.HasChange("throttle_authenticated_api_enabled") {
		options.ThrottleAuthenticatedAPIEnabled = gitlab.Ptr(d.Get("throttle_authenticated_api_enabled").(bool))
	}

	if d.HasChange("throttle_authenticated_api_period_in_seconds") {
		options.ThrottleAuthenticatedAPIPeriodInSeconds = gitlab.Ptr(int64(d.Get("throttle_authenticated_api_period_in_seconds").(int)))
	}

	if d.HasChange("throttle_authenticated_api_requests_per_period") {
		options.ThrottleAuthenticatedAPIRequestsPerPeriod = gitlab.Ptr(int64(d.Get("throttle_authenticated_api_requests_per_period").(int)))
	}

	if d.HasChange("throttle_authenticated_packages_api_enabled") {
		options.ThrottleAuthenticatedPackagesAPIEnabled = gitlab.Ptr(d.Get("throttle_authenticated_packages_api_enabled").(bool))
	}

	if d.HasChange("throttle_authenticated_packages_api_period_in_seconds") {
		options.ThrottleAuthenticatedPackagesAPIPeriodInSeconds = gitlab.Ptr(int64(d.Get("throttle_authenticated_packages_api_period_in_seconds").(int)))
	}

	if d.HasChange("throttle_authenticated_packages_api_requests_per_period") {
		options.ThrottleAuthenticatedPackagesAPIRequestsPerPeriod = gitlab.Ptr(int64(d.Get("throttle_authenticated_packages_api_requests_per_period").(int)))
	}

	if d.HasChange("throttle_authenticated_web_enabled") {
		options.ThrottleAuthenticatedWebEnabled = gitlab.Ptr(d.Get("throttle_authenticated_web_enabled").(bool))
	}

	if d.HasChange("throttle_authenticated_web_period_in_seconds") {
		options.ThrottleAuthenticatedWebPeriodInSeconds = gitlab.Ptr(int64(d.Get("throttle_authenticated_web_period_in_seconds").(int)))
	}

	if d.HasChange("throttle_authenticated_web_requests_per_period") {
		options.ThrottleAuthenticatedWebRequestsPerPeriod = gitlab.Ptr(int64(d.Get("throttle_authenticated_web_requests_per_period").(int)))
	}

	if d.HasChange("throttle_unauthenticated_api_enabled") {
		options.ThrottleUnauthenticatedAPIEnabled = gitlab.Ptr(d.Get("throttle_unauthenticated_api_enabled").(bool))
	}

	if d.HasChange("throttle_unauthenticated_api_period_in_seconds") {
		options.ThrottleUnauthenticatedAPIPeriodInSeconds = gitlab.Ptr(int64(d.Get("throttle_unauthenticated_api_period_in_seconds").(int)))
	}

	if d.HasChange("throttle_unauthenticated_api_requests_per_period") {
		options.ThrottleUnauthenticatedAPIRequestsPerPeriod = gitlab.Ptr(int64(d.Get("throttle_unauthenticated_api_requests_per_period").(int)))
	}

	if d.HasChange("throttle_unauthenticated_packages_api_enabled") {
		options.ThrottleUnauthenticatedPackagesAPIEnabled = gitlab.Ptr(d.Get("throttle_unauthenticated_packages_api_enabled").(bool))
	}

	if d.HasChange("throttle_unauthenticated_packages_api_period_in_seconds") {
		options.ThrottleUnauthenticatedPackagesAPIPeriodInSeconds = gitlab.Ptr(int64(d.Get("throttle_unauthenticated_packages_api_period_in_seconds").(int)))
	}

	if d.HasChange("throttle_unauthenticated_packages_api_requests_per_period") {
		options.ThrottleUnauthenticatedPackagesAPIRequestsPerPeriod = gitlab.Ptr(int64(d.Get("throttle_unauthenticated_packages_api_requests_per_period").(int)))
	}

	if d.HasChange("throttle_unauthenticated_web_enabled") {
		options.ThrottleUnauthenticatedWebEnabled = gitlab.Ptr(d.Get("throttle_unauthenticated_web_enabled").(bool))
	}

	if d.HasChange("throttle_unauthenticated_web_period_in_seconds") {
		options.ThrottleUnauthenticatedWebPeriodInSeconds = gitlab.Ptr(int64(d.Get("throttle_unauthenticated_web_period_in_seconds").(int)))
	}

	if d.HasChange("throttle_unauthenticated_web_requests_per_period") {
		options.ThrottleUnauthenticatedWebRequestsPerPeriod = gitlab.Ptr(int64(d.Get("throttle_unauthenticated_web_requests_per_period").(int)))
	}

	if d.HasChange("time_tracking_limit_to_hours") {
		options.TimeTrackingLimitToHours = gitlab.Ptr(d.Get("time_tracking_limit_to_hours").(bool))
	}

	if d.HasChange("two_factor_grace_period") {
		options.TwoFactorGracePeriod = gitlab.Ptr(int64(d.Get("two_factor_grace_period").(int)))
	}

	if d.HasChange("unconfirmed_users_delete_after_days") {
		options.UnconfirmedUsersDeleteAfterDays = gitlab.Ptr(int64(d.Get("unconfirmed_users_delete_after_days").(int)))
	}

	if d.HasChange("unique_ips_limit_enabled") {
		options.UniqueIPsLimitEnabled = gitlab.Ptr(d.Get("unique_ips_limit_enabled").(bool))
	}

	if d.HasChange("unique_ips_limit_per_user") {
		options.UniqueIPsLimitPerUser = gitlab.Ptr(int64(d.Get("unique_ips_limit_per_user").(int)))
	}

	if d.HasChange("unique_ips_limit_time_window") {
		options.UniqueIPsLimitTimeWindow = gitlab.Ptr(int64(d.Get("unique_ips_limit_time_window").(int)))
	}

	if d.HasChange("update_runner_versions_enabled") {
		options.UpdateRunnerVersionsEnabled = gitlab.Ptr(d.Get("update_runner_versions_enabled").(bool))
	}

	if d.HasChange("usage_ping_enabled") {
		options.UsagePingEnabled = gitlab.Ptr(d.Get("usage_ping_enabled").(bool))
	}

	if d.HasChange("use_clickhouse_for_analytics") {
		options.UseClickhouseForAnalytics = gitlab.Ptr(d.Get("use_clickhouse_for_analytics").(bool))
	}

	if d.HasChange("user_deactivation_emails_enabled") {
		options.UserDeactivationEmailsEnabled = gitlab.Ptr(d.Get("user_deactivation_emails_enabled").(bool))
	}

	if d.HasChange("user_default_external") {
		options.UserDefaultExternal = gitlab.Ptr(d.Get("user_default_external").(bool))
	}

	if d.HasChange("user_default_internal_regex") {
		options.UserDefaultInternalRegex = gitlab.Ptr(d.Get("user_default_internal_regex").(string))
	}

	if d.HasChange("user_defaults_to_private_profile") {
		options.UserDefaultsToPrivateProfile = gitlab.Ptr(d.Get("user_defaults_to_private_profile").(bool))
	}

	if d.HasChange("user_oauth_applications") {
		options.UserOauthApplications = gitlab.Ptr(d.Get("user_oauth_applications").(bool))
	}

	if d.HasChange("user_show_add_ssh_key_message") {
		options.UserShowAddSSHKeyMessage = gitlab.Ptr(d.Get("user_show_add_ssh_key_message").(bool))
	}

	if d.HasChange("valid_runner_registrars") {
		v := d.Get("valid_runner_registrars").([]any)
		registrars := make([]string, len(v))
		for i, reg := range v {
			registrars[i] = reg.(string)
		}
		options.ValidRunnerRegistrars = &registrars
	}

	if d.HasChange("version_check_enabled") {
		options.VersionCheckEnabled = gitlab.Ptr(d.Get("version_check_enabled").(bool))
	}

	if d.HasChange("whats_new_variant") {
		options.WhatsNewVariant = gitlab.Ptr(d.Get("whats_new_variant").(string))
	}

	if d.HasChange("web_ide_clientside_preview_enabled") {
		options.WebIDEClientsidePreviewEnabled = gitlab.Ptr(d.Get("web_ide_clientside_preview_enabled").(bool))
	}

	if d.HasChange("wiki_page_max_content_bytes") {
		options.WikiPageMaxContentBytes = gitlab.Ptr(int64(d.Get("wiki_page_max_content_bytes").(int)))
	}
	return &options
}
