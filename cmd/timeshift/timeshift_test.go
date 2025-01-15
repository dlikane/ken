package main

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	fromDate, toDate, err := parseDate("20230416")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedFromDate := time.Date(2023, 4, 3, 0, 0, 0, 0, time.UTC)
	expectedToDate := time.Date(2023, 4, 16, 0, 0, 0, 0, time.UTC)

	if !fromDate.Equal(expectedFromDate) {
		t.Errorf("Expected fromDate %v, got %v", expectedFromDate, fromDate)
	}

	if !toDate.Equal(expectedToDate) {
		t.Errorf("Expected toDate %v, got %v", expectedToDate, toDate)
	}
}

func TestSortName(t *testing.T) {
	name := "John Doe"
	sortedName := sortName(name)
	expectedName := "Doe John"

	if sortedName != expectedName {
		t.Errorf("Expected sorted name %v, got %v", expectedName, sortedName)
	}
}

func TestMaxDate(t *testing.T) {
	date1 := xmlTime(time.Date(2023, 4, 3, 0, 0, 0, 0, time.UTC))
	date2 := time.Date(2023, 4, 16, 0, 0, 0, 0, time.UTC)
	maxDate := maxDate(&date1, date2)

	expectedDate := xmlTime(date2)
	if *maxDate != expectedDate {
		t.Errorf("Expected max date %v, got %v", expectedDate, *maxDate)
	}
}

func TestMinDate(t *testing.T) {
	date1 := xmlTime(time.Date(2023, 4, 3, 0, 0, 0, 0, time.UTC))
	date2 := time.Date(2023, 4, 16, 0, 0, 0, 0, time.UTC)
	minDate := minDate(&date1, date2)

	expectedDate := xmlTime(time.Date(2023, 4, 3, 0, 0, 0, 0, time.UTC))
	if *minDate != expectedDate {
		t.Errorf("Expected min date %v, got %v", expectedDate, *minDate)
	}
}
