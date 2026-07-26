package services

import "testing"

func TestParseSearchQueryAfter(t *testing.T) {
	got := ParseSearchQuery("skrillex :after:2026")
	if got.Text != "skrillex" || got.YearFrom != 2026 || got.YearTo != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseSearchQueryBeforeAndRange(t *testing.T) {
	got := ParseSearchQuery("dualipa :after:2018 :before:2022")
	if got.Text != "dualipa" || got.YearFrom != 2018 || got.YearTo != 2022 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseSearchQueryAliasesAndSwap(t *testing.T) {
	got := ParseSearchQuery("nero :from:2024 :until:2020")
	if got.Text != "nero" || got.YearFrom != 2020 || got.YearTo != 2024 {
		t.Fatalf("swapped range want 2020–2024, got %+v", got)
	}
}

func TestParseSearchQueryNoFilters(t *testing.T) {
	got := ParseSearchQuery("hello world")
	if got.Text != "hello world" || got.HasYearFilter() {
		t.Fatalf("got %+v", got)
	}
}
