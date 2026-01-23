package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = isValidDateValidator{}

type isValidDateValidator struct{}

func (v isValidDateValidator) Description(ctx context.Context) string {
	return "string must be a valid date in YYYY-MM-DD format"
}

func (v isValidDateValidator) MarkdownDescription(ctx context.Context) string {
	return "string must be a valid date in `YYYY-MM-DD` format (for example, `2025-01-22`)"
}

func (v isValidDateValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Attempt to parse the date using the YYYY-MM-DD layout
	// Go's reference date is "2006-01-02" which represents YYYY-MM-DD
	_, err := time.Parse("2006-01-02", value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Date Format",
			fmt.Sprintf("Value %q is not in the expected date format. It must be in format YYYY-MM-DD (for example, 2025-01-22).", value),
		)
		return
	}
}

func IsValidDate() validator.String {
	return isValidDateValidator{}
}
