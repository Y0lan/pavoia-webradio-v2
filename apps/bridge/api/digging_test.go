package api

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestComputeStreaks_Empty(t *testing.T) {
	s := computeStreaks(nil)
	if s.Current != 0 || s.Longest != 0 {
		t.Errorf("expected zeros for empty, got current=%d longest=%d", s.Current, s.Longest)
	}
}

func TestComputeStreaks_SingleDay(t *testing.T) {
	s := computeStreaks([]time.Time{date("2026-03-20")})
	if s.Longest != 1 {
		t.Errorf("expected longest=1, got %d", s.Longest)
	}
	if s.LastDate != "2026-03-20" {
		t.Errorf("expected last_dig_date=2026-03-20, got %s", s.LastDate)
	}
}

func TestComputeStreaks_ConsecutiveDays(t *testing.T) {
	dates := []time.Time{
		date("2026-03-20"),
		date("2026-03-21"),
		date("2026-03-22"),
		date("2026-03-23"),
	}
	s := computeStreaks(dates)
	if s.Longest != 4 {
		t.Errorf("expected longest=4, got %d", s.Longest)
	}
}

func TestComputeStreaks_WithGap(t *testing.T) {
	dates := []time.Time{
		date("2026-03-10"),
		date("2026-03-11"),
		date("2026-03-12"),
		// gap
		date("2026-03-15"),
		date("2026-03-16"),
	}
	s := computeStreaks(dates)
	if s.Longest != 3 {
		t.Errorf("expected longest=3, got %d", s.Longest)
	}
}

func TestComputeStreaks_MultipleStreaks(t *testing.T) {
	dates := []time.Time{
		date("2026-01-01"),
		date("2026-01-02"),
		// gap
		date("2026-02-01"),
		date("2026-02-02"),
		date("2026-02-03"),
		date("2026-02-04"),
		date("2026-02-05"),
		// gap
		date("2026-03-01"),
	}
	s := computeStreaks(dates)
	if s.Longest != 5 {
		t.Errorf("expected longest=5, got %d", s.Longest)
	}
}

func TestComputeBestWeek_Empty(t *testing.T) {
	if v := computeBestWeek(nil); v != 0 {
		t.Errorf("expected 0 for empty, got %d", v)
	}
}

func TestComputeBestWeek_SingleDay(t *testing.T) {
	v := computeBestWeek([]dayCount{
		{date: date("2026-03-20"), count: 5},
	})
	if v != 5 {
		t.Errorf("expected 5, got %d", v)
	}
}

func TestComputeBestWeek_SpreadOut(t *testing.T) {
	v := computeBestWeek([]dayCount{
		{date: date("2026-03-01"), count: 3},
		{date: date("2026-03-03"), count: 4},
		{date: date("2026-03-05"), count: 2},
		// gap > 7 days
		{date: date("2026-03-20"), count: 10},
	})
	// Best 7-day window: either [Mar 1-7] = 3+4+2 = 9, or [Mar 20-26] = 10
	if v != 10 {
		t.Errorf("expected 10, got %d", v)
	}
}

func TestComputeBestWeek_DenseWeek(t *testing.T) {
	v := computeBestWeek([]dayCount{
		{date: date("2026-03-01"), count: 2},
		{date: date("2026-03-02"), count: 3},
		{date: date("2026-03-03"), count: 5},
		{date: date("2026-03-04"), count: 1},
		{date: date("2026-03-05"), count: 4},
		{date: date("2026-03-06"), count: 2},
		{date: date("2026-03-07"), count: 3},
	})
	// All within 7 days: 2+3+5+1+4+2+3 = 20
	if v != 20 {
		t.Errorf("expected 20, got %d", v)
	}
}
