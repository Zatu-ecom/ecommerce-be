package util

import (
	"fmt"
	"time"
)

type ReportQueryFilter struct {
	TimeRange   string `form:"time_range"`
	LastXAmount int    `form:"last_x_amount"`
	LastXUnit   string `form:"last_x_unit"` // hour, day, week, month, quarter, year
	StartDate   string `form:"start_date"`  // expected ISO8601
	EndDate     string `form:"end_date"`    // expected ISO8601
	Interval    string `form:"interval"`    // default "day"
	Compare     bool   `form:"compare,default=true"`
}

// ReportPeriods holds the resolved time ranges used for reporting.
// It includes the current period being queried and the preceding period
// used to calculate comparison trends.
type ReportPeriods struct {
	CurrStart time.Time
	CurrEnd   time.Time
	PrevStart time.Time
	PrevEnd   time.Time
}

// CalculatePeriods resolves the start and end dates for both the current
// reporting period and the previous comparison period based on the given filter.
//
// How it works:
// 1. First, it determines the current period (CurrStart, CurrEnd) using calculateCurrentPeriod.
//   - For predefined ranges (e.g., "today", "this_month"), it aligns to the beginning dates.
//   - For "last_x" or "custom", it resolves exact timestamps.
//
// 2. Second, it determines the previous period (PrevStart, PrevEnd) using calculatePreviousPeriod.
//   - If the Compare flag is false, the previous period simply mirrors the current period.
//   - If true, it usually shifts backwards by the exact duration of the current period,
//     or matches calendar logic (e.g. going from "this_month" to the exact bounds of last month).
func CalculatePeriods(
	filter ReportQueryFilter,
) (ReportPeriods, error) {
	now := time.Now()

	currStart, currEnd, err := calculateCurrentPeriod(filter, now)
	if err != nil {
		return ReportPeriods{}, err
	}

	prevStart, prevEnd := calculatePreviousPeriod(filter, currStart, currEnd, now)

	return ReportPeriods{
		CurrStart: currStart,
		CurrEnd:   currEnd,
		PrevStart: prevStart,
		PrevEnd:   prevEnd,
	}, nil
}

func calculateCurrentPeriod(filter ReportQueryFilter, now time.Time) (time.Time, time.Time, error) {
	var currStart, currEnd time.Time

	switch filter.TimeRange {
	case "today":
		currStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		// End-of-day. Future hours have no rows, so data is unchanged; labels span the full 24h.
		currEnd = endOfDay(now)
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		currStart = time.Date(
			yesterday.Year(),
			yesterday.Month(),
			yesterday.Day(),
			0,
			0,
			0,
			0,
			yesterday.Location(),
		)
		currEnd = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, -1, now.Location())
	case "this_week":
		daysSinceMonday := int(now.Weekday()) - 1
		if daysSinceMonday < 0 {
			daysSinceMonday = 6 // Sunday
		}
		monday := now.AddDate(0, 0, -daysSinceMonday)
		currStart = time.Date(
			monday.Year(),
			monday.Month(),
			monday.Day(),
			0,
			0,
			0,
			0,
			now.Location(),
		)
		// End of the week (Sunday 23:59:59.999…) so labels cover all 7 days.
		sunday := monday.AddDate(0, 0, 6)
		currEnd = endOfDay(sunday)
	case "this_month":
		currStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		// Last day of the month at 23:59:59.999…; future days resolve to zero rows.
		currEnd = endOfDay(
			time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()),
		)
	case "this_quarter":
		startMonth := getStartOfQuarter(now.Month())
		currStart = time.Date(now.Year(), startMonth, 1, 0, 0, 0, 0, now.Location())
		// End of the quarter (last day of startMonth+2).
		currEnd = endOfDay(
			time.Date(now.Year(), startMonth+3, 0, 0, 0, 0, 0, now.Location()),
		)
	case "this_year":
		currStart = time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())
		// End of the year so label generation produces all 12 months.
		currEnd = endOfDay(
			time.Date(now.Year(), time.December, 31, 0, 0, 0, 0, now.Location()),
		)
	case "last_x":
		return calculateLastXPeriod(filter, now)
	case "custom":
		return calculateCustomPeriod(filter)
	default:
		// Default to exactly exactly 30 days
		currEnd = now
		currStart = now.AddDate(0, 0, -30)
	}

	return currStart, currEnd, nil
}

