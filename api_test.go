package main

import (
	"testing"
	"time"
)

func TestParsePCOTimePreservesWallClockTimeForRFC3339Values(t *testing.T) {
	originalLocation := pcoTimeLocation
	testLocation := time.FixedZone("MDT", -6*60*60)
	pcoTimeLocation = testLocation
	t.Cleanup(func() {
		pcoTimeLocation = originalLocation
	})

	parsed, ok := parsePCOTime("2026-03-08T09:00:00Z")
	if !ok {
		t.Fatal("expected timestamp to parse")
	}

	if parsed.Location() != testLocation {
		t.Fatalf("expected %q location, got %q", testLocation, parsed.Location())
	}
	if parsed.Hour() != 9 {
		t.Fatalf("expected 9am wall-clock time, got %s", parsed.Format(time.RFC3339))
	}
}

func TestParsePCOTimePreservesWallClockTimeForOffsetValues(t *testing.T) {
	originalLocation := pcoTimeLocation
	testLocation := time.FixedZone("MST", -7*60*60)
	pcoTimeLocation = testLocation
	t.Cleanup(func() {
		pcoTimeLocation = originalLocation
	})

	parsed, ok := parsePCOTime("2026-12-13T09:00:00-06:00")
	if !ok {
		t.Fatal("expected timestamp to parse")
	}

	if parsed.Location() != testLocation {
		t.Fatalf("expected %q location, got %q", testLocation, parsed.Location())
	}
	if parsed.Hour() != 9 {
		t.Fatalf("expected 9am wall-clock time, got %s", parsed.Format(time.RFC3339))
	}
}

func TestParsePCOTimeParsesDateOnlyValuesInConfiguredLocation(t *testing.T) {
	originalLocation := pcoTimeLocation
	testLocation := time.FixedZone("MST", -7*60*60)
	pcoTimeLocation = testLocation
	t.Cleanup(func() {
		pcoTimeLocation = originalLocation
	})

	parsed, ok := parsePCOTime("2026-03-08")
	if !ok {
		t.Fatal("expected date to parse")
	}

	if parsed.Location() != testLocation {
		t.Fatalf("expected %q location, got %q", testLocation, parsed.Location())
	}
	if parsed.Hour() != 0 || parsed.Minute() != 0 {
		t.Fatalf("expected local midnight, got %s", parsed.Format(time.RFC3339))
	}
}
