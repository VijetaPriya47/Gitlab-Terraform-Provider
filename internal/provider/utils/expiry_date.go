package utils

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

type GitlabAccessTokenRotationConfiguration struct {
	ExpirationDays   types.Int64 `tfsdk:"expiration_days"`
	RotateBeforeDays types.Int64 `tfsdk:"rotate_before_days"`
}

func ValidateISOTimeExpiryDate(expiryDate gitlab.ISOTime) error {
	return ValidateExpiryDateValid(time.Time(expiryDate))
}

func ValidateExpiryDateValid(expiryDate time.Time) error {
	now := api.CurrentTime()
	if now.After(expiryDate) {
		return fmt.Errorf("Expiry date %s must be in the future. Current time is %s", expiryDate.Format(time.RFC3339), now.Format(time.RFC3339))
	}
	return nil
}

func DetermineExpiryDate(expiresAt types.String, rotationConfiguration *GitlabAccessTokenRotationConfiguration, expirationDaysFallback *int) (*gitlab.ISOTime, error) {
	if !expiresAt.IsNull() && !expiresAt.IsUnknown() && rotationConfiguration == nil {
		isoTime, err := gitlab.ParseISOTime(expiresAt.ValueString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse expiration date into ISOTime. Provided value: %s", expiresAt.ValueString())
		}
		return &isoTime, nil
	}

	if rotationConfiguration != nil {
		if !rotationConfiguration.ExpirationDays.IsNull() && !rotationConfiguration.ExpirationDays.IsUnknown() {
			now := api.CurrentTime()
			expiryDate := now.AddDate(0, 0, int(rotationConfiguration.ExpirationDays.ValueInt64()))
			expiryIsoTime, err := gitlab.ParseISOTime(expiryDate.Format(api.Iso8601))

			return &expiryIsoTime, err
		}

		if expirationDaysFallback != nil {
			now := api.CurrentTime()
			expiryDate := now.AddDate(0, 0, *expirationDaysFallback)
			expiryIsoTime, err := gitlab.ParseISOTime(expiryDate.Format(api.Iso8601))

			return &expiryIsoTime, err
		}
	}

	return nil, nil
}

func DetermineRFC3339ExpiryDate(expiresAt timetypes.RFC3339) (*time.Time, error) {
	if !expiresAt.IsNull() && !expiresAt.IsUnknown() {
		parsedExpiresAt, err := expiresAt.ValueRFC3339Time()
		if err != nil {
			return nil, fmt.Errorf("Failed to determine expiration date. Provided value: %s", expiresAt.ValueString())
		}

		return &parsedExpiresAt, nil
	}

	return nil, nil
}
