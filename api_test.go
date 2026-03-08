package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizePlanCapturesFirstServiceTimeFromIncludedPlanTimes(t *testing.T) {
	originalLocation := pcoTimeLocation
	testLocation := time.FixedZone("MDT", -6*60*60)
	pcoTimeLocation = testLocation
	t.Cleanup(func() {
		pcoTimeLocation = originalLocation
	})

	plan := normalizePlan(
		pcoPlanResource{
			ID: "plan-1",
			Attributes: pcoPlanAttributes{
				Title:       "Sunday",
				SeriesTitle: "Series",
				SortDate:    "2026-03-08T07:00:00Z",
				LastTimeAt:  stringPtr("2026-03-08T11:00:00Z"),
			},
			Relationships: pcoPlanRelationships{
				PlanTimes: pcoRelationshipCollection{
					Data: []pcoRelationshipData{
						{ID: "call", Type: "PlanTime"},
						{ID: "service-2", Type: "PlanTime"},
						{ID: "service-1", Type: "PlanTime"},
					},
				},
			},
		},
		[]pcoIncludedResource{
			includedPlanTime("call", "other", "2026-03-08T07:00:00Z"),
			includedPlanTime("service-2", "service", "2026-03-08T11:00:00Z"),
			includedPlanTime("service-1", "service", "2026-03-08T09:00:00Z"),
		},
	)

	if plan.FirstServiceAt == nil {
		t.Fatal("expected first service time to be populated")
	}
	if plan.FirstServiceAt.Hour() != 9 {
		t.Fatalf("expected first service at 9am, got %s", plan.FirstServiceAt.Format(time.RFC3339))
	}
	if plan.SortDate.Hour() != 7 {
		t.Fatalf("expected sort date to remain distinct, got %s", plan.SortDate.Format(time.RFC3339))
	}
}

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

func stringPtr(value string) *string {
	return &value
}

func includedPlanTime(id, timeType, startsAt string) pcoIncludedResource {
	return pcoIncludedResource{
		ID:   id,
		Type: "PlanTime",
		Attributes: map[string]json.RawMessage{
			"time_type": mustRawJSON(`"` + timeType + `"`),
			"starts_at": mustRawJSON(`"` + startsAt + `"`),
		},
	}
}

func mustRawJSON(value string) json.RawMessage {
	return json.RawMessage(value)
}
