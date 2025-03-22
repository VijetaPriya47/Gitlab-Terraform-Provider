package api

import (
	"encoding/json"
	"net/http"
	"time"

	"gitlab.com/gitlab-org/api/client-go"
)

type Settings struct {
	ID                                                    int                      `json:"id"`
	AbuseNotificationEmail                                string                   `json:"abuse_notification_email"`
	AdminMode                                             bool                     `json:"admin_mode"`
	AfterSignOutPath                                      string                   `json:"after_sign_out_path"`
	AfterSignUpText                                       string                   `json:"after_sign_up_text"`
	AkismetAPIKey                                         string                   `json:"akismet_api_key"`
	AkismetEnabled                                        bool                     `json:"akismet_enabled"`
	AllowAccountDeletion                                  bool                     `json:"allow_account_deletion"`
	AllowGroupOwnersToManageLDAP                          bool                     `json:"allow_group_owners_to_manage_ldap"`
	AllowLocalRequestsFromSystemHooks                     bool                     `json:"allow_local_requests_from_system_hooks"`
	AllowLocalRequestsFromWebHooksAndServices             bool                     `json:"allow_local_requests_from_web_hooks_and_services"`
	AllowProjectCreationForGuestAndBelow                  bool                     `json:"allow_project_creation_for_guest_and_below"`
	AllowRunnerRegistrationToken                          bool                     `json:"allow_runner_registration_token"`
	ArchiveBuildsInHumanReadable                          string                   `json:"archive_builds_in_human_readable"`
	AsciidocMaxIncludes                                   int                      `json:"asciidoc_max_includes"`
	AssetProxyAllowlist                                   []string                 `json:"asset_proxy_allowlist"`
	AssetProxyEnabled                                     bool                     `json:"asset_proxy_enabled"`
	AssetProxyURL                                         string                   `json:"asset_proxy_url"`
	AssetProxySecretKey                                   string                   `json:"asset_proxy_secret_key"`
	AuthorizedKeysEnabled                                 bool                     `json:"authorized_keys_enabled"`
	AutoBanUserOnExcessiveProjectsDownload                bool                     `json:"auto_ban_user_on_excessive_projects_download"`
	AutoDevOpsDomain                                      string                   `json:"auto_devops_domain"`
	AutoDevOpsEnabled                                     bool                     `json:"auto_devops_enabled"`
	AutomaticPurchasedStorageAllocation                   bool                     `json:"automatic_purchased_storage_allocation"`
	BulkImportConcurrentPipelineBatchLimit                int                      `json:"bulk_import_concurrent_pipeline_batch_limit"`
	BulkImportEnabled                                     bool                     `json:"bulk_import_enabled"`
	BulkImportMaxDownloadFileSize                         int                      `json:"bulk_import_max_download_file_size"`
	CanCreateGroup                                        bool                     `json:"can_create_group"`
	CheckNamespacePlan                                    bool                     `json:"check_namespace_plan"`
	CIMaxIncludes                                         int                      `json:"ci_max_includes"`
	CIMaxTotalYAMLSizeBytes                               int                      `json:"ci_max_total_yaml_size_bytes"`
	CommitEmailHostname                                   string                   `json:"commit_email_hostname"`
	ConcurrentBitbucketImportJobsLimit                    int                      `json:"concurrent_bitbucket_import_jobs_limit"`
	ConcurrentBitbucketServerImportJobsLimit              int                      `json:"concurrent_bitbucket_server_import_jobs_limit"`
	ConcurrentGithubImportJobsLimit                       int                      `json:"concurrent_github_import_jobs_limit"`
	ContainerExpirationPoliciesEnableHistoricEntries      bool                     `json:"container_expiration_policies_enable_historic_entries"`
	ContainerRegistryCleanupTagsServiceMaxListSize        int                      `json:"container_registry_cleanup_tags_service_max_list_size"`
	ContainerRegistryDeleteTagsServiceTimeout             int                      `json:"container_registry_delete_tags_service_timeout"`
	ContainerRegistryExpirationPoliciesCaching            bool                     `json:"container_registry_expiration_policies_caching"`
	ContainerRegistryExpirationPoliciesWorkerCapacity     int                      `json:"container_registry_expiration_policies_worker_capacity"`
	ContainerRegistryImportMaxRetries                     int                      `json:"container_registry_import_max_retries"`
	ContainerRegistryImportMaxStepDuration                int                      `json:"container_registry_import_max_step_duration"`
	ContainerRegistryImportMaxTagsCount                   int                      `json:"container_registry_import_max_tags_count"`
	ContainerRegistryImportStartMaxRetries                int                      `json:"container_registry_import_start_max_retries"`
	ContainerRegistryImportTargetPlan                     string                   `json:"container_registry_import_target_plan"`
	ContainerRegistryTokenExpireDelay                     int                      `json:"container_registry_token_expire_delay"`
	CreatedAt                                             *time.Time               `json:"created_at"`
	CustomHTTPCloneURLRoot                                string                   `json:"custom_http_clone_url_root"`
	DNSRebindingProtectionEnabled                         bool                     `json:"dns_rebinding_protection_enabled"`
	DSAKeyRestriction                                     int                      `json:"dsa_key_restriction"`
	DeactivateDormantUsers                                bool                     `json:"deactivate_dormant_users"`
	DeactivateDormantUsersPeriod                          int                      `json:"deactivate_dormant_users_period"`
	DecompressArchiveFileTimeout                          int                      `json:"decompress_archive_file_timeout"`
	DefaultArtifactsExpireIn                              string                   `json:"default_artifacts_expire_in"`
	DefaultBranchName                                     string                   `json:"default_branch_name"`
	DefaultBranchProtection                               int                      `json:"default_branch_protection"`
	DefaultCiConfigPath                                   string                   `json:"default_ci_config_path"`
	DefaultGroupVisibility                                gitlab.VisibilityValue   `json:"default_group_visibility"`
	DefaultPreferredLanguage                              string                   `json:"default_preferred_language"`
	DefaultProjectCreation                                int                      `json:"default_project_creation"`
	DefaultProjectDeletionProtection                      bool                     `json:"default_project_deletion_protection"`
	DefaultProjectVisibility                              gitlab.VisibilityValue   `json:"default_project_visibility"`
	DefaultProjectsLimit                                  int                      `json:"default_projects_limit"`
	DefaultSnippetVisibility                              gitlab.VisibilityValue   `json:"default_snippet_visibility"`
	DefaultSyntaxHighlightingTheme                        int                      `json:"default_syntax_highlighting_theme"`
	DelayedGroupDeletion                                  bool                     `json:"delayed_group_deletion"`
	DelayedProjectDeletion                                bool                     `json:"delayed_project_deletion"`
	DeleteInactiveProjects                                bool                     `json:"delete_inactive_projects"`
	DeleteUnconfirmedUsers                                bool                     `json:"delete_unconfirmed_users"`
	DeletionAdjournedPeriod                               int                      `json:"deletion_adjourned_period"`
	DiagramsnetEnabled                                    bool                     `json:"diagramsnet_enabled"`
	DiagramsnetURL                                        string                   `json:"diagramsnet_url"`
	DiffMaxFiles                                          int                      `json:"diff_max_files"`
	DiffMaxLines                                          int                      `json:"diff_max_lines"`
	DiffMaxPatchBytes                                     int                      `json:"diff_max_patch_bytes"`
	DisableAdminOauthScopes                               bool                     `json:"disable_admin_oauth_scopes"`
	DisableFeedToken                                      bool                     `json:"disable_feed_token"`
	DisableOverridingApproversPerMergeRequest             bool                     `json:"disable_overriding_approvers_per_merge_request"`
	DisablePersonalAccessTokens                           bool                     `json:"disable_personal_access_tokens"`
	DisabledOauthSignInSources                            []string                 `json:"disabled_oauth_sign_in_sources"`
	DomainAllowlist                                       []string                 `json:"domain_allowlist"`
	DomainDenylist                                        []string                 `json:"domain_denylist"`
	DomainDenylistEnabled                                 bool                     `json:"domain_denylist_enabled"`
	DownstreamPipelineTriggerLimitPerProjectUserSHA       int                      `json:"downstream_pipeline_trigger_limit_per_project_user_sha"`
	DuoFeaturesEnabled                                    bool                     `json:"duo_features_enabled"`
	ECDSAKeyRestriction                                   int                      `json:"ecdsa_key_restriction"`
	ECDSASKKeyRestriction                                 int                      `json:"ecdsa_sk_key_restriction"`
	EKSAccessKeyID                                        string                   `json:"eks_access_key_id"`
	EKSAccountID                                          string                   `json:"eks_account_id"`
	EKSIntegrationEnabled                                 bool                     `json:"eks_integration_enabled"`
	EKSSecretAccessKey                                    string                   `json:"eks_secret_access_key"`
	Ed25519KeyRestriction                                 int                      `json:"ed25519_key_restriction"`
	Ed25519SKKeyRestriction                               int                      `json:"ed25519_sk_key_restriction"`
	ElasticsearchAWS                                      bool                     `json:"elasticsearch_aws"`
	ElasticsearchAWSAccessKey                             string                   `json:"elasticsearch_aws_access_key"`
	ElasticsearchAWSRegion                                string                   `json:"elasticsearch_aws_region"`
	ElasticsearchAWSSecretAccessKey                       string                   `json:"elasticsearch_aws_secret_access_key"`
	ElasticsearchAnalyzersKuromojiEnabled                 bool                     `json:"elasticsearch_analyzers_kuromoji_enabled"`
	ElasticsearchAnalyzersKuromojiSearch                  bool                     `json:"elasticsearch_analyzers_kuromoji_search"`
	ElasticsearchAnalyzersSmartCNEnabled                  bool                     `json:"elasticsearch_analyzers_smartcn_enabled"`
	ElasticsearchAnalyzersSmartCNSearch                   bool                     `json:"elasticsearch_analyzers_smartcn_search"`
	ElasticsearchClientRequestTimeout                     int                      `json:"elasticsearch_client_request_timeout"`
	ElasticsearchIndexedFieldLengthLimit                  int                      `json:"elasticsearch_indexed_field_length_limit"`
	ElasticsearchIndexedFileSizeLimitKB                   int                      `json:"elasticsearch_indexed_file_size_limit_kb"`
	ElasticsearchIndexing                                 bool                     `json:"elasticsearch_indexing"`
	ElasticsearchLimitIndexing                            bool                     `json:"elasticsearch_limit_indexing"`
	ElasticsearchMaxBulkConcurrency                       int                      `json:"elasticsearch_max_bulk_concurrency"`
	ElasticsearchMaxBulkSizeMB                            int                      `json:"elasticsearch_max_bulk_size_mb"`
	ElasticsearchMaxCodeIndexingConcurrency               int                      `json:"elasticsearch_max_code_indexing_concurrency"`
	ElasticsearchNamespaceIDs                             []int                    `json:"elasticsearch_namespace_ids"`
	ElasticsearchPassword                                 string                   `json:"elasticsearch_password"`
	ElasticsearchPauseIndexing                            bool                     `json:"elasticsearch_pause_indexing"`
	ElasticsearchProjectIDs                               []int                    `json:"elasticsearch_project_ids"`
	ElasticsearchReplicas                                 int                      `json:"elasticsearch_replicas"`
	ElasticsearchRequeueWorkers                           bool                     `json:"elasticsearch_requeue_workers"`
	ElasticsearchSearch                                   bool                     `json:"elasticsearch_search"`
	ElasticsearchShards                                   int                      `json:"elasticsearch_shards"`
	ElasticsearchURL                                      []string                 `json:"elasticsearch_url"`
	ElasticsearchUsername                                 string                   `json:"elasticsearch_username"`
	ElasticsearchWorkerNumberOfShards                     int                      `json:"elasticsearch_worker_number_of_shards"`
	EmailAdditionalText                                   string                   `json:"email_additional_text"`
	EmailAuthorInBody                                     bool                     `json:"email_author_in_body"`
	EmailConfirmationSetting                              string                   `json:"email_confirmation_setting"`
	EmailRestrictions                                     string                   `json:"email_restrictions"`
	EmailRestrictionsEnabled                              bool                     `json:"email_restrictions_enabled"`
	EnableArtifactExternalRedirectWarningPage             bool                     `json:"enable_artifact_external_redirect_warning_page"`
	EnabledGitAccessProtocol                              string                   `json:"enabled_git_access_protocol"`
	EnforceNamespaceStorageLimit                          bool                     `json:"enforce_namespace_storage_limit"`
	EnforcePATExpiration                                  bool                     `json:"enforce_pat_expiration"`
	EnforceSSHKeyExpiration                               bool                     `json:"enforce_ssh_key_expiration"`
	EnforceTerms                                          bool                     `json:"enforce_terms"`
	ExternalAuthClientCert                                string                   `json:"external_auth_client_cert"`
	ExternalAuthClientKey                                 string                   `json:"external_auth_client_key"`
	ExternalAuthClientKeyPass                             string                   `json:"external_auth_client_key_pass"`
	ExternalAuthorizationServiceDefaultLabel              string                   `json:"external_authorization_service_default_label"`
	ExternalAuthorizationServiceEnabled                   bool                     `json:"external_authorization_service_enabled"`
	ExternalAuthorizationServiceTimeout                   float64                  `json:"external_authorization_service_timeout"`
	ExternalAuthorizationServiceURL                       string                   `json:"external_authorization_service_url"`
	ExternalPipelineValidationServiceTimeout              int                      `json:"external_pipeline_validation_service_timeout"`
	ExternalPipelineValidationServiceToken                string                   `json:"external_pipeline_validation_service_token"`
	ExternalPipelineValidationServiceURL                  string                   `json:"external_pipeline_validation_service_url"`
	FailedLoginAttemptsUnlockPeriodInMinutes              int                      `json:"failed_login_attempts_unlock_period_in_minutes"`
	FileTemplateProjectID                                 int                      `json:"file_template_project_id"`
	FirstDayOfWeek                                        int                      `json:"first_day_of_week"`
	FlocEnabled                                           bool                     `json:"floc_enabled"`
	GeoNodeAllowedIPs                                     string                   `json:"geo_node_allowed_ips"`
	GeoStatusTimeout                                      int                      `json:"geo_status_timeout"`
	GitRateLimitUsersAlertlist                            []int                    `json:"git_rate_limit_users_alertlist"`
	GitRateLimitUsersAllowlist                            []string                 `json:"git_rate_limit_users_allowlist"`
	GitTwoFactorSessionExpiry                             int                      `json:"git_two_factor_session_expiry"`
	GitalyTimeoutDefault                                  int                      `json:"gitaly_timeout_default"`
	GitalyTimeoutFast                                     int                      `json:"gitaly_timeout_fast"`
	GitalyTimeoutMedium                                   int                      `json:"gitaly_timeout_medium"`
	GitlabDedicatedInstance                               bool                     `json:"gitlab_dedicated_instance"`
	GitlabEnvironmentToolkitInstance                      bool                     `json:"gitlab_environment_toolkit_instance"`
	GitlabShellOperationLimit                             int                      `json:"gitlab_shell_operation_limit"`
	GitpodEnabled                                         bool                     `json:"gitpod_enabled"`
	GitpodURL                                             string                   `json:"gitpod_url"`
	GloballyAllowedIps                                    string                   `json:"globally_allowed_ips"`
	GrafanaEnabled                                        bool                     `json:"grafana_enabled"`
	GrafanaURL                                            string                   `json:"grafana_url"`
	GravatarEnabled                                       bool                     `json:"gravatar_enabled"`
	GroupDownloadExportLimit                              int                      `json:"group_download_export_limit"`
	GroupExportLimit                                      int                      `json:"group_export_limit"`
	GroupImportLimit                                      int                      `json:"group_import_limit"`
	GroupOwnersCanManageDefaultBranchProtection           bool                     `json:"group_owners_can_manage_default_branch_protection"`
	GroupRunnerTokenExpirationInterval                    int                      `json:"group_runner_token_expiration_interval"`
	HTMLEmailsEnabled                                     bool                     `json:"html_emails_enabled"`
	HashedStorageEnabled                                  bool                     `json:"hashed_storage_enabled"`
	HelpPageDocumentationBaseURL                          string                   `json:"help_page_documentation_base_url"`
	HelpPageHideCommercialContent                         bool                     `json:"help_page_hide_commercial_content"`
	HelpPageSupportURL                                    string                   `json:"help_page_support_url"`
	HelpPageText                                          string                   `json:"help_page_text"`
	HelpText                                              string                   `json:"help_text"`
	HideThirdPartyOffers                                  bool                     `json:"hide_third_party_offers"`
	HomePageURL                                           string                   `json:"home_page_url"`
	HousekeepingBitmapsEnabled                            bool                     `json:"housekeeping_bitmaps_enabled"`
	HousekeepingEnabled                                   bool                     `json:"housekeeping_enabled"`
	HousekeepingFullRepackPeriod                          int                      `json:"housekeeping_full_repack_period"`
	HousekeepingGcPeriod                                  int                      `json:"housekeeping_gc_period"`
	HousekeepingIncrementalRepackPeriod                   int                      `json:"housekeeping_incremental_repack_period"`
	HousekeepingOptimizeRepositoryPeriod                  int                      `json:"housekeeping_optimize_repository_period"`
	ImportSources                                         []string                 `json:"import_sources"`
	InactiveProjectsDeleteAfterMonths                     int                      `json:"inactive_projects_delete_after_months"`
	InactiveProjectsMinSizeMB                             int                      `json:"inactive_projects_min_size_mb"`
	InactiveProjectsSendWarningEmailAfterMonths           int                      `json:"inactive_projects_send_warning_email_after_months"`
	IncludeOptionalMetricsInServicePing                   bool                     `json:"include_optional_metrics_in_service_ping"`
	InProductMarketingEmailsEnabled                       bool                     `json:"in_product_marketing_emails_enabled"`
	InvisibleCaptchaEnabled                               bool                     `json:"invisible_captcha_enabled"`
	IssuesCreateLimit                                     int                      `json:"issues_create_limit"`
	JiraConnectApplicationKey                             string                   `json:"jira_connect_application_key"`
	JiraConnectProxyURL                                   string                   `json:"jira_connect_proxy_url"`
	JiraConnectPublicKeyStorageEnabled                    bool                     `json:"jira_connect_public_key_storage_enabled"`
	KeepLatestArtifact                                    bool                     `json:"keep_latest_artifact"`
	KrokiEnabled                                          bool                     `json:"kroki_enabled"`
	KrokiFormats                                          map[string]bool          `json:"kroki_formats"`
	KrokiURL                                              string                   `json:"kroki_url"`
	LocalMarkdownVersion                                  int                      `json:"local_markdown_version"`
	LockDuoFeaturesEnabled                                bool                     `json:"lock_duo_features_enabled"`
	LockMembershipsToLDAP                                 bool                     `json:"lock_memberships_to_ldap"`
	LoginRecaptchaProtectionEnabled                       bool                     `json:"login_recaptcha_protection_enabled"`
	MailgunEventsEnabled                                  bool                     `json:"mailgun_events_enabled"`
	MailgunSigningKey                                     string                   `json:"mailgun_signing_key"`
	MaintenanceMode                                       bool                     `json:"maintenance_mode"`
	MaintenanceModeMessage                                string                   `json:"maintenance_mode_message"`
	MavenPackageRequestsForwarding                        bool                     `json:"maven_package_requests_forwarding"`
	MaxArtifactsSize                                      int                      `json:"max_artifacts_size"`
	MaxAttachmentSize                                     int                      `json:"max_attachment_size"`
	MaxDecompressedArchiveSize                            int                      `json:"max_decompressed_archive_size"`
	MaxExportSize                                         int                      `json:"max_export_size"`
	MaxImportRemoteFileSize                               int                      `json:"max_import_remote_file_size"`
	MaxImportSize                                         int                      `json:"max_import_size"`
	MaxLoginAttempts                                      int                      `json:"max_login_attempts"`
	MaxNumberOfRepositoryDownloads                        int                      `json:"max_number_of_repository_downloads"`
	MaxNumberOfRepositoryDownloadsWithinTimePeriod        int                      `json:"max_number_of_repository_downloads_within_time_period"`
	MaxPagesSize                                          int                      `json:"max_pages_size"`
	MaxPersonalAccessTokenLifetime                        int                      `json:"max_personal_access_token_lifetime"`
	MaxSSHKeyLifetime                                     int                      `json:"max_ssh_key_lifetime"`
	MaxTerraformStateSizeBytes                            int                      `json:"max_terraform_state_size_bytes"`
	MaxYAMLDepth                                          int                      `json:"max_yaml_depth"`
	MaxYAMLSizeBytes                                      int                      `json:"max_yaml_size_bytes"`
	MetricsMethodCallThreshold                            int                      `json:"metrics_method_call_threshold"`
	MinimumPasswordLength                                 int                      `json:"minimum_password_length"`
	MirrorAvailable                                       bool                     `json:"mirror_available"`
	MirrorCapacityThreshold                               int                      `json:"mirror_capacity_threshold"`
	MirrorMaxCapacity                                     int                      `json:"mirror_max_capacity"`
	MirrorMaxDelay                                        int                      `json:"mirror_max_delay"`
	NPMPackageRequestsForwarding                          bool                     `json:"npm_package_requests_forwarding"`
	NotesCreateLimit                                      int                      `json:"notes_create_limit"`
	NotifyOnUnknownSignIn                                 bool                     `json:"notify_on_unknown_sign_in"`
	NugetSkipMetadataURLValidation                        bool                     `json:"nuget_skip_metadata_url_validation"`
	OutboundLocalRequestsAllowlistRaw                     string                   `json:"outbound_local_requests_allowlist_raw"`
	OutboundLocalRequestsWhitelist                        []string                 `json:"outbound_local_requests_whitelist"`
	PackageMetadataPurlTypes                              []int                    `json:"package_metadata_purl_types"`
	PackageRegistryAllowAnyoneToPullOption                bool                     `json:"package_registry_allow_anyone_to_pull_option"`
	PackageRegistryCleanupPoliciesWorkerCapacity          int                      `json:"package_registry_cleanup_policies_worker_capacity"`
	PagesDomainVerificationEnabled                        bool                     `json:"pages_domain_verification_enabled"`
	PasswordAuthenticationEnabledForGit                   bool                     `json:"password_authentication_enabled_for_git"`
	PasswordAuthenticationEnabledForWeb                   bool                     `json:"password_authentication_enabled_for_web"`
	PasswordNumberRequired                                bool                     `json:"password_number_required"`
	PasswordSymbolRequired                                bool                     `json:"password_symbol_required"`
	PasswordUppercaseRequired                             bool                     `json:"password_uppercase_required"`
	PasswordLowercaseRequired                             bool                     `json:"password_lowercase_required"`
	PerformanceBarAllowedGroupID                          int                      `json:"performance_bar_allowed_group_id"`
	PerformanceBarAllowedGroupPath                        string                   `json:"performance_bar_allowed_group_path"`
	PerformanceBarEnabled                                 bool                     `json:"performance_bar_enabled"`
	PersonalAccessTokenPrefix                             string                   `json:"personal_access_token_prefix"`
	PipelineLimitPerProjectUserSha                        int                      `json:"pipeline_limit_per_project_user_sha"`
	PlantumlEnabled                                       bool                     `json:"plantuml_enabled"`
	PlantumlURL                                           string                   `json:"plantuml_url"`
	PollingIntervalMultiplier                             float64                  `json:"polling_interval_multiplier,string"`
	PreventMergeRequestsAuthorApproval                    bool                     `json:"prevent_merge_request_author_approval"`
	PreventMergeRequestsCommittersApproval                bool                     `json:"prevent_merge_request_committers_approval"`
	ProjectDownloadExportLimit                            int                      `json:"project_download_export_limit"`
	ProjectExportEnabled                                  bool                     `json:"project_export_enabled"`
	ProjectExportLimit                                    int                      `json:"project_export_limit"`
	ProjectImportLimit                                    int                      `json:"project_import_limit"`
	ProjectJobsAPIRateLimit                               int                      `json:"project_jobs_api_rate_limit"`
	ProjectRunnerTokenExpirationInterval                  int                      `json:"project_runner_token_expiration_interval"`
	ProjectsAPIRateLimitUnauthenticated                   int                      `json:"projects_api_rate_limit_unauthenticated"`
	PrometheusMetricsEnabled                              bool                     `json:"prometheus_metrics_enabled"`
	ProtectedCIVariables                                  bool                     `json:"protected_ci_variables"`
	PseudonymizerEnabled                                  bool                     `json:"pseudonymizer_enabled"`
	PushEventActivitiesLimit                              int                      `json:"push_event_activities_limit"`
	PushEventHooksLimit                                   int                      `json:"push_event_hooks_limit"`
	PyPIPackageRequestsForwarding                         bool                     `json:"pypi_package_requests_forwarding"`
	RSAKeyRestriction                                     int                      `json:"rsa_key_restriction"`
	RateLimitingResponseText                              string                   `json:"rate_limiting_response_text"`
	RawBlobRequestLimit                                   int                      `json:"raw_blob_request_limit"`
	RecaptchaEnabled                                      bool                     `json:"recaptcha_enabled"`
	RecaptchaPrivateKey                                   string                   `json:"recaptcha_private_key"`
	RecaptchaSiteKey                                      string                   `json:"recaptcha_site_key"`
	ReceiveMaxInputSize                                   int                      `json:"receive_max_input_size"`
	ReceptiveClusterAgentsEnabled                         bool                     `json:"receptive_cluster_agents_enabled"`
	RememberMeEnabled                                     bool                     `json:"remember_me_enabled"`
	RepositoryChecksEnabled                               bool                     `json:"repository_checks_enabled"`
	RepositorySizeLimit                                   int                      `json:"repository_size_limit"`
	RepositoryStorages                                    []string                 `json:"repository_storages"`
	RepositoryStoragesWeighted                            map[string]int           `json:"repository_storages_weighted"`
	RequireAdminApprovalAfterUserSignup                   bool                     `json:"require_admin_approval_after_user_signup"`
	RequireAdminTwoFactorAuthentication                   bool                     `json:"require_admin_two_factor_authentication"`
	RequirePersonalAccessTokenExpiry                      bool                     `json:"require_personal_access_token_expiry"`
	RequireTwoFactorAuthentication                        bool                     `json:"require_two_factor_authentication"`
	RestrictedVisibilityLevels                            []gitlab.VisibilityValue `json:"restricted_visibility_levels"`
	RunnerTokenExpirationInterval                         int                      `json:"runner_token_expiration_interval"`
	SearchRateLimit                                       int                      `json:"search_rate_limit"`
	SearchRateLimitUnauthenticated                        int                      `json:"search_rate_limit_unauthenticated"`
	SecretDetectionRevocationTokenTypesURL                string                   `json:"secret_detection_revocation_token_types_url"`
	SecretDetectionTokenRevocationEnabled                 bool                     `json:"secret_detection_token_revocation_enabled"`
	SecretDetectionTokenRevocationToken                   string                   `json:"secret_detection_token_revocation_token"`
	SecretDetectionTokenRevocationURL                     string                   `json:"secret_detection_token_revocation_url"`
	SecurityApprovalPoliciesLimit                         int                      `json:"security_approval_policies_limit"`
	SecurityPolicyGlobalGroupApproversEnabled             bool                     `json:"security_policy_global_group_approvers_enabled"`
	SecurityTXTContent                                    string                   `json:"security_txt_content"`
	SendUserConfirmationEmail                             bool                     `json:"send_user_confirmation_email"`
	SentryClientsideDSN                                   string                   `json:"sentry_clientside_dsn"`
	SentryDSN                                             string                   `json:"sentry_dsn"`
	SentryEnabled                                         bool                     `json:"sentry_enabled"`
	SentryEnvironment                                     string                   `json:"sentry_environment"`
	ServiceAccessTokensExpirationEnforced                 bool                     `json:"service_access_tokens_expiration_enforced"`
	SessionExpireDelay                                    int                      `json:"session_expire_delay"`
	SharedRunnersEnabled                                  bool                     `json:"shared_runners_enabled"`
	SharedRunnersMinutes                                  int                      `json:"shared_runners_minutes"`
	SharedRunnersText                                     string                   `json:"shared_runners_text"`
	SidekiqJobLimiterCompressionThresholdBytes            int                      `json:"sidekiq_job_limiter_compression_threshold_bytes"`
	SidekiqJobLimiterLimitBytes                           int                      `json:"sidekiq_job_limiter_limit_bytes"`
	SidekiqJobLimiterMode                                 string                   `json:"sidekiq_job_limiter_mode"`
	SignInText                                            string                   `json:"sign_in_text"`
	SignupEnabled                                         bool                     `json:"signup_enabled"`
	SilentAdminExportsEnabled                             bool                     `json:"silent_admin_exports_enabled"`
	SilentModeEnabled                                     bool                     `json:"silent_mode_enabled"`
	SlackAppEnabled                                       bool                     `json:"slack_app_enabled"`
	SlackAppID                                            string                   `json:"slack_app_id"`
	SlackAppSecret                                        string                   `json:"slack_app_secret"`
	SlackAppSigningSecret                                 string                   `json:"slack_app_signing_secret"`
	SlackAppVerificationToken                             string                   `json:"slack_app_verification_token"`
	SnippetSizeLimit                                      int                      `json:"snippet_size_limit"`
	SnowplowAppID                                         string                   `json:"snowplow_app_id"`
	SnowplowCollectorHostname                             string                   `json:"snowplow_collector_hostname"`
	SnowplowCookieDomain                                  string                   `json:"snowplow_cookie_domain"`
	SnowplowDatabaseCollectorHostname                     string                   `json:"snowplow_database_collector_hostname"`
	SnowplowEnabled                                       bool                     `json:"snowplow_enabled"`
	SourcegraphEnabled                                    bool                     `json:"sourcegraph_enabled"`
	SourcegraphPublicOnly                                 bool                     `json:"sourcegraph_public_only"`
	SourcegraphURL                                        string                   `json:"sourcegraph_url"`
	SpamCheckAPIKey                                       string                   `json:"spam_check_api_key"`
	SpamCheckEndpointEnabled                              bool                     `json:"spam_check_endpoint_enabled"`
	SpamCheckEndpointURL                                  string                   `json:"spam_check_endpoint_url"`
	StaticObjectsExternalStorageAuthToken                 string                   `json:"static_objects_external_storage_auth_token"`
	StaticObjectsExternalStorageURL                       string                   `json:"static_objects_external_storage_url"`
	SuggestPipelineEnabled                                bool                     `json:"suggest_pipeline_enabled"`
	TerminalMaxSessionTime                                int                      `json:"terminal_max_session_time"`
	Terms                                                 string                   `json:"terms"`
	ThrottleAuthenticatedAPIEnabled                       bool                     `json:"throttle_authenticated_api_enabled"`
	ThrottleAuthenticatedAPIPeriodInSeconds               int                      `json:"throttle_authenticated_api_period_in_seconds"`
	ThrottleAuthenticatedAPIRequestsPerPeriod             int                      `json:"throttle_authenticated_api_requests_per_period"`
	ThrottleAuthenticatedDeprecatedAPIEnabled             bool                     `json:"throttle_authenticated_deprecated_api_enabled"`
	ThrottleAuthenticatedDeprecatedAPIPeriodInSeconds     int                      `json:"throttle_authenticated_deprecated_api_period_in_seconds"`
	ThrottleAuthenticatedDeprecatedAPIRequestsPerPeriod   int                      `json:"throttle_authenticated_deprecated_api_requests_per_period"`
	ThrottleAuthenticatedFilesAPIEnabled                  bool                     `json:"throttle_authenticated_files_api_enabled"`
	ThrottleAuthenticatedFilesAPIPeriodInSeconds          int                      `json:"throttle_authenticated_files_api_period_in_seconds"`
	ThrottleAuthenticatedFilesAPIRequestsPerPeriod        int                      `json:"throttle_authenticated_files_api_requests_per_period"`
	ThrottleAuthenticatedGitLFSEnabled                    bool                     `json:"throttle_authenticated_git_lfs_enabled"`
	ThrottleAuthenticatedGitLFSPeriodInSeconds            int                      `json:"throttle_authenticated_git_lfs_period_in_seconds"`
	ThrottleAuthenticatedGitLFSRequestsPerPeriod          int                      `json:"throttle_authenticated_git_lfs_requests_per_period"`
	ThrottleAuthenticatedPackagesAPIEnabled               bool                     `json:"throttle_authenticated_packages_api_enabled"`
	ThrottleAuthenticatedPackagesAPIPeriodInSeconds       int                      `json:"throttle_authenticated_packages_api_period_in_seconds"`
	ThrottleAuthenticatedPackagesAPIRequestsPerPeriod     int                      `json:"throttle_authenticated_packages_api_requests_per_period"`
	ThrottleAuthenticatedWebEnabled                       bool                     `json:"throttle_authenticated_web_enabled"`
	ThrottleAuthenticatedWebPeriodInSeconds               int                      `json:"throttle_authenticated_web_period_in_seconds"`
	ThrottleAuthenticatedWebRequestsPerPeriod             int                      `json:"throttle_authenticated_web_requests_per_period"`
	ThrottleIncidentManagementNotificationEnabled         bool                     `json:"throttle_incident_management_notification_enabled"`
	ThrottleIncidentManagementNotificationPerPeriod       int                      `json:"throttle_incident_management_notification_per_period"`
	ThrottleIncidentManagementNotificationPeriodInSeconds int                      `json:"throttle_incident_management_notification_period_in_seconds"`
	ThrottleProtectedPathsEnabled                         bool                     `json:"throttle_protected_paths_enabled"`
	ThrottleProtectedPathsPeriodInSeconds                 int                      `json:"throttle_protected_paths_period_in_seconds"`
	ThrottleProtectedPathsRequestsPerPeriod               int                      `json:"throttle_protected_paths_requests_per_period"`
	ThrottleUnauthenticatedAPIEnabled                     bool                     `json:"throttle_unauthenticated_api_enabled"`
	ThrottleUnauthenticatedAPIPeriodInSeconds             int                      `json:"throttle_unauthenticated_api_period_in_seconds"`
	ThrottleUnauthenticatedAPIRequestsPerPeriod           int                      `json:"throttle_unauthenticated_api_requests_per_period"`
	ThrottleUnauthenticatedDeprecatedAPIEnabled           bool                     `json:"throttle_unauthenticated_deprecated_api_enabled"`
	ThrottleUnauthenticatedDeprecatedAPIPeriodInSeconds   int                      `json:"throttle_unauthenticated_deprecated_api_period_in_seconds"`
	ThrottleUnauthenticatedDeprecatedAPIRequestsPerPeriod int                      `json:"throttle_unauthenticated_deprecated_api_requests_per_period"`
	ThrottleUnauthenticatedFilesAPIEnabled                bool                     `json:"throttle_unauthenticated_files_api_enabled"`
	ThrottleUnauthenticatedFilesAPIPeriodInSeconds        int                      `json:"throttle_unauthenticated_files_api_period_in_seconds"`
	ThrottleUnauthenticatedFilesAPIRequestsPerPeriod      int                      `json:"throttle_unauthenticated_files_api_requests_per_period"`
	ThrottleUnauthenticatedGitLFSEnabled                  bool                     `json:"throttle_unauthenticated_git_lfs_enabled"`
	ThrottleUnauthenticatedGitLFSPeriodInSeconds          int                      `json:"throttle_unauthenticated_git_lfs_period_in_seconds"`
	ThrottleUnauthenticatedGitLFSRequestsPerPeriod        int                      `json:"throttle_unauthenticated_git_lfs_requests_per_period"`
	ThrottleUnauthenticatedPackagesAPIEnabled             bool                     `json:"throttle_unauthenticated_packages_api_enabled"`
	ThrottleUnauthenticatedPackagesAPIPeriodInSeconds     int                      `json:"throttle_unauthenticated_packages_api_period_in_seconds"`
	ThrottleUnauthenticatedPackagesAPIRequestsPerPeriod   int                      `json:"throttle_unauthenticated_packages_api_requests_per_period"`
	ThrottleUnauthenticatedWebEnabled                     bool                     `json:"throttle_unauthenticated_web_enabled"`
	ThrottleUnauthenticatedWebPeriodInSeconds             int                      `json:"throttle_unauthenticated_web_period_in_seconds"`
	ThrottleUnauthenticatedWebRequestsPerPeriod           int                      `json:"throttle_unauthenticated_web_requests_per_period"`
	TimeTrackingLimitToHours                              bool                     `json:"time_tracking_limit_to_hours"`
	TwoFactorGracePeriod                                  int                      `json:"two_factor_grace_period"`
	UnconfirmedUsersDeleteAfterDays                       int                      `json:"unconfirmed_users_delete_after_days"`
	UniqueIPsLimitEnabled                                 bool                     `json:"unique_ips_limit_enabled"`
	UniqueIPsLimitPerUser                                 int                      `json:"unique_ips_limit_per_user"`
	UniqueIPsLimitTimeWindow                              int                      `json:"unique_ips_limit_time_window"`
	UpdateRunnerVersionsEnabled                           bool                     `json:"update_runner_versions_enabled"`
	UpdatedAt                                             *time.Time               `json:"updated_at"`
	UpdatingNameDisabledForUsers                          bool                     `json:"updating_name_disabled_for_users"`
	UsagePingEnabled                                      bool                     `json:"usage_ping_enabled"`
	UsagePingFeaturesEnabled                              bool                     `json:"usage_ping_features_enabled"`
	UseClickhouseForAnalytics                             bool                     `json:"use_clickhouse_for_analytics"`
	UserDeactivationEmailsEnabled                         bool                     `json:"user_deactivation_emails_enabled"`
	UserDefaultExternal                                   bool                     `json:"user_default_external"`
	UserDefaultInternalRegex                              string                   `json:"user_default_internal_regex"`
	UserDefaultsToPrivateProfile                          bool                     `json:"user_defaults_to_private_profile"`
	UserOauthApplications                                 bool                     `json:"user_oauth_applications"`
	UserShowAddSSHKeyMessage                              bool                     `json:"user_show_add_ssh_key_message"`
	UsersGetByIDLimit                                     int                      `json:"users_get_by_id_limit"`
	UsersGetByIDLimitAllowlistRaw                         string                   `json:"users_get_by_id_limit_allowlist_raw"`
	ValidRunnerRegistrars                                 []string                 `json:"valid_runner_registrars"`
	VersionCheckEnabled                                   bool                     `json:"version_check_enabled"`
	WebIDEClientsidePreviewEnabled                        bool                     `json:"web_ide_clientside_preview_enabled"`
	WhatsNewVariant                                       string                   `json:"whats_new_variant"`
	WikiPageMaxContentBytes                               int                      `json:"wiki_page_max_content_bytes"`

	// This uses a custom struct; see below
	DefaultBranchProtectionDefaults DefaultBranchProtectionDefaultsStruct `json:"default_branch_protection_defaults"`
}

