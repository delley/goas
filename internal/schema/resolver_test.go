package schema

import (
	"testing"

	"github.com/delley/goas/internal/types"
)

func TestResolveBasicType(t *testing.T) {
	r := NewResolver(false)

	tests := []struct {
		typeName string
		wantType string
		wantNil  bool
	}{
		{"int", "integer", false},
		{"string", "string", false},
		{"bool", "boolean", false},
		{"float64", "number", false},
		{"error", "", true},
		{"MyType", "", true},
	}

	for _, tt := range tests {
		schema := r.ResolveBasicType(tt.typeName)
		if tt.wantNil {
			if schema != nil {
				t.Errorf("ResolveBasicType(%q) = %v, want nil", tt.typeName, schema)
			}
		} else {
			if schema == nil {
				t.Errorf("ResolveBasicType(%q) = nil, want %q", tt.typeName, tt.wantType)
			} else if *schema.Type != tt.wantType {
				t.Errorf("ResolveBasicType(%q).Type = %q, want %q", tt.typeName, *schema.Type, tt.wantType)
			}
		}
	}
}

func TestHandleSliceType(t *testing.T) {
	r := NewResolver(false)

	tests := []struct {
		typeName  string
		wantSlice bool
		wantType  string
	}{
		{"[]int", true, "array"},
		{"[]string", true, "array"},
		{"int", false, ""},
		{"string", false, ""},
	}

	for _, tt := range tests {
		schema, err := r.HandleSliceType("main", tt.typeName)
		if err != nil {
			t.Errorf("HandleSliceType(%q) error: %v", tt.typeName, err)
			continue
		}
		if tt.wantSlice {
			if schema == nil {
				t.Errorf("HandleSliceType(%q) = nil, want slice schema", tt.typeName)
			} else if *schema.Type != tt.wantType {
				t.Errorf("HandleSliceType(%q).Type = %q, want %q", tt.typeName, *schema.Type, tt.wantType)
			}
		} else {
			if schema != nil {
				t.Errorf("HandleSliceType(%q) = %v, want nil", tt.typeName, schema)
			}
		}
	}
}

func TestHandleMapType(t *testing.T) {
	r := NewResolver(false)

	tests := []struct {
		typeName string
		wantMap  bool
		wantType string
	}{
		{"map[]int", true, "object"},
		{"map[]string", true, "object"},
		{"int", false, ""},
		{"[]string", false, ""},
	}

	for _, tt := range tests {
		schema, err := r.HandleMapType("main", tt.typeName)
		if err != nil {
			t.Errorf("HandleMapType(%q) error: %v", tt.typeName, err)
			continue
		}
		if tt.wantMap {
			if schema == nil {
				t.Errorf("HandleMapType(%q) = nil, want map schema", tt.typeName)
			} else if *schema.Type != tt.wantType {
				t.Errorf("HandleMapType(%q).Type = %q, want %q", tt.typeName, *schema.Type, tt.wantType)
			}
		} else {
			if schema != nil {
				t.Errorf("HandleMapType(%q) = %v, want nil", tt.typeName, schema)
			}
		}
	}
}

