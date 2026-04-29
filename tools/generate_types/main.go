// generate_types fetches the Kalshi OpenAPI spec and generates Go struct types.
//
// Usage:
//
//	go run ./tools/generate_types
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

const specURL = "https://docs.kalshi.com/openapi.yaml"

// skipPrefixes are schema name prefixes to exclude from generation.
var skipPrefixes = []string{"Subaccount", "IntraExchange", "Fcm"}

// typeAliases are schemas that map to simple Go types (not full structs).
var typeAliases = map[string]string{
	"FixedPointDollars": "string",
	"FixedPointCount":   "string",
}

// nullableOverrides are (schema, field) pairs where the spec says required
// but the real API returns null. Discovered via pykalshi's production testing.
var nullableOverrides = map[[2]string]bool{
	{"Series", "tags"}:                                   true,
	{"Series", "settlement_sources"}:                     true,
	{"Series", "contract_url"}:                           true,
	{"Series", "contract_terms_url"}:                     true,
	{"Series", "additional_prohibitions"}:                true,
	{"GetOrderQueuePositionsResponse", "queue_positions"}: true,
	{"GetLiveDatasResponse", "live_datas"}:                true,
}

// fieldTypeOverrides map (schema_name, field_name) to a Go type override.
// Used for inline enums that should use our named enum types from enums.go.
var fieldTypeOverrides = map[[2]string]string{
	// Side: yes | no (used across many schemas)
	{"CreateOrderRequest", "side"}:  "Side",
	{"AmendOrderRequest", "side"}:   "Side",
	{"Order", "side"}:               "Side",
	{"Trade", "side"}:               "Side",
	{"Trade", "taker_side"}:         "Side",
	{"Fill", "side"}:                "Side",
	{"MarketPosition", "side"}:      "Side",
	{"MveSelectedLeg", "side"}:      "Side",

	// Action: buy | sell
	{"CreateOrderRequest", "action"}: "Action",
	{"AmendOrderRequest", "action"}:  "Action",
	{"Order", "action"}:              "Action",
	{"Fill", "action"}:               "Action",

	// OrderType: limit | market
	{"Order", "type"}: "OrderType",

	// TimeInForce: fill_or_kill | good_till_canceled | immediate_or_cancel
	{"CreateOrderRequest", "time_in_force"}: "TimeInForce",

	// MarketStatus
	{"Market", "status"}: "MarketStatus",

	// MarketResult
	{"Market", "result"}: "MarketResult",

	// MarketType
	{"Market", "market_type"}: "MarketType",

	// AnnouncementType / AnnouncementStatus
	{"Announcement", "type"}:   "AnnouncementType",
	{"Announcement", "status"}: "AnnouncementStatus",
}

// Spec is a minimal representation of the OpenAPI spec.
type Spec struct {
	Components struct {
		Schemas map[string]*Schema `yaml:"schemas"`
	} `yaml:"components"`
	Paths map[string]map[string]*Operation `yaml:"paths"`
}

// Operation represents an API endpoint.
type Operation struct {
	OperationID string `yaml:"operationId"`
	Summary     string `yaml:"summary"`
	Description string `yaml:"description"`
}

// Schema represents an OpenAPI schema.
type Schema struct {
	Type                 string             `yaml:"type"`
	Format               string             `yaml:"format"`
	Description          string             `yaml:"description"`
	Enum                 []any              `yaml:"enum"`
	Properties           map[string]*Schema `yaml:"properties"`
	Required             []string           `yaml:"required"`
	Items                *Schema            `yaml:"items"`
	Ref                  string             `yaml:"$ref"`
	AllOf                []*Schema          `yaml:"allOf"`
	Nullable             bool               `yaml:"nullable"`
	Deprecated           bool               `yaml:"deprecated"`
	AdditionalProperties any                `yaml:"additionalProperties"`
}

func main() {
	spec, err := fetchSpec()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch spec: %v\n", err)
		os.Exit(1)
	}

	schemas := spec.Components.Schemas
	fmt.Printf("Loaded %d schemas from spec\n", len(schemas))

	enums, objects := categorize(schemas)
	fmt.Printf("  Enums: %d, Objects: %d\n", len(enums), len(objects))

	outDir := findPackageRoot()

	// Generate enums
	enumCode := generateEnums(enums, schemas)
	writeFile(outDir, "enums_generated.go", enumCode)

	// Generate object types (core, requests, responses all in one file)
	objectCode := generateObjects(objects, schemas)
	writeFile(outDir, "types_generated.go", objectCode)

	fmt.Println("Done!")
}

