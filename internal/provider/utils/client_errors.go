package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func AugmentVariableClientError(ctx context.Context, masked bool, err error) (bool, error) {
	if !masked {
		return false, nil
	}

	var httpErr *gitlab.ErrorResponse
	if errors.As(err, &httpErr) {
		if httpErr.Response.StatusCode == http.StatusBadRequest &&
			strings.Contains(httpErr.Message, "value") &&
			strings.Contains(httpErr.Message, "invalid") {
			tflog.Error(ctx, fmt.Sprintf("[ERROR] %v", err))
			return true, err
		}
	}
	return false, nil
}