// LocationToPostgresTZ returns a Postgres-compatible timezone spec for the
// location of the given time. IANA names (e.g. "Asia/Kolkata") are returned
// as-is; anonymous/local zones fall back to a fixed offset like "+05:30"
// which Postgres accepts via `AT TIME ZONE INTERVAL '+05:30'` — we emit it
// as a quoted string since the `AT TIME ZONE '<spec>'` form accepts both
// IANA names and POSIX-style offsets (minus signs flipped).
func LocationToPostgresTZ(t time.Time) string {
	name, offsetSec := t.Zone()

	// Go's time.LoadLocation("Local") returns a zone whose Name() is an
	// IANA string ("Asia/Kolkata") on most platforms, but CI/containers
	// sometimes surface it as just "Local" or an abbreviation like "IST"
	// which Postgres doesn't recognise. Accept only values that look like
	// an IANA-style name (contain a "/" or are "UTC"); otherwise fall back
	// to a numeric offset.
	if name == "UTC" {
		return "UTC"
	}
	for _, r := range name {
		if r == '/' {
			// Looks like "Asia/Kolkata" — trust it.
			return name
		}
	}

	// Postgres `AT TIME ZONE '<numeric-offset>'` uses POSIX sign convention
	// (inverted from ISO). So for an ISO offset of +05:30 we must emit
	// '-05:30' to get the correct conversion. Reference:
	// https://www.postgresql.org/docs/current/datatype-datetime.html#DATATYPE-TIMEZONES
	sign := "-"
	if offsetSec < 0 {
		sign = "+"
		offsetSec = -offsetSec
	}
	h := offsetSec / 3600
	m := (offsetSec % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, h, m)
}

// endOfDay returns the last representable instant of the calendar day
// containing t, preserving its timezone. Used so trend labels cover the full
// calendar bucket while the DB filter `placed_at <= currEnd` still matches
// only existing rows (future hours/days have no data).
func endOfDay(t time.Time) time.Time {
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		23, 59, 59, int(time.Second-time.Nanosecond),
		t.Location(),
	)
}

func getStartOfQuarter(month time.Month) time.Month {
	if month <= 3 {
		return time.January
	} else if month <= 6 {
		return time.April
	} else if month <= 9 {
		return time.July
	}
	return time.October
}

func calculateLastXPeriod(filter ReportQueryFilter, now time.Time) (time.Time, time.Time, error) {
	if filter.LastXAmount <= 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("last_x_amount must be greater than 0")
	}

	currEnd := now
	var currStart time.Time

	switch filter.LastXUnit {
	case "hour":
		currStart = now.Add(-time.Duration(filter.LastXAmount) * time.Hour)
	case "day":
		currStart = now.AddDate(0, 0, -filter.LastXAmount)
	case "week":
		currStart = now.AddDate(0, 0, -filter.LastXAmount*7)
	case "month":
		currStart = now.AddDate(0, -filter.LastXAmount, 0)
	case "quarter":
		currStart = now.AddDate(0, -filter.LastXAmount*3, 0)
	case "year":
		currStart = now.AddDate(-filter.LastXAmount, 0, 0)
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported last_x_unit: %s", filter.LastXUnit)
	}
	return currStart, currEnd, nil
}

func calculateCustomPeriod(filter ReportQueryFilter) (time.Time, time.Time, error) {
	if filter.StartDate == "" || filter.EndDate == "" {
		return time.Time{}, time.Time{}, fmt.Errorf(
			"start_date and end_date are required for custom time_range",
		)
	}
	currStart, err := time.Parse(time.RFC3339, filter.StartDate)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date: %v", err)
	}
	currEnd, err := time.Parse(time.RFC3339, filter.EndDate)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date: %v", err)
	}
	return currStart, currEnd, nil
}

func calculatePreviousPeriod(
	filter ReportQueryFilter,
	currStart, currEnd, now time.Time,
) (time.Time, time.Time) {
	if !filter.Compare {
		return currStart, currEnd
	}

	duration := currEnd.Sub(currStart)
	var prevStart, prevEnd time.Time

	switch filter.TimeRange {
	case "this_month":
		prevEnd = currStart.Add(-time.Nanosecond)
		prevStart = time.Date(prevEnd.Year(), prevEnd.Month(), 1, 0, 0, 0, 0, now.Location())
	case "this_year":
		prevEnd = currStart.Add(-time.Nanosecond)
		prevStart = time.Date(prevEnd.Year(), time.January, 1, 0, 0, 0, 0, now.Location())
	default:
		prevEnd = currStart
		prevStart = prevEnd.Add(-duration)
	}

	return prevStart, prevEnd
}
