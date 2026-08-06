// Package session validates the repository-owned NYSE schedule artifact and
// resolves immutable scanner-session facts for an explicit trading date.
package session

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	_ "time/tzdata"
)

const (
	dateLayout               = "2006-01-02"
	approvedScheduleSchema   = "nyse-trading-days-v1"
	approvedProvenanceSchema = "nyse-schedule-provenance-v1"
	approvedSourceURL        = "https://www.nyse.com/trade/hours-calendars"
	approvedArtifactPath     = "internal/session/nyse_trading_days.json"
	approvedArtifactSHA256   = "3fb957d2c41f53e64883699d6324f378fd896755fa299a4bd209d1e60d190784"
	approvedProvenanceSHA256 = "afc10e63f418d7578e8ec3167cac63d1ecd22b1166b3036182d48e9508b86345"
)

//go:embed nyse_trading_days.json
var embeddedScheduleArtifact []byte

//go:embed nyse_schedule_provenance.json
var embeddedScheduleProvenance []byte

type scheduleRow struct {
	Date           string `json:"date"`
	RegularCloseET string `json:"regular_close_et"`
}

type scheduleDocument struct {
	SchemaVersion       string        `json:"schema_version"`
	ScheduleVersion     string        `json:"schedule_version"`
	CoverageStart       string        `json:"coverage_start"`
	CoverageEnd         string        `json:"coverage_end"`
	TradingDays         []scheduleRow `json:"trading_days"`
	ExceptionalClosures []string      `json:"exceptional_closures"`
}

type provenanceDocument struct {
	SchemaVersion            string   `json:"schema_version"`
	ScheduleVersion          string   `json:"schedule_version"`
	SourceURL                string   `json:"source_url"`
	RetrievedAndReviewedDate string   `json:"retrieved_and_reviewed_date"`
	CoverageStart            string   `json:"coverage_start"`
	CoverageEnd              string   `json:"coverage_end"`
	TradingDayRowCount       int      `json:"trading_day_row_count"`
	EarlyCloseDates          []string `json:"early_close_dates"`
	ExceptionalClosures      []string `json:"exceptional_closures"`
	ExceptionalClosureReview string   `json:"exceptional_closure_review"`
	Artifact                 string   `json:"artifact"`
	ArtifactSHA256           string   `json:"artifact_sha256"`
}

type scheduleEntry struct {
	date        string
	closeOffset time.Duration
}

// Schedule is an immutable, validated view of the embedded exchange schedule.
// It exposes no mutable representation of its trading-day rows.
type Schedule struct {
	schemaVersion   string
	scheduleVersion string
	artifactSHA256  string
	coverageStart   string
	coverageEnd     string
	location        *time.Location
	entries         []scheduleEntry
}

// Facts are the immutable schedule facts for one explicitly requested trading
// date. Schedule time is configuration only; these values make no market-data
// arrival, currentness, readiness, or watermark claim.
type Facts struct {
	TradingDate            string
	SessionStart           time.Time
	SessionEnd             time.Time
	PriorSessionDate       string
	PriorRegularClose      time.Time
	ScheduleSchema         string
	ScheduleVersion        string
	ScheduleArtifactSHA256 string
}

// Contains reports membership in the exact half-open scanner session [S,E).
func (f Facts) Contains(instant time.Time) bool {
	return !instant.Before(f.SessionStart) && instant.Before(f.SessionEnd)
}

// Load validates and returns the repository-owned schedule and provenance.
func Load() (*Schedule, error) {
	return loadSchedule(
		embeddedScheduleArtifact,
		embeddedScheduleProvenance,
		approvedArtifactSHA256,
		approvedProvenanceSHA256,
		time.LoadLocation,
	)
}

