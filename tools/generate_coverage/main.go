// generate_coverage statically analyzes the gokalshi package and generates
// docs/API_COVERAGE.md with per-endpoint unit and integration test status.
//
// Usage:
//
//	go run ./tools/generate_coverage
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// APIMethod describes a single Client receiver method.
type APIMethod struct {
	Name     string // e.g. "GetExchangeStatus"
	Endpoint string // e.g. "GET /trade-api/v2/exchange/status"
	File     string // e.g. "exchange.go"
	Category string // e.g. "Exchange"
}

// WSChannel describes a WebSocket channel.
type WSChannel struct {
	Name string // e.g. "orderbook_delta"
}

// fileToCategory maps source filenames to display category names.
var fileToCategory = map[string]string{
	"account.go":            "Account",
	"exchange.go":           "Exchange",
	"orders.go":             "Orders",
	"event_orders.go":       "Event Orders (V2)",
	"portfolio.go":          "Portfolio",
	"summary.go":            "Portfolio", // override: contains GetPortfolioRestingOrderTotalValue
	"subaccounts.go":        "Subaccounts",
	"order_groups.go":       "Order Groups",
	"markets.go":            "Markets",
	"events.go":             "Events",
	"series.go":             "Series",
	"search.go":             "Search",
	"communications.go":     "Communications",
	"api_keys.go":           "API Keys",
	"historical.go":         "Historical",
	"incentive.go":          "Incentive Programs",
	"live_data.go":          "Live Data",
	"milestones.go":         "Milestones",
	"mve_collections.go":    "Multivariate Event Collections",
	"structured_targets.go": "Structured Targets",
}

// categoryOrder defines the display order (matches interfaces.go groupings).
var categoryOrder = []string{
	"Account",
	"Exchange",
	"Orders",
	"Event Orders (V2)",
	"Portfolio",
	"Subaccounts",
	"Order Groups",
	"Markets",
	"Events",
	"Series",
	"Search",
	"Communications",
	"API Keys",
	"Historical",
	"Incentive Programs",
	"Live Data",
	"Milestones",
	"Multivariate Event Collections",
	"Structured Targets",
}

// skipFiles are files that contain *Client methods but are not API endpoints.
var skipFiles = map[string]bool{
	"client.go":    true,
	"ws_client.go": true,
}

var endpointRe = regexp.MustCompile(`^//\s*(GET|POST|PUT|DELETE|PATCH)\s+(/trade-api/.+)$`)

func main() {
	dir := findPackageRoot()

	methods := discoverAPIMethods(dir)
	unitTests := discoverUnitTests(dir)
	integrationTests := discoverIntegrationTests(dir)
	wsChannels := discoverWSChannels(dir)
	channelNames := wsChannelNames(wsChannels)
	wsUnitTests := scanFileForChannels(filepath.Join(dir, "ws_client_test.go"), channelNames)
	wsIntegrationTests := scanFileForChannels(filepath.Join(dir, "ws_integration_test.go"), channelNames)

	report := generateMarkdown(methods, unitTests, integrationTests, wsChannels, wsUnitTests, wsIntegrationTests)

	outPath := filepath.Join(dir, "docs", "API_COVERAGE.md")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, []byte(report), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Coverage report written to docs/API_COVERAGE.md (%d HTTP endpoints, %d WS channels)\n",
		len(methods), len(wsChannels))
}

// discoverAPIMethods parses all non-test, non-generated .go files and extracts
// exported *Client receiver methods with their HTTP endpoints from godoc.
func discoverAPIMethods(dir string) []APIMethod {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read dir: %v\n", err)
		os.Exit(1)
	}

	var methods []APIMethod

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_generated.go") {
			continue
		}
		if skipFiles[name] {
			continue
		}

		category, ok := fileToCategory[name]
		if !ok {
			continue // not an API domain file
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", name, err)
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				continue
			}
			if !isClientReceiver(fn.Recv) {
				continue
			}
			if !fn.Name.IsExported() {
				continue
			}

			endpoint := extractEndpoint(fn.Doc)
			methods = append(methods, APIMethod{
				Name:     fn.Name.Name,
				Endpoint: endpoint,
				File:     name,
				Category: category,
			})
		}
	}

	return methods
}

