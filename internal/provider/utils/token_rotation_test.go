package utils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mockRotatableToken implements the RotatableToken interface for testing
type mockRotatableToken struct {
	expiresAt        types.String
	expirationDays   types.Int64
	rotateBeforeDays types.Int64
	hasConfig        bool
}

func (m *mockRotatableToken) GetExpiresAt() types.String {
	return m.expiresAt
}

func (m *mockRotatableToken) GetExpirationDays() types.Int64 {
	return m.expirationDays
}

func (m *mockRotatableToken) GetRotateBeforeDays() types.Int64 {
	return m.rotateBeforeDays
}

func (m *mockRotatableToken) HasRotationConfiguration() bool {
	return m.hasConfig
}

func (m *mockRotatableToken) SetExpiresAt(v types.String) {
	m.expiresAt = v
}

func (m *mockRotatableToken) SetID(v types.String) {}

func (m *mockRotatableToken) SetToken(v types.String) {}

func (m *mockRotatableToken) SetCreatedAt(v types.String) {}

func (m *mockRotatableToken) GetLogPrefix() string {
	return "TestToken"
}

func TestShouldRotateToken_NewToken(t *testing.T) {
	t.Setenv("GITLAB_TESTING_TIME", "2024-12-15T00:00:00Z")
	ctx := context.Background()

	token := &mockRotatableToken{
		expiresAt:        types.StringValue("2024-12-31"),
		expirationDays:   types.Int64Value(30),
		rotateBeforeDays: types.Int64Value(7),
		hasConfig:        true,
	}

	// When state is nil (new token), should rotate
	shouldRotate, err := ShouldRotateToken(ctx, token, nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !shouldRotate {
		t.Errorf("expected shouldRotate=true for new token, got false")
	}
}

func TestShouldRotateToken_ExpiresAtChanged(t *testing.T) {
	t.Setenv("GITLAB_TESTING_TIME", "2024-12-15T00:00:00Z")
	ctx := context.Background()

	planToken := &mockRotatableToken{
		expiresAt:        types.StringValue("2024-12-31"),
		expirationDays:   types.Int64Value(30),
		rotateBeforeDays: types.Int64Value(7),
		hasConfig:        true,
	}

	// When expires_at changed in configuration
	stateToken := &mockRotatableToken{
		expiresAt:        types.StringValue("2024-12-25"), // different from plan
		expirationDays:   types.Int64Value(30),
		rotateBeforeDays: types.Int64Value(7),
		hasConfig:        true,
	}

	shouldRotate, err := ShouldRotateToken(ctx, planToken, stateToken)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !shouldRotate {
		t.Errorf("expected shouldRotate=true when expires_at changed, got false")
	}
}

func TestShouldRotateToken_ConfigurationChanged(t *testing.T) {
	t.Setenv("GITLAB_TESTING_TIME", "2024-12-15T00:00:00Z")
	ctx := context.Background()

	tests := []struct {
		name                  string
		stateExpirationDays   types.Int64
		planExpirationDays    types.Int64
		stateRotateBeforeDays types.Int64
		planRotateBeforeDays  types.Int64
		shouldRotate          bool
	}{
		{
			name:                  "expiration_days changed",
			stateExpirationDays:   types.Int64Value(30),
			planExpirationDays:    types.Int64Value(20),
			stateRotateBeforeDays: types.Int64Value(7),
			planRotateBeforeDays:  types.Int64Value(7),
			shouldRotate:          true,
		},
		{
			name:                  "rotate_before_days changed",
			stateExpirationDays:   types.Int64Value(30),
			planExpirationDays:    types.Int64Value(30),
			stateRotateBeforeDays: types.Int64Value(7),
			planRotateBeforeDays:  types.Int64Value(3),
			shouldRotate:          true,
		},
		{
			name:                  "no changes",
			stateExpirationDays:   types.Int64Value(30),
			planExpirationDays:    types.Int64Value(30),
			stateRotateBeforeDays: types.Int64Value(7),
			planRotateBeforeDays:  types.Int64Value(7),
			shouldRotate:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planToken := &mockRotatableToken{
				expiresAt:        types.StringValue("2024-12-31"),
				expirationDays:   tt.planExpirationDays,
				rotateBeforeDays: tt.planRotateBeforeDays,
				hasConfig:        true,
			}
			stateToken := &mockRotatableToken{
				expiresAt:        types.StringValue("2024-12-31"), // same as plan
				expirationDays:   tt.stateExpirationDays,
				rotateBeforeDays: tt.stateRotateBeforeDays,
				hasConfig:        true,
			}

			shouldRotate, err := ShouldRotateToken(ctx, planToken, stateToken)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if shouldRotate != tt.shouldRotate {
				t.Errorf("expected shouldRotate=%v, got %v", tt.shouldRotate, shouldRotate)
			}
		})
	}
}

