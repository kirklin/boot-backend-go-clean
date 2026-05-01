package server

import (
	"testing"
	"time"
)

// TestGlobalTimezoneIsUTC ensures that the application strictly operates in UTC.
// This test acts as a safeguard against any third-party library or developer
// accidentally modifying time.Local or failing to configure UTC.
func TestGlobalTimezoneIsUTC(t *testing.T) {
	// 1. Initialize the application which locks the timezone.
	// We ignore the error because config validation might fail in the test environment,
	// but the timezone lock happens at the very beginning of NewApplication().
	_, _ = NewApplication()

	// 2. Check if time.Local is explicitly UTC
	if time.Local != time.UTC {
		t.Errorf("CRITICAL: Global timezone (time.Local) is not locked to time.UTC. Found: %v", time.Local)
	}

	// 3. Check if time.Now() defaults to UTC
	zone, offset := time.Now().Zone()
	if zone != "UTC" || offset != 0 {
		t.Errorf("CRITICAL: time.Now() is not generating UTC time! Found Zone: %s, Offset: %d", zone, offset)
	}
}
