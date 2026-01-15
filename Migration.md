# Migrating a datasource or resource from SDK to Terraform Plugin Framework

This guide is designed to help Duo migrate datasources and resources from the SDK to the Framework Plugin.

It provides specific instructions based off feedback from prior migration attempts.

## File locations

Datasources and resources being migrated live in `internal/provider/sdk`.
After migration, they should live in `internal/provider`.
Delete any files from `internal/provider/sdk` that have been replaced due to the migration.
Datasources in the `internal/provider/sdk` folder have filename prefix `data_source_`.
Migrated datasources in the `internal/provider` have filename prefix `datasource_`.

## Configure Function

All Framework datasources and resources have a `Configure` function.

The contents of the function for datasources is always:

```go
if req.ProviderData == nil {
    return
}

datasource := req.ProviderData.(*GitLabDatasourceData)
d.client = datasource.Client
```

The contents of the function for resources is always:

```go
if req.ProviderData == nil {
    return
}

resourceData := req.ProviderData.(*GitLabResourceData)
r.client = resourceData.Client
```

Important - the `r.client` field is NOT a secret and does not require obfuscating. Do not set it to asterisks! Set it to `resourceData.Client` as shown above.

## Add migration acceptance test

All resource migrations need an additional migration acceptance test.
Add this to the same test file as the other acceptance tests for the resource.

See file `internal/provider/resource_gitlab_cluster_agent_test.go` for an example called `TestAccGitlabClusterAgent_migrateFromSDKToFramework`.

There are three standard steps to the test:

1. Create the resource with a hardcoded version of the provider using the `ExternalProviders` attribute. This should be the latest released version of the provider, which you can get from the `CHANGELOG.md` file.
2. With the same terraform resource configuration, run against the version being tested.
3. Test the resource can be imported.

## Setting the Resource ID

The resource ID value should only be set in the `Create` function.
Do not set it in the `Read` or `Update` functions.

## Resources that always recreate on change

If all attributes cause the resource to recreate on update, then there is no need for logic in the `Update` function.
The function needs to exist to fulfil the interface.
The contents of the function should error as it should not be called legitimately.

For example:

```go
resp.Diagnostics.AddError(
    "Provider Error, report upstream",
    "Somehow the resource was requested to perform an in-place upgrade which is not possible.",
)
```
