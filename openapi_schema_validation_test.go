//go:build spec_validation

package gokalshi

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// This is the Go counterpart of pykalshi's test_openapi_schema_coverage.
//
// TestOpenAPICoverage only compares endpoints, so a schema that gained a field
// — or a whole type — went unnoticed until something failed to deserialize at
// runtime. Kalshi's 2026 fixed-point migration added fields across most
// schemas, which is exactly the drift an endpoint-only check cannot see.

// schemaSkipPrefixes mirrors skipPrefixes in tools/generate_types: schemas the
// generator deliberately does not emit.
var schemaSkipPrefixes = []string{}

// schemaTypeAliases mirrors typeAliases in tools/generate_types: spec schemas
// that map to a bare Go type rather than a struct.
var schemaTypeAliases = map[string]bool{
	"FixedPointDollars": true,
	"FixedPointCount":   true,
}

// goTypes is what the package actually declares, extracted from source.
type goTypes struct {
	structFields map[string]map[string]bool // type name -> set of json tag names
	enumValues   map[string]map[string]bool // type name -> set of string values
	nonStruct    map[string]bool            // named non-struct types (aliases, enums)
}

// parseGoTypes walks the package's non-test sources and records struct json
// tags, named non-struct types, and the values of typed string constants.
func parseGoTypes(t *testing.T) *goTypes {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	require.NoError(t, err, "parse package sources")

	out := &goTypes{
		structFields: map[string]map[string]bool{},
		enumValues:   map[string]map[string]bool{},
		nonStruct:    map[string]bool{},
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				switch gen.Tok {
				case token.TYPE:
					collectTypeDecls(gen, out)
				case token.CONST:
					collectEnumConsts(gen, out)
				}
			}
		}
	}
	return out
}

func collectTypeDecls(gen *ast.GenDecl, out *goTypes) {
	for _, spec := range gen.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		st, isStruct := ts.Type.(*ast.StructType)
		if !isStruct {
			out.nonStruct[ts.Name.Name] = true
			continue
		}
		fields := map[string]bool{}
		for _, f := range st.Fields.List {
			name := jsonTagName(f)
			if name != "" {
				fields[name] = true
			}
		}
		out.structFields[ts.Name.Name] = fields
	}
}

// jsonTagName returns the wire name a struct field marshals to, or "" if the
// field is unexported, embedded, or explicitly skipped.
func jsonTagName(f *ast.Field) string {
	if f.Tag == nil || len(f.Names) == 0 {
		return ""
	}
	tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
	name, _, _ := strings.Cut(tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

func collectEnumConsts(gen *ast.GenDecl, out *goTypes) {
	for _, spec := range gen.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok || vs.Type == nil {
			continue
		}
		ident, ok := vs.Type.(*ast.Ident)
		if !ok {
			continue
		}
		for _, val := range vs.Values {
			lit, ok := val.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			if out.enumValues[ident.Name] == nil {
				out.enumValues[ident.Name] = map[string]bool{}
			}
			out.enumValues[ident.Name][strings.Trim(lit.Value, `"`)] = true
		}
	}
}

// specSchema is the subset of an OpenAPI schema this test reasons about.
type specSchema struct {
	Type       string                 `yaml:"type"`
	Enum       []string               `yaml:"enum"`
	Properties map[string]*specSchema `yaml:"properties"`
	AllOf      []*specSchema          `yaml:"allOf"`
	Ref        string                 `yaml:"$ref"`
}

// resolveProperties flattens allOf wrappers the way the generator does.
func resolveProperties(s *specSchema, all map[string]*specSchema) map[string]*specSchema {
	if len(s.AllOf) == 0 {
		return s.Properties
	}
	merged := map[string]*specSchema{}
	for _, sub := range s.AllOf {
		if sub.Ref != "" {
			if ref, ok := all[sub.Ref[strings.LastIndex(sub.Ref, "/")+1:]]; ok {
				for k, v := range ref.Properties {
					merged[k] = v
				}
			}
			continue
		}
		for k, v := range sub.Properties {
			merged[k] = v
		}
	}
	for k, v := range s.Properties {
		merged[k] = v
	}
	return merged
}

func skipSchema(name string) bool {
	if schemaTypeAliases[name] {
		return true
	}
	for _, p := range schemaSkipPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func TestOpenAPISchemaCoverage(t *testing.T) {
	body := loadVendoredSpec(t, "openapi")

	var spec struct {
		Components struct {
			Schemas map[string]*specSchema `yaml:"schemas"`
		} `yaml:"components"`
	}
	require.NoError(t, yaml.Unmarshal(body, &spec))
	schemas := spec.Components.Schemas
	require.NotEmpty(t, schemas, "spec declared no component schemas")

	ours := parseGoTypes(t)

	var missingTypes, missingFields, missingEnumValues []string

	for name, schema := range schemas {
		if skipSchema(name) {
			continue
		}

		// String enums: compare the declared constant values.
		if schema.Type == "string" && len(schema.Enum) > 0 {
			values, declared := ours.enumValues[name]
			if !declared {
				if !ours.nonStruct[name] {
					missingTypes = append(missingTypes, name)
				}
				continue
			}
			for _, want := range schema.Enum {
				if !values[want] {
					missingEnumValues = append(missingEnumValues, name+"."+want)
				}
			}
			continue
		}

		fields, declared := ours.structFields[name]
		if !declared {
			if !ours.nonStruct[name] {
				missingTypes = append(missingTypes, name)
			}
			continue
		}
		for prop := range resolveProperties(schema, schemas) {
			if !fields[prop] {
				missingFields = append(missingFields, name+"."+prop)
			}
		}
	}

	sort.Strings(missingTypes)
	sort.Strings(missingFields)
	sort.Strings(missingEnumValues)

	t.Logf("spec schemas: %d", len(schemas))
	t.Logf("Go structs: %d, enums: %d", len(ours.structFields), len(ours.enumValues))
	t.Logf("missing types: %d, missing fields: %d, missing enum values: %d",
		len(missingTypes), len(missingFields), len(missingEnumValues))
	for _, x := range missingTypes {
		t.Logf("  missing type: %s", x)
	}
	for _, x := range missingFields {
		t.Logf("  missing field: %s", x)
	}
	for _, x := range missingEnumValues {
		t.Logf("  missing enum value: %s", x)
	}

	assert.Empty(t, missingTypes, fmt.Sprintf("%d spec schemas have no Go type", len(missingTypes)))
	assert.Empty(t, missingFields, fmt.Sprintf("%d spec fields have no Go struct field", len(missingFields)))
	assert.Empty(t, missingEnumValues, fmt.Sprintf("%d spec enum values are not declared", len(missingEnumValues)))
}
