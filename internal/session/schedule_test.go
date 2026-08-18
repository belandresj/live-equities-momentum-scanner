package session

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestScheduleArtifact(t *testing.T) {
	schedule, err := Load()
	if err != nil {
		t.Fatalf("load reviewed artifact: %v", err)
	}
	facts, err := schedule.ForTradingDate("2026-08-05")
	if err != nil {
		t.Fatalf("select supported date: %v", err)
	}
	if facts.ScheduleSchema != approvedScheduleSchema ||
		facts.ScheduleVersion != "nyse-2026-2027-reviewed-2026-07-29" ||
		facts.ScheduleArtifactSHA256 != approvedArtifactSHA256 {
		t.Fatalf("unexpected artifact identity: %+v", facts)
	}

	tests := []struct {
		name               string
		artifact           []byte
		provenance         []byte
		expectedArtifact   string
		expectedProvenance string
		wantError          string
	}{
		{
			name:               "artifact checksum mismatch",
			artifact:           append(append([]byte(nil), embeddedScheduleArtifact...), '\n'),
			provenance:         embeddedScheduleProvenance,
			expectedArtifact:   approvedArtifactSHA256,
			expectedProvenance: approvedProvenanceSHA256,
			wantError:          "artifact checksum mismatch",
		},
		{
			name: "schema mismatch",
			artifact: rewriteSchedule(t, func(document *scheduleDocument) {
				document.SchemaVersion = "nyse-trading-days-v2"
			}),
			provenance:         embeddedScheduleProvenance,
			expectedProvenance: approvedProvenanceSHA256,
			wantError:          "invalid schedule metadata",
		},
		{
			name: "unsorted dates",
			artifact: rewriteSchedule(t, func(document *scheduleDocument) {
				document.TradingDays[0], document.TradingDays[1] = document.TradingDays[1], document.TradingDays[0]
			}),
			provenance:         embeddedScheduleProvenance,
			expectedProvenance: approvedProvenanceSHA256,
			wantError:          "not strictly sorted and unique",
		},
		{
			name: "unsupported close",
			artifact: rewriteSchedule(t, func(document *scheduleDocument) {
				document.TradingDays[0].RegularCloseET = "15:00:00"
			}),
			provenance:         embeddedScheduleProvenance,
			expectedProvenance: approvedProvenanceSHA256,
			wantError:          "supported closes",
		},
		{
			name: "invalid coverage",
			artifact: rewriteSchedule(t, func(document *scheduleDocument) {
				document.CoverageEnd = "2025-12-31"
			}),
			provenance:         embeddedScheduleProvenance,
			expectedProvenance: approvedProvenanceSHA256,
			wantError:          "invalid schedule coverage end",
		},
		{
			name:       "invalid official-source provenance",
			artifact:   embeddedScheduleArtifact,
			provenance: rewriteProvenance(t, func(document *provenanceDocument) { document.SourceURL = "https://example.invalid" }),
			wantError:  "invalid schedule provenance metadata",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectedArtifact == "" {
				test.expectedArtifact = digest(test.artifact)
			}
			if test.expectedProvenance == "" {
				test.expectedProvenance = digest(test.provenance)
			}
			_, err := loadSchedule(test.artifact, test.provenance, test.expectedArtifact, test.expectedProvenance, time.LoadLocation)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want containing %q", err, test.wantError)
			}
		})
	}

	if _, err := schedule.ForTradingDate("2028-01-03"); err == nil || !strings.Contains(err.Error(), "outside schedule coverage") {
		t.Fatalf("unsupported coverage error = %v", err)
	}
	if _, err := loadSchedule(
		embeddedScheduleArtifact,
		embeddedScheduleProvenance,
		approvedArtifactSHA256,
		approvedProvenanceSHA256,
		func(string) (*time.Location, error) { return nil, errors.New("unavailable") },
	); err == nil || !strings.Contains(err.Error(), "timezone") {
		t.Fatalf("timezone failure = %v", err)
	}
}

