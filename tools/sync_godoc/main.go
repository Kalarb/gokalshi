// sync_godoc fetches the Kalshi OpenAPI spec and updates godoc comments
// on Client methods with the spec's summary, description, and doc link.
//
// Usage:
//
//	go run ./tools/sync_godoc
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const specURL = "https://docs.kalshi.com/openapi.yaml"

// Spec is a minimal representation of the OpenAPI spec.
type Spec struct {
	Paths map[string]map[string]*Operation `yaml:"paths"`
}

// Operation represents an API endpoint.
type Operation struct {
	OperationID string `yaml:"operationId"`
	Summary     string `yaml:"summary"`
	Description string `yaml:"description"`
}

// EndpointInfo holds extracted info about an API endpoint.
type EndpointInfo struct {
	OperationID string
	Summary     string
	Description string
	Method      string // GET, POST, DELETE, etc.
	Path        string // /trade-api/v2/...
}

// methodToOperationID maps Go Client method names to spec operationIds.
var methodToOperationID = map[string]string{
	// Orders
	"CreateOrder":       "CreateOrder",
	"CancelOrder":       "CancelOrder",
	"GetOrder":          "GetOrder",
	"GetOrders":         "GetOrders",
	"BatchCreateOrders": "BatchCreateOrders",
	"BatchCancelOrders": "BatchCancelOrders",
	"AmendOrder":        "AmendOrder",
	"DecreaseOrder":     "DecreaseOrder",
	"GetQueuePositions": "GetOrderQueuePositions",
	"GetQueuePosition":  "GetOrderQueuePosition",

	// Markets
	"GetMarketOrderbook":        "GetMarketOrderbook",
	"GetMarketOrderbooks":       "GetMarketOrderbooks",
	"GetTrades":                 "GetTrades",
	"GetMarket":                 "GetMarket",
	"GetMarkets":                "GetMarkets",
	"GetMarketCandlesticks":     "GetMarketCandlesticks",
	"GetBatchMarketCandlesticks": "BatchGetMarketCandlesticks",

	// Events
	"GetEvent":                          "GetEvent",
	"GetEvents":                         "GetEvents",
	"GetEventMetadata":                  "GetEventMetadata",
	"GetMultivariateEvents":             "GetMultivariateEvents",
	"GetEventCandlesticks":              "GetMarketCandlesticksByEvent",
	"GetEventForecastPercentileHistory": "GetEventForecastPercentilesHistory",

	// Exchange
	"GetExchangeStatus":        "GetExchangeStatus",
	"GetExchangeAnnouncements": "GetExchangeAnnouncements",
	"GetExchangeSchedule":      "GetExchangeSchedule",
	"GetUserDataTimestamp":      "GetUserDataTimestamp",
	"GetSeriesFeeChanges":      "GetSeriesFeeChanges",

	// Portfolio
	"GetBalance":     "GetBalance",
	"GetPositions":   "GetPositions",
	"GetFills":       "GetFills",
	"GetSettlements": "GetSettlements",

	// Series
	"GetSeries":     "GetSeries",
	"GetSeriesList": "GetSeriesList",

	// Search
	"GetTagsByCategories": "GetTagsForSeriesCategories",
	"GetFiltersBySport":   "GetFiltersForSports",

	// Account
	"GetAccountAPILimits": "GetAccountApiLimits",
}

func main() {
	spec, err := fetchSpec()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch spec: %v\n", err)
		os.Exit(1)
	}

	endpoints := buildEndpointMap(spec)
	fmt.Printf("Loaded %d endpoints from spec\n", len(endpoints))

	dir := findPackageRoot()
	total := 0

	// Process each .go file in the package root
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read dir: %v\n", err)
		os.Exit(1)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") || strings.HasSuffix(entry.Name(), "_generated.go") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		n, err := processFile(path, endpoints)
		if err != nil {
			fmt.Fprintf(os.Stderr, "process %s: %v\n", entry.Name(), err)
			continue
		}
		if n > 0 {
			fmt.Printf("  %s: updated %d methods\n", entry.Name(), n)
			total += n
		}
	}

	fmt.Printf("Done! Updated %d method comments.\n", total)
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

func buildEndpointMap(spec *Spec) map[string]*EndpointInfo {
	result := make(map[string]*EndpointInfo)

	for pathKey, methods := range spec.Paths {
		fullPath := pathKey
		if !strings.HasPrefix(pathKey, "/trade-api") {
			fullPath = "/trade-api/v2" + pathKey
		}

		for method, op := range methods {
			httpMethod := strings.ToUpper(method)
			if httpMethod != "GET" && httpMethod != "POST" && httpMethod != "PUT" && httpMethod != "DELETE" && httpMethod != "PATCH" {
				continue
			}
			result[op.OperationID] = &EndpointInfo{
				OperationID: op.OperationID,
				Summary:     strings.TrimSpace(op.Summary),
				Description: strings.TrimSpace(op.Description),
				Method:      httpMethod,
				Path:        fullPath,
			}
		}
	}

	return result
}

