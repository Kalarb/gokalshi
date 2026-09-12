package gokalshi

import "strconv"

// QueryBuilder constructs query parameter maps with zero-value filtering.
type QueryBuilder struct {
	m map[string]string
}

// NewQuery creates a new QueryBuilder.
func NewQuery() *QueryBuilder {
	return &QueryBuilder{m: make(map[string]string)}
}

// String adds a string parameter if non-empty.
func (q *QueryBuilder) String(key, val string) *QueryBuilder {
	if val != "" {
		q.m[key] = val
	}
	return q
}

// IntPtr adds an int parameter when set, including zero and negatives.
//
// Int and Int64 treat any non-positive value as "unset", which is right for
// limits and cursors but wrong for parameters whose meaningful values include
// 0 and -1 — exchange_index being the case that matters: 0 selects the
// event-contract instance and -1 asks Kalshi to auto-route by market ticker,
// and both were silently dropped.
func (q *QueryBuilder) IntPtr(key string, val *int) *QueryBuilder {
	if val != nil {
		q.m[key] = strconv.Itoa(*val)
	}
	return q
}

// Int adds an int parameter if positive.
func (q *QueryBuilder) Int(key string, val int) *QueryBuilder {
	if val > 0 {
		q.m[key] = strconv.Itoa(val)
	}
	return q
}

// Int64 adds an int64 parameter if positive.
func (q *QueryBuilder) Int64(key string, val int64) *QueryBuilder {
	if val > 0 {
		q.m[key] = strconv.FormatInt(val, 10)
	}
	return q
}

// Bool adds a boolean parameter if true.
func (q *QueryBuilder) Bool(key string, val bool) *QueryBuilder {
	if val {
		q.m[key] = "true"
	}
	return q
}

// Build returns the constructed query parameter map.
func (q *QueryBuilder) Build() map[string]string {
	return q.m
}