func TestSessionSelection(t *testing.T) {
	schedule, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		tradingDate string
		priorDate   string
		start       string
		end         string
		priorClose  string
	}{
		{"ordinary trading date", "2026-07-29", "2026-07-28", "2026-07-29T08:00:00Z", "2026-07-30T00:00:00Z", "2026-07-28T20:00:00Z"},
		{"weekend predecessor", "2026-08-03", "2026-07-31", "2026-08-03T08:00:00Z", "2026-08-04T00:00:00Z", "2026-07-31T20:00:00Z"},
		{"exchange holiday predecessor", "2026-07-06", "2026-07-02", "2026-07-06T08:00:00Z", "2026-07-07T00:00:00Z", "2026-07-02T20:00:00Z"},
		{"early-close predecessor does not shorten scanner session", "2026-11-30", "2026-11-27", "2026-11-30T09:00:00Z", "2026-12-01T01:00:00Z", "2026-11-27T18:00:00Z"},
		{"before daylight saving transition", "2026-03-06", "2026-03-05", "2026-03-06T09:00:00Z", "2026-03-07T01:00:00Z", "2026-03-05T21:00:00Z"},
		{"after daylight saving transition", "2026-03-09", "2026-03-06", "2026-03-09T08:00:00Z", "2026-03-10T00:00:00Z", "2026-03-06T21:00:00Z"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			facts, err := schedule.ForTradingDate(test.tradingDate)
			if err != nil {
				t.Fatal(err)
			}
			if facts.TradingDate != test.tradingDate || facts.PriorSessionDate != test.priorDate {
				t.Fatalf("dates = %s/%s, want %s/%s", facts.TradingDate, facts.PriorSessionDate, test.tradingDate, test.priorDate)
			}
			assertTime(t, "session start", facts.SessionStart, test.start)
			assertTime(t, "session end", facts.SessionEnd, test.end)
			assertTime(t, "prior regular close", facts.PriorRegularClose, test.priorClose)
		})
	}

	facts, err := schedule.ForTradingDate("2026-03-09")
	if err != nil {
		t.Fatal(err)
	}
	if !facts.Contains(facts.SessionStart) {
		t.Fatal("04:00 session start must be included")
	}
	finalSecond := facts.SessionEnd.Add(-time.Second)
	if !facts.Contains(finalSecond) || !facts.Contains(facts.SessionEnd.Add(-time.Nanosecond)) {
		t.Fatal("final eligible second [E-1s,E) must be included")
	}
	if facts.Contains(facts.SessionEnd) {
		t.Fatal("20:00 session end must be excluded")
	}
	for _, invalidDate := range []string{"2026-08-02", "2026-02-30"} {
		if _, err := schedule.ForTradingDate(invalidDate); err == nil {
			t.Fatalf("expected explicit rejection for requested date %s", invalidDate)
		}
	}
}

func TestTradingDateOnOrAfterUsesDeclaredSchedule(t *testing.T) {
	schedule, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ input, want string }{
		{"2026-08-17", "2026-08-17"},
		{"2026-08-22", "2026-08-24"},
		{"2026-09-07", "2026-09-08"},
	} {
		got, err := schedule.TradingDateOnOrAfter(test.input)
		if err != nil || got != test.want {
			t.Fatalf("TradingDateOnOrAfter(%s) = %q, %v; want %s", test.input, got, err, test.want)
		}
	}
	for _, input := range []string{"2026-02-30", "2028-01-01"} {
		if _, err := schedule.TradingDateOnOrAfter(input); err == nil {
			t.Fatalf("TradingDateOnOrAfter(%s) unexpectedly succeeded", input)
		}
	}
}

func rewriteSchedule(t *testing.T, mutate func(*scheduleDocument)) []byte {
	t.Helper()
	var document scheduleDocument
	if err := json.Unmarshal(embeddedScheduleArtifact, &document); err != nil {
		t.Fatal(err)
	}
	mutate(&document)
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func rewriteProvenance(t *testing.T, mutate func(*provenanceDocument)) []byte {
	t.Helper()
	var document provenanceDocument
	if err := json.Unmarshal(embeddedScheduleProvenance, &document); err != nil {
		t.Fatal(err)
	}
	mutate(&document)
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func assertTime(t *testing.T, name string, got time.Time, wantText string) {
	t.Helper()
	want, err := time.Parse(time.RFC3339Nano, wantText)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Fatalf("%s = %s (%s), want %s UTC", name, got, got.Location(), want)
	}
}