func processFile(path string, endpoints map[string]*EndpointInfo) (int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return 0, fmt.Errorf("parse: %w", err)
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	type replacement struct {
		start, end int    // byte offsets in original source
		newComment string // replacement text
	}

	var replacements []replacement

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil {
			continue
		}

		// Check if this is a *Client method
		if !isClientReceiver(fn.Recv) {
			continue
		}

		methodName := fn.Name.Name
		opID, ok := methodToOperationID[methodName]
		if !ok {
			continue
		}

		info, ok := endpoints[opID]
		if !ok {
			continue
		}

		// Build new comment
		newDoc := buildComment(methodName, info)

		// Calculate replacement range.
		// Include any whitespace between the doc comment end and the func
		// keyword to prevent blank line accumulation on repeated runs.
		var startOff, endOff int
		if fn.Doc != nil {
			startOff = fset.Position(fn.Doc.Pos()).Offset
			funcOff := fset.Position(fn.Pos()).Offset
			endOff = funcOff // replace everything from doc start to func start
		} else {
			// No existing doc — insert before func keyword
			startOff = fset.Position(fn.Pos()).Offset
			endOff = startOff
		}

		replacements = append(replacements, replacement{
			start:      startOff,
			end:        endOff,
			newComment: newDoc,
		})
	}

	if len(replacements) == 0 {
		return 0, nil
	}

	// Apply replacements in reverse order
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].start > replacements[j].start
	})

	result := make([]byte, len(src))
	copy(result, src)

	for _, r := range replacements {
		var buf bytes.Buffer
		buf.Write(result[:r.start])
		buf.WriteString(r.newComment)
		buf.Write(result[r.end:])
		result = buf.Bytes()
	}

	// Format with gofmt
	formatted, err := format.Source(result)
	if err != nil {
		// Write unformatted for debugging
		os.WriteFile(path, result, 0644)
		return len(replacements), fmt.Errorf("gofmt: %w (wrote unformatted)", err)
	}

	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return 0, err
	}

	return len(replacements), nil
}

func isClientReceiver(recv *ast.FieldList) bool {
	if recv == nil || len(recv.List) != 1 {
		return false
	}
	star, ok := recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	ident, ok := star.X.(*ast.Ident)
	return ok && ident.Name == "Client"
}

var descriptionCleanRe = regexp.MustCompile(`\s+`)

func buildComment(methodName string, info *EndpointInfo) string {
	var lines []string

	// Line 1: summary
	summary := info.Summary
	if summary == "" {
		summary = methodName
	}
	// Ensure first line starts with method name (Go convention)
	if !strings.HasPrefix(summary, methodName) {
		lines = append(lines, fmt.Sprintf("// %s — %s", methodName, summary))
	} else {
		lines = append(lines, fmt.Sprintf("// %s", summary))
	}

	// Line 2: blank + HTTP method/path
	lines = append(lines, "//")
	lines = append(lines, fmt.Sprintf("// %s %s", info.Method, info.Path))

	// Lines 3+: description (first paragraph only, if non-empty)
	if info.Description != "" {
		desc := cleanDescription(info.Description)
		if desc != "" && desc != summary {
			lines = append(lines, "//")
			wrapped := wrapText(desc, 76) // 80 - "// " prefix
			for _, line := range wrapped {
				if line == "" {
					lines = append(lines, "//")
				} else {
					lines = append(lines, "// "+line)
				}
			}
		}
	}

	// Last line: doc link
	docSlug := strings.ToLower(info.OperationID)
	lines = append(lines, "//")
	lines = append(lines, fmt.Sprintf("// See https://trading-api.readme.io/reference/%s", docSlug))

	return strings.Join(lines, "\n") + "\n"
}

func cleanDescription(s string) string {
	// Take first paragraph (up to first blank line or markdown header)
	lines := strings.Split(s, "\n")
	var clean []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && len(clean) > 0 {
			break
		}
		if strings.HasPrefix(trimmed, "##") || strings.HasPrefix(trimmed, "```") {
			break
		}
		clean = append(clean, trimmed)
	}
	result := strings.Join(clean, " ")
	result = descriptionCleanRe.ReplaceAllString(result, " ")
	return strings.TrimSpace(result)
}

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	current := words[0]

	for _, word := range words[1:] {
		if len(current)+1+len(word) > width {
			lines = append(lines, current)
			current = word
		} else {
			current += " " + word
		}
	}
	lines = append(lines, current)
	return lines
}

func findPackageRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return dir
	}
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
