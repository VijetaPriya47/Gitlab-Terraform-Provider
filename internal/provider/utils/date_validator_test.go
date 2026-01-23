package utils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIsValidDate(t *testing.T) {
	tests := []struct {
		name        string
		value       types.String
		expectError bool
	}{
		{
			name:        "valid date",
			value:       types.StringValue("2025-01-22"),
			expectError: false,
		},
		{
			name:        "valid leap year date",
			value:       types.StringValue("2024-02-29"),
			expectError: false,
		},
		{
			name:        "invalid leap year date",
			value:       types.StringValue("2025-02-29"),
			expectError: true,
		},
		{
			name:        "invalid month",
			value:       types.StringValue("2025-13-01"),
			expectError: true,
		},
		{
			name:        "invalid day",
			value:       types.StringValue("2025-01-32"),
			expectError: true,
		},
		{
			name:        "wrong format - slashes",
			value:       types.StringValue("2025/01/22"),
			expectError: true,
		},
		{
			name:        "wrong format - short year",
			value:       types.StringValue("25-01-22"),
			expectError: true,
		},
		{
			name:        "wrong format - with time",
			value:       types.StringValue("2025-01-22T03:45:40Z"),
			expectError: true,
		},
		{
			name:        "null value",
			value:       types.StringNull(),
			expectError: false,
		},
		{
			name:        "unknown value",
			value:       types.StringUnknown(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			IsValidDate().ValidateString(context.Background(), req, resp)

			if tt.expectError && !resp.Diagnostics.HasError() {
				t.Error("expected error, got none")
			}
			if !tt.expectError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}
