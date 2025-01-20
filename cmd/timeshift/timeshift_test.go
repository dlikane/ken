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

func TestGetConnection(t *testing.T) {
	_, err := getConnection()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestProcessCards(t *testing.T) {
	timeCards := &TimeCards{
		TimeCards: []TimeCard{
			{
				EmployeeName: "John Doe",
				Shift: []Shift{
					{
						ShiftHours: []ShiftHours{
							{
								StartTime:  xmlTime(time.Date(2023, 4, 3, 9, 0, 0, 0, time.UTC)),
								FinishTime: xmlTime(time.Date(2023, 4, 3, 17, 0, 0, 0, time.UTC)),
							},
						},
					},
				},
			},
		},
	}
	err := processCards(timeCards)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestProcessAllowances(t *testing.T) {
	timeCards := &TimeCards{
		TimeCards: []TimeCard{
			{
				EmployeeName: "John Doe",
				Shift: []Shift{
					{
						ShiftHours: []ShiftHours{
							{
								StartTime:  xmlTime(time.Date(2023, 4, 3, 9, 0, 0, 0, time.UTC)),
								FinishTime: xmlTime(time.Date(2023, 4, 3, 17, 0, 0, 0, time.UTC)),
							},
						},
					},
				},
			},
		},
	}
	_, err := processAllowances(timeCards)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestProcessLeaves(t *testing.T) {
	fromDate := xmlTime(time.Date(2023, 4, 3, 0, 0, 0, 0, time.UTC))
	toDate := xmlTime(time.Date(2023, 4, 3, 0, 0, 0, 0, time.UTC))
	timeCards := &TimeCards{
		TimeCards: []TimeCard{
			{
				EmployeeName: "John Doe",
				Leave: []Leave{
					{
						Type:     "Annual",
						Hours:    7.6,
						FromDate: &fromDate,
						ToDate:   &toDate,
					},
				},
			},
		},
	}
	leaveData := LeaveData{
		"John Doe": {
			"Mon": 7.6,
		},
	}
	holidayData := HolidayData{
		time.Date(2023, 4, 3, 0, 0, 0, 0, time.UTC): time.Date(2023, 4, 3, 0, 0, 0, 0, time.UTC),
	}
	err := processLeaves(timeCards, leaveData, holidayData)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
