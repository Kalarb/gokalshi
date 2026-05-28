package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestCategoryFromFile(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"exchange.go", "Exchange"},
		{"orders.go", "Orders"},
		{"portfolio.go", "Portfolio"},
		{"summary.go", "Portfolio"}, // override
		{"communications.go", "Communications"},
		{"mve_collections.go", "Multivariate Event Collections"},
	}

	for _, tt := range tests {
		got, ok := fileToCategory[tt.file]
		if !ok {
			t.Errorf("fileToCategory[%q] not found", tt.file)
			continue
		}
		if got != tt.expected {
			t.Errorf("fileToCategory[%q] = %q, want %q", tt.file, got, tt.expected)
		}
	}
}

func TestCategoryFromFile_SkipNonDomain(t *testing.T) {
	nonDomain := []string{"client.go", "ws_client.go", "auth.go", "config.go", "paths.go"}
	for _, f := range nonDomain {
		if _, ok := fileToCategory[f]; ok {
			t.Errorf("fileToCategory[%q] should not exist (non-domain file)", f)
		}
	}
}

func TestExtractMethodFromTestName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"TestGetExchangeStatus", "GetExchangeStatus"},
		{"TestCreateOrder_PayloadSent", "CreateOrder"},
		{"TestCancelOrder_Path", "CancelOrder"},
		{"TestGetTradesParams_toMap_AllFields", ""},  // params test
		{"TestClient_Get_Success", ""},                // client infra test
		{"TestClient_Retry429", ""},                   // client infra test
		{"TestWithHTTPClient", ""},                     // option test
		{"TestWithMaxRetries", ""},                     // option test
		{"TestAPIError_Error", ""},                     // error type test
		{"TestGetBatchMarketCandlesticks", "GetBatchMarketCandlesticks"},
		{"TestLookupTickersForMarketInMultivariateEventCollection", "LookupTickersForMarketInMultivariateEventCollection"},
		{"TestDeleteAPIKey", "DeleteAPIKey"},
	}

	for _, tt := range tests {
		got := extractMethodFromTestName(tt.input)
		if got != tt.expected {
			t.Errorf("extractMethodFromTestName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtractEndpoint(t *testing.T) {
	src := `package gokalshi

// GetExchangeStatus — Get Exchange Status
//
// GET /trade-api/v2/exchange/status
//
// Endpoint for getting the exchange status.
func (c *Client) GetExchangeStatus() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := extractEndpoint(fn.Doc)
		expected := "GET /exchange/status"
		if got != expected {
			t.Errorf("extractEndpoint() = %q, want %q", got, expected)
		}
	}
}

func TestExtractEndpoint_NoDoc(t *testing.T) {
	got := extractEndpoint(nil)
	if got != "—" {
		t.Errorf("extractEndpoint(nil) = %q, want %q", got, "—")
	}
}

func TestExtractEndpoint_NoEndpointLine(t *testing.T) {
	src := `package gokalshi

// Close closes the client.
func (c *Client) Close() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := extractEndpoint(fn.Doc)
		if got != "—" {
			t.Errorf("extractEndpoint() = %q, want %q", got, "—")
		}
	}
}

func TestIsClientReceiver(t *testing.T) {
	src := `package gokalshi

func (c *Client) Foo() {}
func (w *WSClient) Bar() {}
func standalone() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]bool{
		"Foo":        true,
		"Bar":        false,
		"standalone": false,
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := fn.Recv != nil && isClientReceiver(fn.Recv)
		want := expected[fn.Name.Name]
		if got != want {
			t.Errorf("isClientReceiver for %s = %v, want %v", fn.Name.Name, got, want)
		}
	}
}

func TestIsIntegrationFile(t *testing.T) {
	integration := "//go:build integration\n\npackage gokalshi\n"
	regular := "package gokalshi\n\nimport \"testing\"\n"

	if !isIntegrationFile(integration) {
		t.Error("expected integration file to be detected")
	}
	if isIntegrationFile(regular) {
		t.Error("expected regular file to not be detected as integration")
	}
}

func TestGenerateMarkdown_Format(t *testing.T) {
	methods := []APIMethod{
		{Name: "GetFoo", Endpoint: "GET /foo", File: "exchange.go", Category: "Exchange"},
		{Name: "CreateBar", Endpoint: "POST /bar", File: "exchange.go", Category: "Exchange"},
	}
	unitTests := map[string]bool{"GetFoo": true}
	integrationTests := map[string]bool{"GetFoo": true, "CreateBar": true}
	wsChannels := []WSChannel{{Name: "ticker"}}
	wsUnit := map[string]bool{"ticker": true}
	wsIntegration := map[string]bool{"ticker": true}

	result := generateMarkdown(methods, unitTests, integrationTests, wsChannels, wsUnit, wsIntegration)

	// Check header
	if !strings.Contains(result, "# API Coverage") {
		t.Error("missing header")
	}
	if !strings.Contains(result, "Auto-generated") {
		t.Error("missing auto-generated notice")
	}

	// Check summary table
	if !strings.Contains(result, "| Exchange | 2 | 1/2 | 2/2 |") {
		t.Error("incorrect summary row")
	}

	// Check detail table
	if !strings.Contains(result, "| `GetFoo` | `GET /foo` | Y | Y | |") {
		t.Error("missing GetFoo detail row")
	}
	if !strings.Contains(result, "| `CreateBar` | `POST /bar` | — | Y | |") {
		t.Error("missing CreateBar detail row")
	}

	// Check WS section
	if !strings.Contains(result, "## WebSocket Channels") {
		t.Error("missing WS section")
	}
	if !strings.Contains(result, "| `ticker` | Y | Y | |") {
		t.Error("missing ticker WS row")
	}
}
