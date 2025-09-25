package utils

import (
	"testing"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestISOTimeConversion(t *testing.T) {
	originalTimeString := "2025-09-22"

	isoTime, err := gitlab.ParseISOTime(originalTimeString)
	if err != nil {
		t.Fatalf("Test setup failed: could not parse time string: %v", err)
	}

	convertedTime := time.Time(isoTime)
	convertedStrTime := convertedTime.Format("2006-01-02")

	if convertedStrTime != originalTimeString {
		t.Errorf("Conversion from *ISOTime to time.Time was incorrect.\n got: %s\nwant: %s", convertedStrTime, originalTimeString)
	}
}
