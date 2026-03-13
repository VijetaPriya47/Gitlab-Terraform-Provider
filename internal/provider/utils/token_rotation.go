package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// RotatableToken is an interface for tokens that support automatic rotation.
// Implementations provide the necessary information to determine if a token
// needs to be rotated based on its expiration date and rotation configuration,
// and allow the rotation logic to update the plan.
type RotatableToken interface {
	// Getters - read current state
	GetExpiresAt() types.String
	GetExpirationDays() types.Int64
	GetRotateBeforeDays() types.Int64
	HasRotationConfiguration() bool
	GetLogPrefix() string

	// Setters - update the plan
	SetExpiresAt(types.String)
	SetID(types.String)
	SetToken(types.String)
	SetCreatedAt(types.String)
}

// RotationConfigurationChanged returns true if the rotation configuration has changed
// between state and plan, which should trigger immediate rotation.
func RotationConfigurationChanged(stateToken, planToken RotatableToken) bool {
	stateExpDays := stateToken.GetExpirationDays()
	planExpDays := planToken.GetExpirationDays()
	if !stateExpDays.IsNull() && !planExpDays.IsNull() && stateExpDays.ValueInt64() != planExpDays.ValueInt64() {
		return true
	}

	stateRotBefore := stateToken.GetRotateBeforeDays()
	planRotBefore := planToken.GetRotateBeforeDays()
	if !stateRotBefore.IsNull() && !planRotBefore.IsNull() && stateRotBefore.ValueInt64() != planRotBefore.ValueInt64() {
		return true
	}

	return false
}

// ShouldRotateToken determines if a token needs rotation and updates the plan if necessary.
// It handles:
// 1. First-time token creation
// 2. Configuration changes (expiration_days or rotate_before_days)
// 3. Time-based rotation based on rotate_before_days threshold
//
// stateToken may be nil to indicate that no prior state exists (e.g. Create).
// Returns true if the token should be rotated, false otherwise.
func ShouldRotateToken(
	ctx context.Context,
	planToken RotatableToken,
	stateToken RotatableToken,
) (bool, error) {
	logPrefix := planToken.GetLogPrefix()
	now := api.CurrentTime()

	// Derive stateExpiresAt (nil-safe)
	var stateExpiresAt types.String
	if stateToken != nil {
		stateExpiresAt = stateToken.GetExpiresAt()
	} else {
		stateExpiresAt = types.StringNull()
	}

	planExpiresAt := planToken.GetExpiresAt()

	// Special case: if both are null, no rotation needed (token with no expiry stays with no expiry)
	if stateExpiresAt.IsNull() && planExpiresAt.IsNull() {
		return false, nil
	}

	if stateExpiresAt.IsNull() || stateExpiresAt.IsUnknown() || stateExpiresAt != planExpiresAt {
		stateExpiresAtValue := ""
		if !stateExpiresAt.IsNull() && !stateExpiresAt.IsUnknown() {
			stateExpiresAtValue = stateExpiresAt.ValueString()
		}
		planExpiresAtValue := ""
		if !planExpiresAt.IsNull() && !planExpiresAt.IsUnknown() {
			planExpiresAtValue = planExpiresAt.ValueString()
		}
		tflog.Debug(ctx, fmt.Sprintf("[%s] State is not populated, or the expires_at value changed. Token needs rotation.", logPrefix), map[string]any{
			"state_expires_at": stateExpiresAtValue,
			"plan_expires_at":  planExpiresAtValue,
		})
		return true, nil
	}

	// Check if rotation configuration is present
	if !planToken.HasRotationConfiguration() {
		return false, nil
	}

	// Check if rotation configuration changed
	configChanged := RotationConfigurationChanged(stateToken, planToken)

	if configChanged {
		tflog.Debug(ctx, fmt.Sprintf("[%s] Rotation configuration changed, triggering immediate rotation.", logPrefix), map[string]any{
			"state_expiration_days":    stateToken.GetExpirationDays().ValueInt64(),
			"plan_expiration_days":     planToken.GetExpirationDays().ValueInt64(),
			"state_rotate_before_days": stateToken.GetRotateBeforeDays().ValueInt64(),
			"plan_rotate_before_days":  planToken.GetRotateBeforeDays().ValueInt64(),
		})
		return true, nil
	}

	// Check time-based rotation
	expiresAt, err := time.Parse(api.Iso8601, stateExpiresAt.ValueString())
	if err != nil {
		return false, fmt.Errorf("failed to parse expiration date %q: %w", stateExpiresAt.ValueString(), err)
	}

	rotateBeforeDays := planToken.GetRotateBeforeDays()
	if rotateBeforeDays.IsNull() || rotateBeforeDays.IsUnknown() {
		return false, fmt.Errorf("rotate_before_days is not set")
	}

	// Calculate rotation date by subtracting rotate_before_days from expiration
	rotationDate := expiresAt.Add(-time.Duration(rotateBeforeDays.ValueInt64()) * 24 * time.Hour)

	// Token needs rotation if current time is on or after the rotation date
	needsRotation := !now.Before(rotationDate)

	tflog.Debug(ctx, fmt.Sprintf("[%s] Checked time-based rotation.", logPrefix), map[string]any{
		"expires_at":         stateExpiresAt.ValueString(),
		"rotate_before_days": rotateBeforeDays.ValueInt64(),
		"rotation_date":      rotationDate.Format(api.Iso8601),
		"current_time":       now.Format(api.Iso8601),
		"needs_rotation":     needsRotation,
	})

	return needsRotation, nil
}

