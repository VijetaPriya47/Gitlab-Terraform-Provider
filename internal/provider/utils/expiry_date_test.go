package utils

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const mockTimeNow = "2025-10-01T10:00:00Z"

func TestValidateExpiryDate(t *testing.T) {
	// Set a fixed "current time" for predictable test results.
	// t.Setenv automatically cleans this up after the test is done.
	t.Setenv("GITLAB_TESTING_TIME", mockTimeNow)
	fixedNow, _ := time.Parse(time.RFC3339, mockTimeNow)

	// --- Test for ValidateExpiryDateValid ---
	t.Run("ValidateExpiryDateValid", func(t *testing.T) {
		cases := []struct {
			name        string
			expiryDate  time.Time
			expectError bool
		}{
			{
				name:        "DateInTheFuture",
				expiryDate:  fixedNow.Add(24 * time.Hour),
				expectError: false,
			},
			{
				name:        "DateInThePast",
				expiryDate:  fixedNow.Add(-24 * time.Hour),
				expectError: true,
			},
			{
				name:        "DateIsExactlyNow",
				expiryDate:  fixedNow,
				expectError: false,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := ValidateExpiryDateValid(tc.expiryDate)
				hasError := err != nil

				if hasError != tc.expectError {
					t.Fatalf("ValidateExpiryDateValid() FAILED, expected error -> %v, got -> %v", tc.expectError, err)
				}
			})
		}
	})

	// --- Test for ValidateISOTimeExpiryDate ---
	t.Run("ValidateISOTimeExpiryDate", func(t *testing.T) {
		cases := []struct {
			name        string
			expiryDate  gitlab.ISOTime
			expectError bool
		}{
			{
				name:        "ISODateInTheFuture",
				expiryDate:  gitlab.ISOTime(fixedNow.Add(48 * time.Hour)),
				expectError: false,
			},
			{
				name:        "ISODateInThePast",
				expiryDate:  gitlab.ISOTime(fixedNow.Add(-48 * time.Hour)),
				expectError: true,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := ValidateISOTimeExpiryDate(tc.expiryDate)
				hasError := err != nil

				if hasError != tc.expectError {
					t.Fatalf("ValidateISOTimeExpiryDate() FAILED, expected error -> %v, got -> %v", tc.expectError, err)
				}
			})
		}
	})
}

func TestDetermineRFC3339ExpiryDate(t *testing.T) {
	expectedTime, _ := time.Parse(time.RFC3339, mockTimeNow)
	inputTime, _ := timetypes.NewRFC3339Value(mockTimeNow)

	cases := []struct {
		name         string
		input        timetypes.RFC3339
		expectedTime *time.Time
		expectError  bool
	}{
		{
			name:         "ValidRFC3339Time",
			input:        inputTime,
			expectedTime: &expectedTime,
			expectError:  false,
		},
		{
			name:         "NullTime",
			input:        timetypes.NewRFC3339Null(),
			expectedTime: nil,
			expectError:  false,
		},
		{
			name:         "UnknownTime",
			input:        timetypes.NewRFC3339Unknown(),
			expectedTime: nil,
			expectError:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DetermineRFC3339ExpiryDate(tc.input)
			hasError := err != nil

			if hasError != tc.expectError {
				t.Fatalf("DetermineRFC3339ExpiryDate() FAILED, expected error -> %v, got -> %v", tc.expectError, err)
			}

			if (result == nil && tc.expectedTime != nil) || (result != nil && tc.expectedTime == nil) {
				t.Fatalf("DetermineRFC3339ExpiryDate() FAILED, expected time -> %v, got -> %v", tc.expectedTime, result)
			}

			if result != nil && !result.Equal(*tc.expectedTime) {
				t.Fatalf("DetermineRFC3339ExpiryDate() FAILED, expected time -> %v, got -> %v", *tc.expectedTime, *result)
			}
		})
	}
}

func TestDetermineExpiryDate(t *testing.T) {
	t.Setenv("GITLAB_TESTING_TIME", mockTimeNow)
	fallbackDays := 10

	cases := []struct {
		name           string
		expiresAt      types.String
		rotationConfig *GitlabAccessTokenRotationConfiguration
		fallback       *int
		expectedDate   string // YYYY-MM-DD format for simple comparison
		expectError    bool
	}{
		{
			name:         "UsesExpiresAtWhenProvided",
			expiresAt:    types.StringValue("2026-05-20"),
			expectedDate: "2026-05-20",
		},
		{
			name:        "FailsOnInvalidExpiresAt",
			expiresAt:   types.StringValue("not-a-date"),
			expectError: true,
		},
		{
			name: "UsesRotationConfigExpirationDays",
			rotationConfig: &GitlabAccessTokenRotationConfiguration{
				ExpirationDays: types.Int64Value(30),
			},
			// mockTimeNow is 2025-10-01, so +30 days is 2025-10-31
			expectedDate: "2025-10-31",
		},
		{
			name: "UsesFallbackWhenRotationDaysAreNull",
			rotationConfig: &GitlabAccessTokenRotationConfiguration{
				ExpirationDays: types.Int64Null(),
			},
			fallback: &fallbackDays,
			// mockTimeNow is 2025-10-01, so +10 days is 2025-10-11
			expectedDate: "2025-10-11",
		},
		{
			name: "ReturnsNilWhenRotationDaysAndFallbackAreNull",
			rotationConfig: &GitlabAccessTokenRotationConfiguration{
				ExpirationDays: types.Int64Null(),
			},
			fallback:     nil,
			expectedDate: "", // Represents nil
		},
		{
			name:         "ReturnsNilWhenAllInputsAreNil",
			expectedDate: "", // Represents nil
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DetermineExpiryDate(tc.expiresAt, tc.rotationConfig, tc.fallback)
			hasError := err != nil

			if hasError != tc.expectError {
				t.Fatalf("DetermineExpiryDate() FAILED, expected error -> %v, got -> %v", tc.expectError, err)
			}

			var resultDate string
			if result != nil {
				// We format to a common standard for easy string comparison
				resultDate = time.Time(*result).Format("2006-01-02")
			}

			if resultDate != tc.expectedDate {
				t.Fatalf("DetermineExpiryDate() FAILED, expected date -> '%s', got -> '%s'", tc.expectedDate, resultDate)
			}
		})
	}
}