// extractEndpoint scans a godoc comment group for a line like
// "// GET /trade-api/v2/exchange/status" and returns the shortened form.
func extractEndpoint(doc *ast.CommentGroup) string {
	if doc == nil {
		return "—"
	}
	for _, c := range doc.List {
		if m := endpointRe.FindStringSubmatch(c.Text); m != nil {
			// Shorten /trade-api/v2/... to just the path suffix
			path := strings.Replace(m[2], "/trade-api/v2", "", 1)
			return m[1] + " " + path
		}
	}
	return "—"
}

// isClientReceiver returns true if the receiver list is a single *Client.
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

// discoverUnitTests scans non-integration test files for test function names
// and returns the set of API method names that have unit tests.
func discoverUnitTests(dir string) map[string]bool {
	result := make(map[string]bool)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(dir, name)

		// Skip integration test files (have //go:build integration tag)
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if isIntegrationFile(string(src)) {
			continue
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			testName := fn.Name.Name
			if !strings.HasPrefix(testName, "Test") {
				continue
			}

			methodName := extractMethodFromTestName(testName)
			if methodName != "" {
				result[methodName] = true
			}
		}
	}

	return result
}

// extractMethodFromTestName converts "TestGetExchangeStatus" -> "GetExchangeStatus",
// "TestCreateOrder_PayloadSent" -> "CreateOrder", etc.
// Returns "" for non-API test names (e.g. "TestClient_Get_Success", "TestGetTradesParams_toMap").
func extractMethodFromTestName(testName string) string {
	without := strings.TrimPrefix(testName, "Test")
	if without == "" {
		return ""
	}

	// Skip generic client infrastructure tests (TestClient_*)
	if strings.HasPrefix(without, "Client_") {
		return ""
	}

	// Skip params toMap tests (e.g. TestGetTradesParams_toMap_AllFields)
	if strings.Contains(without, "Params_toMap") {
		return ""
	}

	// Skip WithXxx option tests
	if strings.HasPrefix(without, "With") {
		return ""
	}

	// Skip error type tests
	if strings.HasPrefix(without, "APIError") || strings.HasPrefix(without, "RateLimitError") {
		return ""
	}

	// Take the part before the first underscore as the method name
	if idx := strings.Index(without, "_"); idx > 0 {
		without = without[:idx]
	}

	return without
}

// discoverIntegrationTests scans integration test files for method references.
// It looks for t.Run("MethodName" patterns and c.MethodName( call patterns.
func discoverIntegrationTests(dir string) map[string]bool {
	result := make(map[string]bool)

	integrationFiles := []string{
		"http_integration_test.go",
		"http_integration_read_test.go",
		"http_integration_write_test.go",
	}

	tRunRe := regexp.MustCompile(`t\.Run\("(\w+)"`)
	methodCallRe := regexp.MustCompile(`c\.(\w+)\s*\(`)

	for _, name := range integrationFiles {
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		text := string(src)

		// Extract t.Run subtest names
		for _, m := range tRunRe.FindAllStringSubmatch(text, -1) {
			result[m[1]] = true
		}

		// Extract c.MethodName( calls
		for _, m := range methodCallRe.FindAllStringSubmatch(text, -1) {
			result[m[1]] = true
		}
	}

	return result
}

// discoverWSChannels extracts unique WebSocket channel names from ws_types.go.
func discoverWSChannels(dir string) []WSChannel {
	src, err := os.ReadFile(filepath.Join(dir, "ws_types.go"))
	if err != nil {
		return nil
	}

	// Extract channel names from map values like {"orderbook_delta"}
	channelRe := regexp.MustCompile(`"(\w+)"`)
	seen := make(map[string]bool)
	var channels []WSChannel

	inMap := false
	for _, line := range strings.Split(string(src), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "MsgTypeToChannel") && strings.Contains(trimmed, "map[") {
			inMap = true
			continue
		}
		if inMap && trimmed == "}" {
			break
		}
		if !inMap {
			continue
		}

		// Find channel names in the slice values (right side of the colon)
		colonIdx := strings.Index(trimmed, ":")
		if colonIdx < 0 {
			continue
		}
		valuePart := trimmed[colonIdx+1:]
		for _, m := range channelRe.FindAllStringSubmatch(valuePart, -1) {
			name := m[1]
			if !seen[name] {
				seen[name] = true
				channels = append(channels, WSChannel{Name: name})
			}
		}
	}

	return channels
}