// ForTradingDate validates an exchange-local civil date and resolves its
// scanner bounds and immediately preceding completed regular session.
func (s *Schedule) ForTradingDate(date string) (Facts, error) {
	requested, err := parseDate(date, s.location)
	if err != nil {
		return Facts{}, fmt.Errorf("invalid trading date %q: %w", date, err)
	}
	if date < s.coverageStart || date > s.coverageEnd {
		return Facts{}, fmt.Errorf("trading date %s is outside schedule coverage %s..%s", date, s.coverageStart, s.coverageEnd)
	}

	index, found := slices.BinarySearchFunc(s.entries, date, func(entry scheduleEntry, target string) int {
		if entry.date < target {
			return -1
		}
		if entry.date > target {
			return 1
		}
		return 0
	})
	if !found {
		return Facts{}, fmt.Errorf("trading date %s is not declared by the exchange schedule", date)
	}
	if index == 0 {
		return Facts{}, fmt.Errorf("prior completed session for %s is outside schedule coverage", date)
	}

	prior := s.entries[index-1]
	start := time.Date(requested.Year(), requested.Month(), requested.Day(), 4, 0, 0, 0, s.location)
	end := time.Date(requested.Year(), requested.Month(), requested.Day(), 20, 0, 0, 0, s.location)
	priorDate, err := parseDate(prior.date, s.location)
	if err != nil {
		return Facts{}, fmt.Errorf("validated prior session date %s: %w", prior.date, err)
	}
	priorClose := time.Date(
		priorDate.Year(), priorDate.Month(), priorDate.Day(),
		int(prior.closeOffset/time.Hour), 0, 0, 0, s.location,
	)

	return Facts{
		TradingDate:            date,
		SessionStart:           start.UTC(),
		SessionEnd:             end.UTC(),
		PriorSessionDate:       prior.date,
		PriorRegularClose:      priorClose.UTC(),
		ScheduleSchema:         s.schemaVersion,
		ScheduleVersion:        s.scheduleVersion,
		ScheduleArtifactSHA256: s.artifactSHA256,
	}, nil
}

func loadSchedule(
	artifact []byte,
	provenance []byte,
	expectedArtifactSHA256 string,
	expectedProvenanceSHA256 string,
	loadLocation func(string) (*time.Location, error),
) (*Schedule, error) {
	artifactDigest := digest(artifact)
	if artifactDigest != expectedArtifactSHA256 {
		return nil, errors.New("schedule artifact checksum mismatch")
	}
	if digest(provenance) != expectedProvenanceSHA256 {
		return nil, errors.New("schedule provenance checksum mismatch")
	}

	var document scheduleDocument
	if err := decodeStrict(artifact, &document); err != nil {
		return nil, fmt.Errorf("decode schedule artifact: %w", err)
	}
	var source provenanceDocument
	if err := decodeStrict(provenance, &source); err != nil {
		return nil, fmt.Errorf("decode schedule provenance: %w", err)
	}

	location, err := loadLocation("America/New_York")
	if err != nil {
		return nil, fmt.Errorf("load America/New_York timezone: %w", err)
	}
	coverageStart, err := parseDate(document.CoverageStart, location)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule coverage start: %w", err)
	}
	coverageEnd, err := parseDate(document.CoverageEnd, location)
	if err != nil || coverageEnd.Before(coverageStart) {
		return nil, errors.New("invalid schedule coverage end")
	}
	if document.SchemaVersion != approvedScheduleSchema || document.ScheduleVersion == "" || len(document.TradingDays) == 0 {
		return nil, errors.New("invalid schedule metadata")
	}

	entries := make([]scheduleEntry, 0, len(document.TradingDays))
	earlyCloseDates := make([]string, 0)
	previous := ""
	for _, row := range document.TradingDays {
		if row.Date <= previous {
			return nil, errors.New("schedule trading dates are not strictly sorted and unique")
		}
		if _, err := parseDate(row.Date, location); err != nil || row.Date < document.CoverageStart || row.Date > document.CoverageEnd {
			return nil, fmt.Errorf("invalid or uncovered schedule date %q", row.Date)
		}
		closeOffset, err := parseRegularClose(row.RegularCloseET)
		if err != nil {
			return nil, fmt.Errorf("invalid regular close for %s: %w", row.Date, err)
		}
		if closeOffset == 13*time.Hour {
			earlyCloseDates = append(earlyCloseDates, row.Date)
		}
		entries = append(entries, scheduleEntry{date: row.Date, closeOffset: closeOffset})
		previous = row.Date
	}
	if err := validateSortedDates(document.ExceptionalClosures, location); err != nil {
		return nil, fmt.Errorf("invalid exceptional closures: %w", err)
	}
	for _, closed := range document.ExceptionalClosures {
		if closed < document.CoverageStart || closed > document.CoverageEnd {
			return nil, fmt.Errorf("exceptional closure %s is outside coverage", closed)
		}
		if _, found := slices.BinarySearchFunc(entries, closed, func(entry scheduleEntry, target string) int {
			return compare(entry.date, target)
		}); found {
			return nil, fmt.Errorf("exceptional closure %s is also a trading date", closed)
		}
	}

	if err := validateProvenance(source, document, artifactDigest, earlyCloseDates, location); err != nil {
		return nil, err
	}
	return &Schedule{
		schemaVersion:   document.SchemaVersion,
		scheduleVersion: document.ScheduleVersion,
		artifactSHA256:  artifactDigest,
		coverageStart:   document.CoverageStart,
		coverageEnd:     document.CoverageEnd,
		location:        location,
		entries:         entries,
	}, nil
}

