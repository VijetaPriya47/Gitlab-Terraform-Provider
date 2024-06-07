package utils

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// The regex to use for masking the token configurations
const (
	gitlabPersonlAccessTokenRegex string = "glpat-.{20}"
	// see https://gitlab.com/gitlab-org/gitlab/-/blob/9d8d8dd9f1ff51aed56596ba5f6dcf0e6d723b73/app/models/concerns/runners_token_prefixable.rb#L1-9
	gitlabLegacyRunnersTokenRegex string = "GR1348941.{20}"
	// see https://gitlab.com/gitlab-org/gitlab/blob/628cd01f94f1cdd0847ca2faa740c5a048455410/app/models/ci/runner.rb#L495-499
	gitlabRunnersTokenRegex string = "glrt-.{20}"
)

// Takes in a provider context, and applies masking to the root and `GitLab` subsystem logger.
// the `GitLab` subsystem is what is used for logging API calls.
func ApplyLogMaskingToContext(ctx context.Context) context.Context {
	maskRegexes := []*regexp.Regexp{
		regexp.MustCompile(gitlabPersonlAccessTokenRegex),
		regexp.MustCompile(gitlabLegacyRunnersTokenRegex),
		regexp.MustCompile(gitlabRunnersTokenRegex),
	}
	subSystemName := "GitLab"

	ctx = tflog.NewSubsystem(ctx, subSystemName)
	for _, maskRegex := range maskRegexes {
		// Configure the "root" logger within the context
		// This will mask any logging done directly using the tflog command that doesn't explicitly call a subsystem.
		ctx = tflog.MaskMessageRegexes(ctx, maskRegex)
		ctx = tflog.MaskLogRegexes(ctx, maskRegex)
		ctx = tflog.MaskAllFieldValuesRegexes(ctx, maskRegex)

		// The "GitLab" subsystem is what is used for logging API messages, so configure it here as well.
		ctx = tflog.SubsystemMaskMessageRegexes(ctx, subSystemName, maskRegex)
		ctx = tflog.SubsystemMaskLogRegexes(ctx, subSystemName, maskRegex)
		ctx = tflog.SubsystemMaskAllFieldValuesRegexes(ctx, subSystemName, maskRegex)

	}

	return ctx
}
