package types

import (
	"go/parser"
	"testing"
)

func TestIsBasicGoType(t *testing.T) {
	tests := map[string]bool{
		"int":       true,
		"string":    true,
		"bool":      true,
		"float64":   true,
		"MyType":    false,
		"time.Time": false,
	}
	for typeName, expected := range tests {
		if IsBasicGoType(typeName) != expected {
			t.Errorf("IsBasicGoType(%q) = %v, want %v", typeName, !expected, expected)
		}
	}
}

func TestIsGoTypeOASType(t *testing.T) {
	tests := map[string]bool{
		"int":       true,
		"string":    true,
		"bool":      true,
		"float64":   true,
		"error":     false,
		"MyType":    false,
		"time.Time": false,
	}
	for typeName, expected := range tests {
		if IsGoTypeOASType(typeName) != expected {
			t.Errorf("IsGoTypeOASType(%q) = %v, want %v", typeName, !expected, expected)
		}
	}
}

func TestGetOASType(t *testing.T) {
	tests := map[string]string{
		"int":     "integer",
		"string":  "string",
		"bool":    "boolean",
		"float64": "number",
		"unknown": "",
	}
	for goType, expected := range tests {
		if got := GetOASType(goType); got != expected {
			t.Errorf("GetOASType(%q) = %q, want %q", goType, got, expected)
		}
	}
}

func TestGetOASFormat(t *testing.T) {
	tests := map[string]string{
		"int64":   "int64",
		"float32": "float",
		"float64": "double",
		"bool":    "boolean",
		"unknown": "",
	}
	for goType, expected := range tests {
		if got := GetOASFormat(goType); got != expected {
			t.Errorf("GetOASFormat(%q) = %q, want %q", goType, got, expected)
		}
	}
}

func TestTypeAsString(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"int", "int"},
		{"string", "string"},
		{"[]int", "[]int"},
		{"*int", "int"},
		{"interface{}", "interface{}"},
		{"time.Time", "time.Time"},
	}

	for _, tt := range tests {
		expr, err := parser.ParseExpr(tt.code)
		if err != nil {
			t.Fatalf("failed to parse %q: %v", tt.code, err)
		}

		got := TypeAsString(expr)
		if got != tt.expected {
			t.Errorf("TypeAsString(%q) = %q, want %q", tt.code, got, tt.expected)
		}
	}
}

func TestIsSliceOrMapType(t *testing.T) {
	tests := map[string]bool{
		"[]int":       true,
		"map[]string": true,
		"interface{}": true,
		"int":         false,
		"string":      false,
		"time.Time":   false,
	}
	for typeName, expected := range tests {
		if IsSliceOrMapType(typeName) != expected {
			t.Errorf("IsSliceOrMapType(%q) = %v, want %v", typeName, !expected, expected)
		}
	}
}

func TestIsSpecialType(t *testing.T) {
	tests := map[string]bool{
		"time.Time": true,
		"uuid.UUID": true,
		"int":       false,
		"string":    false,
		"MyType":    false,
	}
	for typeName, expected := range tests {
		if IsSpecialType(typeName) != expected {
			t.Errorf("IsSpecialType(%q) = %v, want %v", typeName, !expected, expected)
		}
	}
}

func TestNormalizeTypeName(t *testing.T) {
	tests := map[string]string{
		"MyType":       "MyType",
		"pkg.MyType":   "MyType",
		"a.b.c.MyType": "MyType",
	}
	for typeName, expected := range tests {
		if got := NormalizeTypeName(typeName); got != expected {
			t.Errorf("NormalizeTypeName(%q) = %q, want %q", typeName, got, expected)
		}
	}
}