func fetchSpec() (*Spec, error) {
	fmt.Printf("Fetching %s...\n", specURL)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(specURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var spec Spec
	if err := yaml.Unmarshal(body, &spec); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	return &spec, nil
}

func shouldSkip(name string) bool {
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func categorize(schemas map[string]*Schema) (enums, objects []string) {
	for name, schema := range schemas {
		if shouldSkip(name) {
			continue
		}
		if _, ok := typeAliases[name]; ok {
			continue
		}
		// Skip query/path parameter schemas (not data types)
		if isParameterSchema(name) {
			continue
		}
		if schema.Type == "string" && len(schema.Enum) > 0 {
			enums = append(enums, name)
		} else {
			objects = append(objects, name)
		}
	}
	sort.Strings(enums)
	sort.Strings(objects)
	return enums, objects
}

// isParameterSchema returns true for schemas that represent query/path parameters
// rather than data types. These have names like "TickerQuery", "LimitQuery", etc.
func isParameterSchema(name string) bool {
	suffixes := []string{"Query", "Path"}
	for _, s := range suffixes {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

func refToName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

// resolveGoType converts an OpenAPI property schema to a Go type string.
// Only uses pointer types when the spec explicitly says nullable: true or
// when a nullableOverride is registered. Non-required fields use omitempty
// on the json tag instead of pointers (matching our existing convention).
func resolveGoType(schemaName, fieldName string, prop *Schema, required bool, schemas map[string]*Schema) string {
	// Check for field-level type override first
	if override, ok := fieldTypeOverrides[[2]string{schemaName, fieldName}]; ok {
		goType := override
		forceNullable := nullableOverrides[[2]string{schemaName, fieldName}]
		if prop.Nullable || forceNullable {
			return "*" + goType
		}
		return goType
	}

	goType := resolveGoTypeInner(prop, schemas)

	// Only use pointer when explicitly nullable or forced nullable.
	// Non-required fields just get omitempty on the json tag.
	forceNullable := nullableOverrides[[2]string{schemaName, fieldName}]
	needsPointer := (prop.Nullable || forceNullable) &&
		!strings.HasPrefix(goType, "[]") &&
		!strings.HasPrefix(goType, "map[")
	if needsPointer {
		return "*" + goType
	}
	return goType
}

func resolveGoTypeInner(prop *Schema, schemas map[string]*Schema) string {
	// Handle allOf wrapper (common for nullable refs)
	if len(prop.AllOf) == 1 {
		inner := prop.AllOf[0]
		return resolveGoTypeInner(inner, schemas)
	}

	// Handle $ref
	if prop.Ref != "" {
		refName := refToName(prop.Ref)
		if alias, ok := typeAliases[refName]; ok {
			return alias
		}
		if shouldSkip(refName) {
			return "any"
		}
		return refName
	}

	switch prop.Type {
	case "string":
		if prop.Format == "date-time" {
			return "string"
		}
		return "string"
	case "integer":
		if prop.Format == "int64" {
			return "int64"
		}
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		if prop.Items != nil {
			itemType := resolveGoTypeInner(prop.Items, schemas)
			return "[]" + itemType
		}
		return "[]any"
	case "object":
		if ap, ok := prop.AdditionalProperties.(*Schema); ok && ap != nil {
			valType := resolveGoTypeInner(ap, schemas)
			return "map[string]" + valType
		}
		return "map[string]any"
	default:
		return "any"
	}
}

// toGoFieldName converts a snake_case JSON field name to PascalCase Go field name.
func toGoFieldName(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Handle common abbreviations
		upper := strings.ToUpper(part)
		switch upper {
		case "ID", "URL", "API", "FP", "TS", "IP", "RFQ", "FCM", "MVE", "OHLC", "STP":
			result.WriteString(upper)
		default:
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			result.WriteString(string(runes))
		}
	}
	return result.String()
}

func generateEnums(names []string, schemas map[string]*Schema) string {
	var buf bytes.Buffer
	buf.WriteString(fileHeader("enums_generated.go"))

	buf.WriteString("package gokalshi\n\n")

	for _, name := range names {
		schema := schemas[name]
		if schema.Description != "" {
			buf.WriteString(fmt.Sprintf("// %s %s\n", name, cleanDescription(schema.Description)))
		}
		buf.WriteString(fmt.Sprintf("type %s string\n\n", name))
		buf.WriteString("const (\n")
		for _, val := range schema.Enum {
			constName := enumConstName(name, fmt.Sprint(val))
			buf.WriteString(fmt.Sprintf("\t%s %s = %q\n", constName, name, val))
		}
		buf.WriteString(")\n\n")
	}

	return buf.String()
}

func enumConstName(typeName, value string) string {
	// e.g., OrderStatus + "resting" → OrderStatusResting
	clean := strings.NewReplacer("-", "_", " ", "_").Replace(value)
	parts := strings.Split(clean, "_")
	var result strings.Builder
	result.WriteString(typeName)
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		result.WriteString(string(runes))
	}
	return result.String()
}

func generateObjects(names []string, schemas map[string]*Schema) string {
	sorted := topologicalSort(names, schemas)

	var buf bytes.Buffer
	buf.WriteString(fileHeader("types_generated.go"))

	buf.WriteString("package gokalshi\n\n")

	for _, name := range sorted {
		schema := schemas[name]
		if schema == nil {
			continue
		}

		// Handle array-type schemas
		if schema.Type == "array" && schema.Items != nil {
			itemType := resolveGoTypeInner(schema.Items, schemas)
			if schema.Description != "" {
				buf.WriteString(fmt.Sprintf("// %s %s\n", name, cleanDescription(schema.Description)))
			}
			buf.WriteString(fmt.Sprintf("type %s = []%s\n\n", name, itemType))
			continue
		}

		// Handle object schemas
		requiredSet := make(map[string]bool)
		for _, r := range schema.Required {
			requiredSet[r] = true
		}

		if schema.Description != "" {
			buf.WriteString(fmt.Sprintf("// %s %s\n", name, cleanDescription(schema.Description)))
		} else {
			buf.WriteString(fmt.Sprintf("// %s is a generated type from the Kalshi OpenAPI spec.\n", name))
		}
		buf.WriteString(fmt.Sprintf("type %s struct {\n", name))

		if len(schema.Properties) == 0 {
			buf.WriteString("}\n\n")
			continue
		}

		// Sort properties for deterministic output
		propNames := make([]string, 0, len(schema.Properties))
		for pn := range schema.Properties {
			propNames = append(propNames, pn)
		}
		sort.Strings(propNames)

		for _, propName := range propNames {
			prop := schema.Properties[propName]
			isRequired := requiredSet[propName]
			goType := resolveGoType(name, propName, prop, isRequired, schemas)
			goFieldName := toGoFieldName(propName)

			// Build json tag
			jsonTag := propName
			if !isRequired {
				jsonTag += ",omitempty"
			}

			// Add field description as comment if present
			if prop.Description != "" {
				desc := cleanDescription(prop.Description)
				// Truncate long descriptions
				if len(desc) > 100 {
					desc = desc[:97] + "..."
				}
				buf.WriteString(fmt.Sprintf("\t// %s\n", desc))
			}
			if prop.Deprecated {
				buf.WriteString("\t// Deprecated.\n")
			}
			buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", goFieldName, goType, jsonTag))
		}

		buf.WriteString("}\n\n")
	}

	return buf.String()
}

func topologicalSort(names []string, schemas map[string]*Schema) []string {
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	deps := make(map[string]map[string]bool)
	for _, name := range names {
		refs := collectRefs(schemas[name])
		d := make(map[string]bool)
		for r := range refs {
			if r != name && nameSet[r] {
				d[r] = true
			}
		}
		deps[name] = d
	}

	var sorted []string
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		if visiting[name] {
			return // circular dependency — break
		}
		visiting[name] = true
		for dep := range deps[name] {
			visit(dep)
		}
		delete(visiting, name)
		visited[name] = true
		sorted = append(sorted, name)
	}

	for _, name := range names {
		visit(name)
	}

	return sorted
}

func collectRefs(schema *Schema) map[string]bool {
	refs := make(map[string]bool)
	if schema == nil {
		return refs
	}
	if schema.Ref != "" {
		refs[refToName(schema.Ref)] = true
	}
	for _, prop := range schema.Properties {
		for r := range collectRefs(prop) {
			refs[r] = true
		}
	}
	if schema.Items != nil {
		for r := range collectRefs(schema.Items) {
			refs[r] = true
		}
	}
	for _, a := range schema.AllOf {
		for r := range collectRefs(a) {
			refs[r] = true
		}
	}
	return refs
}

var multiSpaceRe = regexp.MustCompile(`\s+`)

func cleanDescription(s string) string {
	// Take first sentence or first line
	s = strings.TrimSpace(s)
	lines := strings.SplitN(s, "\n", 2)
	s = strings.TrimSpace(lines[0])
	// Remove markdown headers
	s = strings.TrimLeft(s, "# ")
	// Collapse whitespace
	s = multiSpaceRe.ReplaceAllString(s, " ")
	return s
}

func fileHeader(filename string) string {
	return fmt.Sprintf("// Code generated by tools/generate_types from %s — DO NOT EDIT.\n// Source: %s\n\n", specURL, filename)
}

func findPackageRoot() string {
	// Find the gokalshi package root relative to this tool
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}
	// Check if we're already in the package root
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return dir
	}
	// Try parent directories
	for i := 0; i < 3; i++ {
		dir = filepath.Dir(dir)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	fmt.Fprintf(os.Stderr, "cannot find package root (go.mod)\n")
	os.Exit(1)
	return ""
}

func writeFile(dir, filename, content string) {
	// Format with gofmt
	formatted, err := format.Source([]byte(content))
	if err != nil {
		// Write unformatted for debugging
		path := filepath.Join(dir, filename)
		os.WriteFile(path, []byte(content), 0644)
		fmt.Fprintf(os.Stderr, "WARNING: gofmt failed for %s: %v (wrote unformatted)\n", filename, err)
		return
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
		os.Exit(1)
	}

	lines := bytes.Count(formatted, []byte("\n"))
	_ = slices.Contains([]string{}, "") // ensure slices import is used
	fmt.Printf("  Wrote %s (%d lines)\n", filename, lines)
}
