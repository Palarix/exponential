package main

import (
	"testing"
	"time"
)

func TestParseDuration_Minutes(t *testing.T) {
	d, err := parseDuration("30m")
	if err != nil {
		t.Fatal(err)
	}
	if d != 30*time.Minute {
		t.Errorf("got %v", d)
	}
}

func TestParseDuration_Hours(t *testing.T) {
	d, err := parseDuration("24h")
	if err != nil {
		t.Fatal(err)
	}
	if d != 24*time.Hour {
		t.Errorf("got %v", d)
	}
}

func TestParseDuration_Days(t *testing.T) {
	d, err := parseDuration("7d")
	if err != nil {
		t.Fatal(err)
	}
	if d != 7*24*time.Hour {
		t.Errorf("got %v", d)
	}
}

func TestParseDuration_Weeks(t *testing.T) {
	d, err := parseDuration("2w")
	if err != nil {
		t.Fatal(err)
	}
	if d != 2*7*24*time.Hour {
		t.Errorf("got %v", d)
	}
}

func TestParseDuration_Empty(t *testing.T) {
	_, err := parseDuration("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestParseDuration_Invalid(t *testing.T) {
	_, err := parseDuration("abc")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestParseDuration_InvalidNumber(t *testing.T) {
	_, err := parseDuration("xd")
	if err == nil {
		t.Error("expected error for non-numeric days")
	}
}
