package config

import (
	"testing"
)

func TestValidateEstimate_Valid(t *testing.T) {
	cases := []struct{ system string; value int }{
		{"fibonacci", 1}, {"fibonacci", 2}, {"fibonacci", 3}, {"fibonacci", 5}, {"fibonacci", 8},
		{"exponential", 1}, {"exponential", 16},
		{"linear", 1}, {"linear", 5},
		{"shirt", 1}, {"shirt", 8},
	}
	for _, tc := range cases {
		if err := ValidateEstimate(tc.system, tc.value); err != nil {
			t.Errorf("ValidateEstimate(%q, %d) = %v, want nil", tc.system, tc.value, err)
		}
	}
}

func TestValidateEstimate_Invalid(t *testing.T) {
	cases := []struct{ system string; value int }{
		{"fibonacci", 4},
		{"fibonacci", 10},
		{"exponential", 3},
		{"linear", 6},
	}
	for _, tc := range cases {
		if err := ValidateEstimate(tc.system, tc.value); err == nil {
			t.Errorf("ValidateEstimate(%q, %d) should error", tc.system, tc.value)
		}
	}
}

func TestValidateEstimate_Zero(t *testing.T) {
	if err := ValidateEstimate("fibonacci", 0); err == nil {
		t.Error("zero should not be allowed")
	}
}

func TestValidateEstimate_UnknownSystem(t *testing.T) {
	if err := ValidateEstimate("bogus", 1); err == nil {
		t.Error("unknown system should error")
	}
}

func TestEstimateDisplayValue_Shirt(t *testing.T) {
	cases := map[int]string{1: "XS", 2: "S", 3: "M", 5: "L", 8: "XL"}
	for val, want := range cases {
		if got := EstimateDisplayValue("shirt", val); got != want {
			t.Errorf("EstimateDisplayValue(shirt, %d) = %q, want %q", val, got, want)
		}
	}
}

func TestEstimateDisplayValue_NonShirt(t *testing.T) {
	if got := EstimateDisplayValue("fibonacci", 5); got != "5" {
		t.Errorf("EstimateDisplayValue(fibonacci, 5) = %q, want \"5\"", got)
	}
}

func TestEstimateDisplayValue_ShirtUnknownValue(t *testing.T) {
	if got := EstimateDisplayValue("shirt", 99); got != "99" {
		t.Errorf("unknown shirt value should fallback to number, got %q", got)
	}
}

func TestParseEstimateInput_Numeric(t *testing.T) {
	val, err := ParseEstimateInput("fibonacci", "5")
	if err != nil || val != 5 {
		t.Errorf("ParseEstimateInput(fibonacci, \"5\") = (%d, %v)", val, err)
	}
}

func TestParseEstimateInput_ShirtLabel(t *testing.T) {
	cases := map[string]int{"XS": 1, "xs": 1, "S": 2, "m": 3, "L": 5, "xl": 8}
	for input, want := range cases {
		val, err := ParseEstimateInput("shirt", input)
		if err != nil || val != want {
			t.Errorf("ParseEstimateInput(shirt, %q) = (%d, %v), want %d", input, val, err, want)
		}
	}
}

func TestParseEstimateInput_ShirtNumeric(t *testing.T) {
	val, err := ParseEstimateInput("shirt", "3")
	if err != nil || val != 3 {
		t.Errorf("shirt should accept numeric too: (%d, %v)", val, err)
	}
}

func TestParseEstimateInput_Invalid(t *testing.T) {
	_, err := ParseEstimateInput("fibonacci", "abc")
	if err == nil {
		t.Error("non-numeric non-shirt input should error")
	}
}