// wsChannelNames extracts the name strings from a slice of WSChannel.
func wsChannelNames(channels []WSChannel) []string {
	names := make([]string, len(channels))
	for i, ch := range channels {
		names[i] = ch.Name
	}
	return names
}

// scanFileForChannels checks whether a file references each channel name
// as a quoted string (e.g. "orderbook_delta") to avoid substring false positives.
func scanFileForChannels(path string, channels []string) map[string]bool {
	result := make(map[string]bool)
	src, err := os.ReadFile(path)
	if err != nil {
		return result
	}
	text := string(src)

	for _, ch := range channels {
		if strings.Contains(text, `"`+ch+`"`) {
			result[ch] = true
		}
	}
	return result
}

func isIntegrationFile(src string) bool {
	// Check first few lines for build tag
	for _, line := range strings.SplitN(src, "\n", 5) {
		if strings.Contains(line, "//go:build integration") {
			return true
		}
	}
	return false
}

// generateMarkdown produces the full coverage report.
func generateMarkdown(
	methods []APIMethod,
	unitTests, integrationTests map[string]bool,
	wsChannels []WSChannel,
	wsUnitTests, wsIntegrationTests map[string]bool,
) string {
	now := time.Now().UTC().Format("2006-01-02 15:04 UTC")

	// Group methods by category
	byCategory := make(map[string][]APIMethod)
	for _, m := range methods {
		byCategory[m.Category] = append(byCategory[m.Category], m)
	}

	var b strings.Builder

	b.WriteString("# API Coverage\n\n")
	b.WriteString(fmt.Sprintf("> Auto-generated by `go run ./tools/generate_coverage` on %s. Do not edit manually.\n\n", now))

	// Summary table
	b.WriteString("## Summary\n\n")
	b.WriteString("| Category | Endpoints | Unit Tests | Integration Tests |\n")
	b.WriteString("|----------|:---------:|:----------:|:-----------------:|\n")

	totalEndpoints := 0
	totalUnit := 0
	totalIntegration := 0

	for _, cat := range categoryOrder {
		catMethods := byCategory[cat]
		nEndpoints := len(catMethods)
		nUnit := 0
		nIntegration := 0
		for _, m := range catMethods {
			if unitTests[m.Name] {
				nUnit++
			}
			if integrationTests[m.Name] {
				nIntegration++
			}
		}
		totalEndpoints += nEndpoints
		totalUnit += nUnit
		totalIntegration += nIntegration
		b.WriteString(fmt.Sprintf("| %s | %d | %d/%d | %d/%d |\n",
			cat, nEndpoints, nUnit, nEndpoints, nIntegration, nEndpoints))
	}
	b.WriteString(fmt.Sprintf("| **Total** | **%d** | **%d/%d** | **%d/%d** |\n\n",
		totalEndpoints, totalUnit, totalEndpoints, totalIntegration, totalEndpoints))

	// Per-category detail tables
	b.WriteString("## HTTP Endpoints\n\n")

	for _, cat := range categoryOrder {
		catMethods := byCategory[cat]
		if len(catMethods) == 0 {
			continue
		}

		b.WriteString(fmt.Sprintf("### %s\n\n", cat))
		b.WriteString("| Method | Endpoint | Unit | Integration | Notes |\n")
		b.WriteString("|--------|----------|:----:|:-----------:|-------|\n")

		for _, m := range catMethods {
			unit := "—"
			if unitTests[m.Name] {
				unit = "Y"
			}
			integration := "—"
			if integrationTests[m.Name] {
				integration = "Y"
			}
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s | |\n",
				m.Name, m.Endpoint, unit, integration))
		}
		b.WriteString("\n")
	}

	// WebSocket channels
	if len(wsChannels) > 0 {
		b.WriteString("## WebSocket Channels\n\n")
		b.WriteString("| Channel | Unit | Integration | Notes |\n")
		b.WriteString("|---------|:----:|:-----------:|-------|\n")

		for _, ch := range wsChannels {
			unit := "—"
			if wsUnitTests[ch.Name] {
				unit = "Y"
			}
			integration := "—"
			if wsIntegrationTests[ch.Name] {
				integration = "Y"
			}
			b.WriteString(fmt.Sprintf("| `%s` | %s | %s | |\n",
				ch.Name, unit, integration))
		}
		b.WriteString("\n")
	}

	return b.String()
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
