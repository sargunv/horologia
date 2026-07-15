package recipecmd

import (
	"reflect"
	"testing"
)

func TestParseQuantity(t *testing.T) {
	tests := []struct {
		input       string
		quantity    *float64
		quantityMax *float64
		unit        string
	}{
		{input: "500 g", quantity: float64Pointer(500), unit: "g"},
		{input: "1 1/2 cups", quantity: float64Pointer(1.5), unit: "cups"},
		{input: "1/4 tsp", quantity: float64Pointer(0.25), unit: "tsp"},
		{input: "1–2 tbsp", quantity: float64Pointer(1), quantityMax: float64Pointer(2), unit: "tbsp"},
		{input: "to taste", unit: "to taste"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := parseQuantity(test.input)
			if err != nil {
				t.Fatalf("parse quantity: %v", err)
			}
			if !reflect.DeepEqual(got.quantity, test.quantity) || !reflect.DeepEqual(got.quantityMax, test.quantityMax) || got.unit != test.unit {
				t.Fatalf("parseQuantity(%q) = %#v, want quantity=%v max=%v unit=%q", test.input, got, test.quantity, test.quantityMax, test.unit)
			}
		})
	}
}

func TestParseQuantityRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "0 g", "1/0 cup", "2–1 tbsp", "1/ cup"} {
		t.Run(input, func(t *testing.T) {
			if _, err := parseQuantity(input); err == nil {
				t.Fatalf("parseQuantity(%q) unexpectedly succeeded", input)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	for input, want := range map[string]int32{
		"45":        45,
		"45m":       45,
		"1h":        60,
		"1h 15m":    75,
		"1.5 hours": 90,
	} {
		t.Run(input, func(t *testing.T) {
			got, err := parseDuration(input)
			if err != nil {
				t.Fatalf("parse duration: %v", err)
			}
			if got != want {
				t.Fatalf("parseDuration(%q) = %d, want %d", input, got, want)
			}
		})
	}
}

func float64Pointer(value float64) *float64 {
	return &value
}
