package environment

import (
	"encoding/json"
	"testing"
)

func TestVariables_Get(t *testing.T) {
	// Simulate the values the Panel typically sends via JSON.
	v := Variables{}
	if err := json.Unmarshal([]byte(`{
		"nil_value": null,
		"string": "hello",
		"bool": true,
		"float_int": 42,
		"float": 3.14,
		"nested": {"key": "value"},
		"array": [1, 2, 3]
	}`), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	cases := []struct {
		key  string
		want string
	}{
		{"missing", ""},
		{"nil_value", ""},
		{"string", "hello"},
		{"bool", "true"},
		{"float_int", "42"},
		{"float", "3.14"},
		{"nested", ""},
		{"array", ""},
	}

	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			got := v.Get(c.key)
			if got != c.want {
				t.Errorf("Get(%q) = %q, want %q", c.key, got, c.want)
			}
		})
	}
}

func TestVariables_Get_scalars(t *testing.T) {
	v := Variables{
		"int":     int(1),
		"int8":    int8(2),
		"int32":   int32(3),
		"int64":   int64(4),
		"uint":    uint(5),
		"uint64":  uint64(6),
		"float32": float32(1.5),
		"float64": float64(2.5),
		"false":   false,
	}

	cases := []struct {
		key, want string
	}{
		{"int", "1"},
		{"int8", "2"},
		{"int32", "3"},
		{"int64", "4"},
		{"uint", "5"},
		{"uint64", "6"},
		{"float32", "1.5"},
		{"float64", "2.5"},
		{"false", "false"},
	}

	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			if got := v.Get(c.key); got != c.want {
				t.Errorf("Get(%q) = %q, want %q", c.key, got, c.want)
			}
		})
	}
}
