package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func AugmentVariableClientError(ctx context.Context, masked bool, err error, d diag.Diagnostics) bool {
	if !masked {
		return false
	}

	var httpErr *gitlab.ErrorResponse
	if errors.As(err, &httpErr) {
		if httpErr.Response.StatusCode == http.StatusBadRequest &&
			strings.Contains(httpErr.Message, "value") &&
			strings.Contains(httpErr.Message, "invalid") {
			tflog.Error(ctx, fmt.Sprintf("[ERROR] %v", err))
			d.AddError("Invalid value for a masked variable. Check the masked variable requirements: https://docs.gitlab.com/ee/ci/variables/#masked-variable-requirements", err.Error())
			return true
		}
	}
	return false
}