func validateProvenance(source provenanceDocument, document scheduleDocument, artifactDigest string, earlyCloseDates []string, location *time.Location) error {
	if source.SchemaVersion != approvedProvenanceSchema ||
		source.ScheduleVersion != document.ScheduleVersion ||
		source.SourceURL != approvedSourceURL ||
		source.CoverageStart != document.CoverageStart ||
		source.CoverageEnd != document.CoverageEnd ||
		source.TradingDayRowCount != len(document.TradingDays) ||
		source.Artifact != approvedArtifactPath ||
		source.ArtifactSHA256 != artifactDigest ||
		source.ExceptionalClosureReview == "" {
		return errors.New("invalid schedule provenance metadata")
	}
	if _, err := parseDate(source.RetrievedAndReviewedDate, location); err != nil {
		return errors.New("invalid schedule provenance review date")
	}
	if err := validateSortedDates(source.EarlyCloseDates, location); err != nil || !slices.Equal(source.EarlyCloseDates, earlyCloseDates) {
		return errors.New("schedule provenance early closes do not match artifact")
	}
	if err := validateSortedDates(source.ExceptionalClosures, location); err != nil || !slices.Equal(source.ExceptionalClosures, document.ExceptionalClosures) {
		return errors.New("schedule provenance exceptional closures do not match artifact")
	}
	return nil
}

func validateSortedDates(dates []string, location *time.Location) error {
	previous := ""
	for _, date := range dates {
		if date <= previous {
			return errors.New("dates are not strictly sorted and unique")
		}
		if _, err := parseDate(date, location); err != nil {
			return err
		}
		previous = date
	}
	return nil
}

func parseDate(value string, location *time.Location) (time.Time, error) {
	if len(value) != len(dateLayout) {
		return time.Time{}, errors.New("date must use YYYY-MM-DD")
	}
	date, err := time.ParseInLocation(dateLayout, value, location)
	if err != nil || date.Format(dateLayout) != value {
		return time.Time{}, errors.New("date must be a valid YYYY-MM-DD civil date")
	}
	return date, nil
}

func parseRegularClose(value string) (time.Duration, error) {
	closeTime, err := time.Parse("15:04:05", value)
	if err != nil || closeTime.Format("15:04:05") != value {
		return 0, errors.New("close must use HH:MM:SS")
	}
	offset := time.Duration(closeTime.Hour())*time.Hour +
		time.Duration(closeTime.Minute())*time.Minute +
		time.Duration(closeTime.Second())*time.Second
	if offset != 13*time.Hour && offset != 16*time.Hour {
		return 0, errors.New("supported closes are 13:00:00 and 16:00:00 America/New_York")
	}
	return offset, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("trailing JSON value")
	}
	return err
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func compare(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