func TestShouldRotateToken_TimeBasedRotation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		expiresAt    string
		rotateBefore int64
		currentTime  string
		shouldRotate bool
	}{
		{
			name:         "needs rotation - current time after rotation date",
			expiresAt:    "2024-12-31",
			rotateBefore: 7,
			currentTime:  "2024-12-25T00:00:00Z", // After 2024-12-24 rotation date
			shouldRotate: true,
		},
		{
			name:         "needs rotation - current time equals rotation date",
			expiresAt:    "2024-12-31",
			rotateBefore: 7,
			currentTime:  "2024-12-24T00:00:00Z", // Exactly at rotation date
			shouldRotate: true,
		},
		{
			name:         "does not need rotation - current time before rotation date",
			expiresAt:    "2024-12-31",
			rotateBefore: 7,
			currentTime:  "2024-12-23T00:00:00Z", // Before 2024-12-24 rotation date
			shouldRotate: false,
		},
		{
			name:         "needs rotation - 1 day before expiry with 1 day rotate_before",
			expiresAt:    "2024-12-31",
			rotateBefore: 1,
			currentTime:  "2024-12-30T00:00:00Z",
			shouldRotate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITLAB_TESTING_TIME", tt.currentTime)

			planToken := &mockRotatableToken{
				expiresAt:        types.StringValue(tt.expiresAt),
				expirationDays:   types.Int64Value(30),
				rotateBeforeDays: types.Int64Value(tt.rotateBefore),
				hasConfig:        true,
			}
			stateToken := &mockRotatableToken{
				expiresAt:        types.StringValue(tt.expiresAt), // same as plan
				expirationDays:   types.Int64Value(30),
				rotateBeforeDays: types.Int64Value(tt.rotateBefore),
				hasConfig:        true,
			}

			shouldRotate, err := ShouldRotateToken(ctx, planToken, stateToken)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if shouldRotate != tt.shouldRotate {
				t.Errorf("expected shouldRotate=%v, got %v", tt.shouldRotate, shouldRotate)
			}
		})
	}
}

func TestShouldRotateToken_NoRotationConfiguration(t *testing.T) {
	t.Setenv("GITLAB_TESTING_TIME", "2024-12-15T00:00:00Z")
	ctx := context.Background()

	planToken := &mockRotatableToken{
		expiresAt:        types.StringValue("2024-12-31"),
		expirationDays:   types.Int64Null(),
		rotateBeforeDays: types.Int64Null(),
		hasConfig:        false,
	}
	stateToken := &mockRotatableToken{
		expiresAt:        types.StringValue("2024-12-31"), // same as plan
		expirationDays:   types.Int64Null(),
		rotateBeforeDays: types.Int64Null(),
		hasConfig:        false,
	}

	// When there's no rotation configuration and expires_at hasn't changed
	shouldRotate, err := ShouldRotateToken(ctx, planToken, stateToken)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if shouldRotate {
		t.Errorf("expected shouldRotate=false when no rotation config, got true")
	}
}

func TestRotationConfigurationChanged(t *testing.T) {
	tests := []struct {
		name                  string
		stateExpirationDays   types.Int64
		planExpirationDays    types.Int64
		stateRotateBeforeDays types.Int64
		planRotateBeforeDays  types.Int64
		expectChanged         bool
	}{
		{
			name:                  "expiration_days changed from 30 to 10",
			stateExpirationDays:   types.Int64Value(30),
			planExpirationDays:    types.Int64Value(10),
			stateRotateBeforeDays: types.Int64Value(7),
			planRotateBeforeDays:  types.Int64Value(7),
			expectChanged:         true,
		},
		{
			name:                  "expiration_days changed from 30 to 25",
			stateExpirationDays:   types.Int64Value(30),
			planExpirationDays:    types.Int64Value(25),
			stateRotateBeforeDays: types.Int64Value(7),
			planRotateBeforeDays:  types.Int64Value(7),
			expectChanged:         true,
		},
		{
			name:                  "rotate_before_days changed from 7 to 3",
			stateExpirationDays:   types.Int64Value(30),
			planExpirationDays:    types.Int64Value(30),
			stateRotateBeforeDays: types.Int64Value(7),
			planRotateBeforeDays:  types.Int64Value(3),
			expectChanged:         true,
		},
		{
			name:                  "no changes - both values same",
			stateExpirationDays:   types.Int64Value(30),
			planExpirationDays:    types.Int64Value(30),
			stateRotateBeforeDays: types.Int64Value(7),
			planRotateBeforeDays:  types.Int64Value(7),
			expectChanged:         false,
		},
		{
			name:                  "both values changed",
			stateExpirationDays:   types.Int64Value(30),
			planExpirationDays:    types.Int64Value(20),
			stateRotateBeforeDays: types.Int64Value(7),
			planRotateBeforeDays:  types.Int64Value(5),
			expectChanged:         true,
		},
		{
			name:                  "null state values - no change detected",
			stateExpirationDays:   types.Int64Null(),
			planExpirationDays:    types.Int64Value(30),
			stateRotateBeforeDays: types.Int64Null(),
			planRotateBeforeDays:  types.Int64Value(7),
			expectChanged:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateToken := &mockRotatableToken{
				expirationDays:   tt.stateExpirationDays,
				rotateBeforeDays: tt.stateRotateBeforeDays,
			}
			planToken := &mockRotatableToken{
				expirationDays:   tt.planExpirationDays,
				rotateBeforeDays: tt.planRotateBeforeDays,
			}
			changed := RotationConfigurationChanged(stateToken, planToken)
			if changed != tt.expectChanged {
				t.Errorf("expected configuration changed=%v but got %v", tt.expectChanged, changed)
			}
		})
	}
}