func TestHandleCompoundType(t *testing.T) {
	r := NewResolver(false)

	tests := []struct {
		typeName            string
		wantCompound        bool
		wantField           string
		wantError           bool
		expectedSchemaCount int
	}{
		{"oneOf(int,string)", true, "OneOf", false, 2},
		{"anyOf(int,string)", true, "AnyOf", false, 2},
		{"allOf(int,string)", true, "AllOf", false, 2},
		{"not(int)", true, "Not", false, 1},
		{"int", false, "", false, 0},
		{"string", false, "", false, 0},
		{"not(int,string)", true, "", true, 0}, // error: not takes only 1 arg
		{"oneOf()", true, "", true, 0},         // error: empty args
	}

	for _, tt := range tests {
		schema, err := r.HandleCompoundType("main", tt.typeName)
		if tt.wantError {
			if err == nil {
				t.Errorf("HandleCompoundType(%q) expected error, got nil", tt.typeName)
			}
		} else if err != nil {
			t.Errorf("HandleCompoundType(%q) error: %v", tt.typeName, err)
		} else if tt.wantCompound {
			if schema == nil {
				t.Errorf("HandleCompoundType(%q) = nil, want compound schema", tt.typeName)
			} else {
				// Check that appropriate field is set
				switch tt.wantField {
				case "OneOf":
					if schema.OneOf == nil || len(schema.OneOf) != tt.expectedSchemaCount {
						t.Errorf("HandleCompoundType(%q).OneOf length = %d, want %d", tt.typeName, len(schema.OneOf), tt.expectedSchemaCount)
					}
				case "AnyOf":
					if schema.AnyOf == nil || len(schema.AnyOf) != tt.expectedSchemaCount {
						t.Errorf("HandleCompoundType(%q).AnyOf length = %d, want %d", tt.typeName, len(schema.AnyOf), tt.expectedSchemaCount)
					}
				case "AllOf":
					if schema.AllOf == nil || len(schema.AllOf) != tt.expectedSchemaCount {
						t.Errorf("HandleCompoundType(%q).AllOf length = %d, want %d", tt.typeName, len(schema.AllOf), tt.expectedSchemaCount)
					}
				case "Not":
					if schema.Not == nil {
						t.Errorf("HandleCompoundType(%q).Not = nil, want schema", tt.typeName)
					}
				}
			}
		} else {
			if schema != nil {
				t.Errorf("HandleCompoundType(%q) = %v, want nil", tt.typeName, schema)
			}
		}
	}
}

func TestResolveType_BasicTypes(t *testing.T) {
	r := NewResolver(false)

	tests := []string{
		"int",
		"string",
		"bool",
		"float64",
	}

	for _, typeName := range tests {
		schema, err := r.ResolveType("main", typeName, false)
		if err != nil {
			t.Errorf("ResolveType(%q) error: %v", typeName, err)
		}
		if schema.Type == nil {
			t.Errorf("ResolveType(%q).Type = nil, want valid type", typeName)
		}
	}
}

func TestResolveType_Arrays(t *testing.T) {
	r := NewResolver(false)

	schema, err := r.ResolveType("main", "[]int", false)
	if err != nil {
		t.Errorf("ResolveType([]int) error: %v", err)
	}
	if schema.Type == nil || *schema.Type != "array" {
		t.Errorf("ResolveType([]int).Type = %q, want 'array'", *schema.Type)
	}
}

func TestResolveType_Maps(t *testing.T) {
	r := NewResolver(false)

	schema, err := r.ResolveType("main", "map[]string", false)
	if err != nil {
		t.Errorf("ResolveType(map[]string) error: %v", err)
	}
	if schema.Type == nil || *schema.Type != "object" {
		t.Errorf("ResolveType(map[]string).Type = %q, want 'object'", *schema.Type)
	}
}

func TestResolver_RegistryIntegration(t *testing.T) {
	r := NewResolver(false)

	// Registry should be initialized
	if r.Registry == nil {
		t.Error("Registry is nil")
	}
	if r.Registry.OmitPackages != false {
		t.Errorf("Registry.OmitPackages = %v, want false", r.Registry.OmitPackages)
	}

	// Test that registry is properly initialized
	if len(r.Registry.ApiSchemaNames) != 0 {
		t.Error("ApiSchemaNames should be empty initially")
	}
}

func TestResolver_TypeIndexing(t *testing.T) {
	r := NewResolver(false)

	if r.Types == nil {
		t.Error("Types index is nil")
	}

	if r.Types.TypeSpecs == nil {
		t.Error("TypeSpecs map is nil")
	}

	// Test registering a type spec
	// Note: We can't test actual TypeSpec registration without importing go/ast
	// But we can verify the structure is set up correctly
	if len(r.Types.TypeSpecs) != 0 {
		t.Error("TypeSpecs should be empty initially")
	}
}

// Test that basic type mapping is correct
func TestTypeMapping(t *testing.T) {
	r := NewResolver(false)

	for goType := range types.GoTypesOASTypes {
		schema := r.ResolveBasicType(goType)
		if schema == nil {
			t.Errorf("ResolveBasicType(%q) = nil, expected schema", goType)
		} else if schema.Type == nil {
			t.Errorf("ResolveBasicType(%q).Type = nil", goType)
		}
	}
}
