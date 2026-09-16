package gokalshi

import (
	"strings"
)

// GET /account/endpoint_costs returns the token cost of every endpoint whose
// cost differs from the default. Its paths carry placeholders for path
// parameters, and the syntax is not the OpenAPI spec's: the live API returns
// Gin-style ":order_id" and "*endpoint", not "{order_id}".
//
// Matching those against a real request path used to be done by converting the
// whole path to a regex, which only recognised "{param}". Every ":param" entry
// compiled to a literal that could never match, so nine of seventeen live
// entries were silently discarded and billed at the default cost instead —
// cancels at 10 tokens rather than 2. Nothing surfaced, because falling back to
// the default is indistinguishable from an endpoint that has no override.
//
// Matching is now segment-wise, so it does not depend on regex escaping, and
// every placeholder syntax in circulation is recognised. A syntax nobody has
// seen yet would still not match — so unmatched entries are reported rather
// than dropped, and TestEndpointCostsMatchLiveTable asserts every entry the
// live API returns is usable.

// costRoute is one entry from the endpoint-costs table, prepared for matching.
type costRoute struct {
	method   string
	segments []string // path split on "/"; placeholders kept verbatim
	cost     float64
	literals int    // number of non-placeholder segments, for specificity
	rawPath  string // as returned by the API, for diagnostics
}

// newCostRoute prepares a table entry for matching.
func newCostRoute(method, path string, cost float64) costRoute {
	if !strings.HasPrefix(path, "/trade-api") {
		path = "/trade-api/v2" + path
	}
	segments := splitPath(path)

	literals := 0
	for _, seg := range segments {
		if !isPathPlaceholder(seg) {
			literals++
		}
	}
	return costRoute{
		method:   strings.ToUpper(method),
		segments: segments,
		cost:     cost,
		literals: literals,
		rawPath:  path,
	}
}

// isPathPlaceholder reports whether a path segment stands in for a parameter
// rather than matching literally.
//
// Deliberately generous: Kalshi has served ":order_id" and "*endpoint" while
// the OpenAPI spec writes "{order_id}", and there is no guarantee a third form
// will not appear. Recognising a form that never shows up costs nothing;
// failing to recognise one silently mis-bills every call to that endpoint.
func isPathPlaceholder(segment string) bool {
	if segment == "" {
		return false
	}
	switch segment[0] {
	case ':', '*', '$':
		return true
	}
	if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
		return true
	}
	if strings.HasPrefix(segment, "<") && strings.HasSuffix(segment, ">") {
		return true
	}
	return false
}

// isTrailingWildcard reports whether a placeholder consumes the rest of the
// path rather than a single segment, as "*endpoint" does.
func isTrailingWildcard(segment string) bool {
	return strings.HasPrefix(segment, "*")
}

// matches reports whether this route describes the given request.
func (r costRoute) matches(method, path string) bool {
	if r.method != strings.ToUpper(method) {
		return false
	}
	return segmentsMatch(r.segments, splitPath(path))
}

func segmentsMatch(pattern, actual []string) bool {
	for i, want := range pattern {
		// A trailing wildcard swallows every remaining segment, so it needs at
		// least one to swallow.
		if isTrailingWildcard(want) {
			return len(actual) > i
		}
		if i >= len(actual) {
			return false
		}
		if isPathPlaceholder(want) {
			continue
		}
		if want != actual[i] {
			return false
		}
	}
	return len(actual) == len(pattern)
}

// splitPath splits a URL path into non-empty segments.
func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// suspectUnmarkedPlaceholder reports whether a segment looks like a path
// parameter that this code failed to recognise as one.
//
// isPathPlaceholder covers every marker syntax seen in the wild, but it cannot
// cover one that has not appeared yet. If Kalshi ever serves a bare "order_id"
// where it now serves ":order_id", that segment would be treated as a literal
// and the entry would quietly stop matching — the exact failure this package
// already suffered once, undetected, across nine endpoints.
//
// An unmarked parameter is hard to distinguish from a literal in general, but
// the overwhelmingly common shape is an identifier: "id", or something ending
// "_id". Flagging those is enough to turn a silent regression into a log line.
func suspectUnmarkedPlaceholder(segment string) bool {
	if isPathPlaceholder(segment) {
		return false
	}
	return segment == "id" || strings.HasSuffix(segment, "_id")
}

// suspiciousSegments returns segments that look like unrecognised placeholders.
func (r costRoute) suspiciousSegments() []string {
	var out []string
	for _, seg := range r.segments {
		if suspectUnmarkedPlaceholder(seg) {
			out = append(out, seg)
		}
	}
	return out
}
