# internal/provider/utils Package

This package contains shared utility functions and types used across resource implementations.

## Token Rotation Architecture

The token rotation logic is centralized in `token_rotation.go` to avoid duplication across the 4 token resource types (Project, Group, Personal, Service Account).

### Key Types

#### `RotatableToken` Interface

All token resource models implement this interface to participate in the centralized rotation logic:

```go
type RotatableToken interface {
    GetExpiresAt() types.String
    GetExpirationDays() types.Int64
    GetRotateBeforeDays() types.Int64
    HasRotationConfiguration() bool

    SetExpiresAt(types.String)
    SetID(types.String)
    SetToken(types.String)
    SetCreatedAt(types.String)
}
```

### Key Functions

#### `ShouldRotateToken(ctx, planToken, stateToken, logPrefix)`

Determines if a token needs rotation. `stateToken` may be `nil` (e.g. during Create). Internally calls `api.CurrentTime()` — no `time.Time` argument is required from callers.

Rotation is triggered if:
1. `stateToken` is `nil` (new token being created)
2. `expires_at` changed between state and plan
3. `expiration_days` or `rotate_before_days` changed in the rotation configuration
4. Current time is on or after `expires_at - rotate_before_days`

#### `ApplyTokenRotation(ctx, token, rotationConfig, stateData, logPrefix, fallbackExpirationDays)`

Calculates the new expiration date and marks related plan attributes (`id`, `token`, `created_at`) as unknown to trigger re-creation.

#### `RotationConfigurationChanged(stateExpirationDays, planExpirationDays, stateRotateBeforeDays, planRotateBeforeDays)`

Returns `true` if the rotation configuration fields changed between state and plan.

### Time Mocking in Tests

`api.CurrentTime()` reads the `GITLAB_TESTING_TIME` environment variable (ISO 8601 date string) when set, allowing tests to control time without passing it as an argument:

```go
t.Setenv("GITLAB_TESTING_TIME", "2024-12-15")
```

## Date Validation

`date_validator.go` provides a custom Terraform attribute validator that ensures date strings conform to the ISO 8601 (`YYYY-MM-DD`) format expected by the GitLab API.
