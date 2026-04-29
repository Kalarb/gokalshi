package gokalshi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryBuilder_EmptyBuild(t *testing.T) {
	q := NewQuery().Build()
	assert.Empty(t, q)
}

func TestQueryBuilder_String_NonEmpty(t *testing.T) {
	q := NewQuery().String("key", "value").Build()
	assert.Equal(t, "value", q["key"])
}

func TestQueryBuilder_String_Empty(t *testing.T) {
	q := NewQuery().String("key", "").Build()
	_, exists := q["key"]
	assert.False(t, exists, "empty string should not be added")
}

func TestQueryBuilder_Int_Positive(t *testing.T) {
	q := NewQuery().Int("limit", 10).Build()
	assert.Equal(t, "10", q["limit"])
}

func TestQueryBuilder_Int_Zero(t *testing.T) {
	q := NewQuery().Int("limit", 0).Build()
	_, exists := q["limit"]
	assert.False(t, exists, "zero int should not be added")
}

func TestQueryBuilder_Int64_Positive(t *testing.T) {
	q := NewQuery().Int64("ts", 1700000000).Build()
	assert.Equal(t, "1700000000", q["ts"])
}

func TestQueryBuilder_Int64_Zero(t *testing.T) {
	q := NewQuery().Int64("ts", 0).Build()
	_, exists := q["ts"]
	assert.False(t, exists, "zero int64 should not be added")
}

func TestQueryBuilder_Bool_True(t *testing.T) {
	q := NewQuery().Bool("active", true).Build()
	assert.Equal(t, "true", q["active"])
}

func TestQueryBuilder_Bool_False(t *testing.T) {
	q := NewQuery().Bool("active", false).Build()
	_, exists := q["active"]
	assert.False(t, exists, "false bool should not be added")
}

func TestQueryBuilder_Chaining(t *testing.T) {
	q := NewQuery().
		String("ticker", "KXBTC").
		Int("limit", 5).
		Int64("min_ts", 100).
		Bool("nested", true).
		String("empty", "").
		Int("zero", 0).
		Bool("nope", false).
		Build()

	assert.Len(t, q, 4)
	assert.Equal(t, "KXBTC", q["ticker"])
	assert.Equal(t, "5", q["limit"])
	assert.Equal(t, "100", q["min_ts"])
	assert.Equal(t, "true", q["nested"])
}