// There is no client-go struct for this setup, only a struct for the create/update options, which isn't quite
// what's needed. Instead, this matches the structure for the group options, defined in groups:line 49
type DefaultBranchProtectionDefaultsStruct struct {
	AllowedToPush           []*gitlab.GroupAccessLevel `json:"allowed_to_push"`
	AllowForcePush          bool                       `json:"allow_force_push"`
	AllowedToMerge          []*gitlab.GroupAccessLevel `json:"allowed_to_merge"`
	DeveloperCanInitialPush bool                       `json:"developer_can_initial_push"`
}

// This is a temporary replacement for the normal gitlab client function for `GetSettings` that removes the container import settings
// and several more deprecated settings
func GetSettings(client *gitlab.Client, options ...gitlab.RequestOptionFunc) (*Settings, *gitlab.Response, error) {
	req, err := client.NewRequest(http.MethodGet, "application/settings", nil, options)
	if err != nil {
		return nil, nil, err
	}

	as := new(Settings)
	resp, err := client.Do(req, as)
	if err != nil {
		return nil, resp, err
	}

	return as, resp, nil
}

// This is a temporary replacement for the normal gitlab client function for `UpdateSettings` that removes the container import settings
// and several more deprecated settings. This is required because even though the Update wouldn't have the offending fields, the returned
// response from the udpate attempts to parse the same Settings struct, resulting in an error.
func UpdateSettings(client *gitlab.Client, opt *gitlab.UpdateSettingsOptions, options ...gitlab.RequestOptionFunc) (*Settings, *gitlab.Response, error) {
	req, err := client.NewRequest(http.MethodPut, "application/settings", opt, options)
	if err != nil {
		return nil, nil, err
	}

	as := new(Settings)
	resp, err := client.Do(req, as)
	if err != nil {
		return nil, resp, err
	}

	return as, resp, nil
}

// Settings requires a custom unmarshaller because time.Time attribute may sometimes return ""  which is not parseable as a date.
// This causes an error any time GetSettings is called on an instance with the registry disabled.
// To prevent this error, Settings has a custom UnmarshalJson that nils out the empty string value prior to parsing.
func (s *Settings) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" || string(data) == `""` {
		return nil
	}

	// Parse into an interface so we can check the value of the container registry to see if it's empty
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// If empty string, remove the value to leave it nil in the response, so it can parse to time properly.
	if m["container_registry_import_created_before"] == "" {
		delete(m, "container_registry_import_created_before")
	}
	if m["updated_at"] == "" {
		delete(m, "updated_at")
	}
	if m["created_at"] == "" {
		delete(m, "created_at")
	}

	// Write the interface to string so we can re-marshal to settings
	jsonString, err := json.Marshal(m)
	if err != nil {
		return err
	}

	// use an intermediate type to prevent infinite recursion
	type MarshalSettings Settings
	intermediate := MarshalSettings(*s)
	if err := json.Unmarshal(jsonString, &intermediate); err != nil {
		return err
	}
	*s = (Settings)(intermediate)
	return nil
}
