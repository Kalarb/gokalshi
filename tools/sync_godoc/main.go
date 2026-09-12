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
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/Kalarb/gokalshi/tools/internal/specsrc"
)

// Spec is a minimal representation of the OpenAPI spec.
type Spec struct {
	Paths map[string]map[string]*Operation `yaml:"paths"`
}

// Operation represents an API endpoint.
type Operation struct {
	OperationID string   `yaml:"operationId"`
	Summary     string   `yaml:"summary"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Deprecated  bool     `yaml:"deprecated"`
}

// EndpointInfo holds extracted info about an API endpoint.
type EndpointInfo struct {
	OperationID string
	Summary     string
	Description string
	Method      string // GET, POST, DELETE, etc.
	Path        string // /trade-api/v2/...
	Tag         string // spec tag, used to build the docs URL
	Deprecated  bool
}

// methodToOperationID maps Go Client method names to spec operationIds.
var methodToOperationID = map[string]string{
	// Orders
	"GetOrder":          "GetOrder",
	"GetOrders":         "GetOrders",
	"GetQueuePositions": "GetOrderQueuePositions",
	"GetQueuePosition":  "GetOrderQueuePosition",

	// Markets
	"GetMarketOrderbook":         "GetMarketOrderbook",
	"GetMarketOrderbooks":        "GetMarketOrderbooks",
	"GetTrades":                  "GetTrades",
	"GetMarket":                  "GetMarket",
	"GetMarkets":                 "GetMarkets",
	"GetMarketCandlesticks":      "GetMarketCandlesticks",
	"GetBatchMarketCandlesticks": "BatchGetMarketCandlesticks",

	// Events
	"GetEvent":                          "GetEvent",
	"GetEvents":                         "GetEvents",
	"GetEventMetadata":                  "GetEventMetadata",
	"GetMultivariateEvents":             "GetMultivariateEvents",
	"GetEventCandlesticks":              "GetMarketCandlesticksByEvent",
	"GetEventForecastPercentileHistory": "GetEventForecastPercentilesHistory",

	// Exchange
	"GetExchangeStatus":    "GetExchangeStatus",
	"GetExchangeSchedule":  "GetExchangeSchedule",
	"GetUserDataTimestamp": "GetUserDataTimestamp",
	"GetSeriesFeeChanges":  "GetSeriesFeeChanges",

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

	// Account
	"GetAccountEndpointCosts": "GetAccountEndpointCosts",

	// Portfolio
	"GetDeposits":                        "GetDeposits",
	"GetWithdrawals":                     "GetWithdrawals",
	"GetPortfolioRestingOrderTotalValue": "GetPortfolioRestingOrderTotalValue",

	// API Keys
	"GetAPIKeys":     "GetApiKeys",
	"CreateAPIKey":   "CreateApiKey",
	"GenerateAPIKey": "GenerateApiKey",
	"DeleteAPIKey":   "DeleteApiKey",

	// Event Orders (V2)
	"CreateOrderV2":       "CreateOrderV2",
	"BatchCreateOrdersV2": "BatchCreateOrdersV2",
	"BatchCancelOrdersV2": "BatchCancelOrdersV2",
	"CancelOrderV2":       "CancelOrderV2",
	"AmendOrderV2":        "AmendOrderV2",
	"DecreaseOrderV2":     "DecreaseOrderV2",

	// Events
	"GetEventFeeChanges": "GetEventFeeChanges",

	// Historical
	"GetHistoricalCutoff":             "GetHistoricalCutoff",
	"GetHistoricalFills":              "GetHistoricalFills",
	"GetHistoricalOrders":             "GetHistoricalOrders",
	"GetHistoricalTrades":             "GetHistoricalTrades",
	"GetHistoricalMarkets":            "GetHistoricalMarkets",
	"GetHistoricalMarket":             "GetHistoricalMarket",
	"GetHistoricalMarketCandlesticks": "GetHistoricalMarketCandlesticks",

	// Incentive Programs
	"GetIncentivePrograms": "GetIncentivePrograms",

	// Live Data
	"GetLiveDataBatch":       "GetLiveDataBatch",
	"GetLiveDataByMilestone": "GetLiveDataByMilestone",
	"GetMilestoneGameStats":  "GetMilestoneGameStats",
	"GetLiveData":            "GetLiveData",

	// Milestones
	"GetMilestones": "GetMilestones",
	"GetMilestone":  "GetMilestone",

	// Multivariate Event Collections
	"GetMultivariateEventCollections":           "GetMultivariateEventCollections",
	"GetMultivariateEventCollection":            "GetMultivariateEventCollection",
	"CreateMarketInMultivariateEventCollection": "CreateMarketInMultivariateEventCollection",

	// Structured Targets
	"GetStructuredTargets": "GetStructuredTargets",
	"GetStructuredTarget":  "GetStructuredTarget",

	// Subaccounts
	"CreateSubaccount":        "CreateSubaccount",
	"GetSubaccountBalances":   "GetSubaccountBalances",
	"GetSubaccountNetting":    "GetSubaccountNetting",
	"UpdateSubaccountNetting": "UpdateSubaccountNetting",
	"ApplySubaccountTransfer": "ApplySubaccountTransfer",
	"GetSubaccountTransfers":  "GetSubaccountTransfers",

	// Communications
	"GetCommunicationsID": "GetCommunicationsID",
	"CreateRFQ":           "CreateRFQ",
	"GetRFQs":             "GetRFQs",
	"GetRFQ":              "GetRFQ",
	"DeleteRFQ":           "DeleteRFQ",
	"CreateQuote":         "CreateQuote",
	"GetQuotes":           "GetQuotes",
	"GetQuote":            "GetQuote",
	"DeleteQuote":         "DeleteQuote",
	"AcceptQuote":         "AcceptQuote",
	"ConfirmQuote":        "ConfirmQuote",

	// Order Groups
	"CreateOrderGroup":      "CreateOrderGroup",
	"GetOrderGroups":        "GetOrderGroups",
	"GetOrderGroup":         "GetOrderGroup",
	"DeleteOrderGroup":      "DeleteOrderGroup",
	"ResetOrderGroup":       "ResetOrderGroup",
	"TriggerOrderGroup":     "TriggerOrderGroup",
	"UpdateOrderGroupLimit": "UpdateOrderGroupLimit",
	// Added for 3.30.0 parity
	"CancelAllOrders":                       "CancelAllOrders",
	"GetAccountAPIUsageLevelVolumeProgress": "GetAccountApiUsageLevelVolumeProgress",
	"UpgradeAPIUsageLevel":                  "UpgradeAccountApiUsageLevel",
	"GetHistoricalPositions":                "GetHistoricalPositions",
	"GetEventLiveData":                      "GetEventLiveData",
	"GetWeatherIndex":                       "GetWeatherIndex",
	"GetWeatherIndexCalibrations":           "GetWeatherIndexCalibrations",
	"GetTargetBalanceAllocation":            "GetTargetBalanceAllocation",
	"SetTargetBalanceAllocation":            "SetTargetBalanceAllocation",
	"IntraExchangeInstanceTransfer":         "IntraExchangeInstanceTransfer",
	"GetIntraExchangeInstanceTransfers":     "GetIntraExchangeInstanceTransfers",
	"GetIntraExchangeInstanceTransfer":      "GetIntraExchangeInstanceTransfer",
	"GetBlockTradeProposals":                "GetBlockTradeProposals",
	"ProposeBlockTrade":                     "ProposeBlockTrade",
	"AcceptBlockTradeProposal":              "AcceptBlockTradeProposal",
	"GetRFQQuote":                           "GetRFQQuote",
	"DeleteRFQQuote":                        "DeleteRFQQuote",
	"AcceptRFQQuote":                        "AcceptRFQQuote",
	"ConfirmRFQQuote":                       "ConfirmRFQQuote",
	"GetFCMOrders":                          "GetFCMOrders",
	"GetFCMPositions":                       "GetFCMPositions",
}

func main() {
	spec, err := loadSpec()
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

	// methodToOperationID is hand-maintained, and a method missing from it is
	// skipped in silence — which is how several methods came to keep a dead
	// documentation link across many runs. Make the gap visible instead.
	if unmapped := unmappedMethods(dir); len(unmapped) > 0 {
		fmt.Fprintf(os.Stderr, "\nWARNING: %d Client methods carry an endpoint annotation but are absent\n", len(unmapped))
		fmt.Fprintf(os.Stderr, "from methodToOperationID, so their godoc was not synced:\n")
		for _, m := range unmapped {
			fmt.Fprintf(os.Stderr, "  %s\n", m)
		}
	}
}

// endpointDocRe matches the "// GET /trade-api/v2/..." line that marks a method
// as an API endpoint wrapper.
var endpointDocRe = regexp.MustCompile(`(?m)^//\s+(?:GET|POST|PUT|DELETE|PATCH)\s+/trade-api/v2\S*`)

// unmappedMethods lists Client methods that look like endpoint wrappers but
// have no entry in methodToOperationID.
func unmappedMethods(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
		if err != nil {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Doc == nil {
				continue
			}
			if !endpointDocRe.MatchString(fn.Doc.Text()) && !endpointDocRe.MatchString(commentText(fn.Doc)) {
				continue
			}
			if _, mapped := methodToOperationID[fn.Name.Name]; !mapped {
				out = append(out, fn.Name.Name)
			}
		}
	}
	sort.Strings(out)
	return out
}