func TestShouldRotateToken_NullExpirationScenarios(t *testing.T) {
	t.Setenv("GITLAB_TESTING_TIME", "2024-12-15T00:00:00Z")
	ctx := context.Background()

	tests := []struct {
		name           string
		planExpiresAt  types.String
		stateExpiresAt types.String
		shouldRotate   bool
		description    string
	}{
		{
			name:           "both null - no rotation",
			planExpiresAt:  types.StringNull(),
			stateExpiresAt: types.StringNull(),
			shouldRotate:   false,
			description:    "When both state and plan have null expiry, no rotation needed",
		},
		{
			name:           "state null, plan has date - rotation needed",
			planExpiresAt:  types.StringValue("2024-12-31"),
			stateExpiresAt: types.StringNull(),
			shouldRotate:   true,
			description:    "New token creation with expiry date",
		},
		{
			name:           "state has date, plan null - rotation needed",
			planExpiresAt:  types.StringNull(),
			stateExpiresAt: types.StringValue("2024-12-31"),
			shouldRotate:   true,
			description:    "Removing expiry from existing token (self-hosted instances)",
		},
		{
			name:           "both have same date - no rotation",
			planExpiresAt:  types.StringValue("2024-12-31"),
			stateExpiresAt: types.StringValue("2024-12-31"),
			shouldRotate:   false,
			description:    "No change in expiry date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planToken := &mockRotatableToken{
				expiresAt:        tt.planExpiresAt,
				expirationDays:   types.Int64Null(),
				rotateBeforeDays: types.Int64Null(),
				hasConfig:        false,
			}
			stateToken := &mockRotatableToken{
				expiresAt:        tt.stateExpiresAt,
				expirationDays:   types.Int64Null(),
				rotateBeforeDays: types.Int64Null(),
				hasConfig:        false,
			}

			shouldRotate, err := ShouldRotateToken(ctx, planToken, stateToken)
			if err != nil {
				t.Fatalf("unexpected error: %s - %s", tt.description, err)
			}
			if shouldRotate != tt.shouldRotate {
				t.Errorf("%s: expected shouldRotate=%v, got %v", tt.description, tt.shouldRotate, shouldRotate)
			}
		})
	}
}

func TestApplyTokenRotation_NullExpirationScenarios(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		planExpiresAt  types.String
		stateExpiresAt types.String
		expectModified bool
		description    string
	}{
		{
			name:           "both null - no modification",
			planExpiresAt:  types.StringNull(),
			stateExpiresAt: types.StringNull(),
			expectModified: false,
			description:    "When both are null, plan should not be modified",
		},
		{
			name:           "state null, plan has date - modification needed",
			planExpiresAt:  types.StringValue("2024-12-31"),
			stateExpiresAt: types.StringNull(),
			expectModified: true,
			description:    "Adding expiry date to token",
		},
		{
			name:           "state has date, plan null - modification needed",
			planExpiresAt:  types.StringNull(),
			stateExpiresAt: types.StringValue("2024-12-31"),
			expectModified: true,
			description:    "Removing expiry date from token",
		},
		{
			name:           "both have same date - no modification",
			planExpiresAt:  types.StringValue("2024-12-31"),
			stateExpiresAt: types.StringValue("2024-12-31"),
			expectModified: false,
			description:    "Same expiry date, no change needed",
		},
		{
			name:           "both have different dates - modification needed",
			planExpiresAt:  types.StringValue("2025-01-15"),
			stateExpiresAt: types.StringValue("2024-12-31"),
			expectModified: true,
			description:    "Expiry date changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planToken := &mockRotatableToken{
				expiresAt:        tt.planExpiresAt,
				expirationDays:   types.Int64Null(),
				rotateBeforeDays: types.Int64Null(),
				hasConfig:        false,
			}

			stateToken := &mockRotatableToken{
				expiresAt:        tt.stateExpiresAt,
				expirationDays:   types.Int64Null(),
				rotateBeforeDays: types.Int64Null(),
				hasConfig:        false,
			}

			modified, err := ApplyTokenRotation(
				ctx,
				planToken,
				nil, // no rotation config
				stateToken,
				nil, // no fallback
			)

			if err != nil {
				t.Fatalf("unexpected error: %s - %s", tt.description, err)
			}
			if modified != tt.expectModified {
				t.Errorf("%s: expected modified=%v, got %v", tt.description, tt.expectModified, modified)
			}
		})
	}
}