// ApplyTokenRotation calculates the new expiration date and updates the token plan if rotation is needed.
// It returns a boolean indicating whether the plan was modified.
// This function encapsulates the duplicated logic found in ModifyPlan methods across token resources.
// The fallbackExpirationDays parameter is used for service account tokens which have a default expiration.
func ApplyTokenRotation(
	ctx context.Context,
	token RotatableToken,
	rotationConfig *GitlabAccessTokenRotationConfiguration,
	stateData RotatableToken,
	fallbackExpirationDays *int,
) (bool, error) {
	logPrefix := token.GetLogPrefix()
	// Calculate new expiry date
	expiryDate, _, err := DetermineExpiryDate(token.GetExpiresAt(), rotationConfig, fallbackExpirationDays)
	if err != nil {
		return false, fmt.Errorf("could not determine new expiry date: %w", err)
	}

	// Check if the newly calculated expiryDate is different than what's in state
	// This check prevents the ID being unknown on every apply with rotation_configuration
	// even if the calculated date is exactly the same as it currently is
	stateExpiresAt := stateData.GetExpiresAt()
	// Check if dates are different, handling null cases properly
	var dateChanged bool
	if expiryDate.IsNull() {
		// Plan wants null expiry, check if state has a non-null expiry
		dateChanged = !stateExpiresAt.IsNull()
	} else if !expiryDate.IsUnknown() {
		// Plan has a concrete date
		if stateExpiresAt.IsNull() || stateExpiresAt.IsUnknown() {
			// State is null/unknown but plan has a date
			dateChanged = true
		} else {
			// Both have dates, compare them
			dateChanged = expiryDate.ValueString() != stateExpiresAt.ValueString()
		}
	}

	if !dateChanged {
		return false, nil
	}

	// Update the plan using the interface setters
	token.SetExpiresAt(expiryDate)
	token.SetID(types.StringUnknown())
	token.SetToken(types.StringUnknown())
	token.SetCreatedAt(types.StringUnknown())

	// Log the rotation action
	tflog.Debug(ctx, fmt.Sprintf("[%s] Rotation is required, updating plan data", logPrefix), map[string]any{
		"new_expires_at": expiryDate.ValueString(),
		"old_expires_at": stateExpiresAt.ValueString(),
	})

	return true, nil
}
