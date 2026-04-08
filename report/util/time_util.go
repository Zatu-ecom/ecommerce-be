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
//    - For predefined ranges (e.g., "today", "this_month"), it aligns to the beginning dates.
//    - For "last_x" or "custom", it resolves exact timestamps.
// 2. Second, it determines the previous period (PrevStart, PrevEnd) using calculatePreviousPeriod.
//    - If the Compare flag is false, the previous period simply mirrors the current period.
//    - If true, it usually shifts backwards by the exact duration of the current period,
//      or matches calendar logic (e.g. going from "this_month" to the exact bounds of last month).
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
		currEnd = now
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
		currEnd = now
	case "this_month":
		currStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		currEnd = now
	case "this_quarter":
		startMonth := getStartOfQuarter(now.Month())
		currStart = time.Date(now.Year(), startMonth, 1, 0, 0, 0, 0, now.Location())
		currEnd = now
	case "this_year":
		currStart = time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())
		currEnd = now
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