func commentText(g *ast.CommentGroup) string {
	var b strings.Builder
	for _, c := range g.List {
		b.WriteString(c.Text)
		b.WriteString("\n")
	}
	return b.String()
}

func loadSpec() (*Spec, error) {
	body, snap, err := specsrc.Load(specsrc.OpenAPI)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Using vendored OpenAPI %s (fetched %s)\n", snap.OpenAPI.Version, snap.FetchedAt)

	var spec Spec
	if err := yaml.Unmarshal(body, &spec); err != nil {
		return nil, fmt.Errorf("parse vendored OpenAPI YAML: %w", err)
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
				Tag:         firstTag(op.Tags),
				Deprecated:  op.Deprecated,
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
		newDoc := buildComment(methodName, info, existingDeprecation(fn.Doc))

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

// existingDeprecation returns the hand-written "// Deprecated: ..." paragraph
// from a doc comment, if any. The spec says an endpoint is deprecated but not
// what to use instead, so a hand-written notice naming the replacement is more
// useful than anything this tool can synthesise — preserve it.
func existingDeprecation(doc *ast.CommentGroup) []string {
	if doc == nil {
		return nil
	}
	var out []string
	for _, c := range doc.List {
		text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		if len(out) > 0 {
			if text == "" {
				break
			}
			out = append(out, c.Text)
			continue
		}
		if strings.HasPrefix(text, "Deprecated:") {
			out = append(out, c.Text)
		}
	}
	return out
}

func firstTag(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return tags[0]
}

func buildComment(methodName string, info *EndpointInfo, deprecation []string) string {
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

	// Deprecation notice, if the spec flags the endpoint. Prefer the
	// hand-written one, which can name the replacement.
	if len(deprecation) > 0 {
		lines = append(lines, "//")
		lines = append(lines, deprecation...)
	} else if info.Deprecated {
		lines = append(lines, "//")
		lines = append(lines, "// Deprecated: this endpoint is marked deprecated in the Kalshi API spec.")
	}

	// Last line: doc link.
	lines = append(lines, "//")
	lines = append(lines, "// See "+docURL(info))

	return strings.Join(lines, "\n") + "\n"
}

// docURL builds the current documentation link. The old trading-api.readme.io
// host now redirects to docs.kalshi.com, so linking it sends readers through a
// 302 to a generic landing page rather than the endpoint.
func docURL(info *EndpointInfo) string {
	if info.Tag == "" {
		return "https://docs.kalshi.com/api-reference"
	}
	return fmt.Sprintf("https://docs.kalshi.com/api-reference/%s/%s", info.Tag, kebabCase(info.OperationID))
}

// kebabCase converts "GetMarketOrderbook" to "get-market-orderbook".
func kebabCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('-')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
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
