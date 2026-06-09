//go:build spec_validation

package gokalshi

import (
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAsyncAPIWSChannelCoverage(t *testing.T) {
	resp, err := http.Get("https://docs.kalshi.com/asyncapi.yaml")
	require.NoError(t, err, "fetch AsyncAPI spec")
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode, "AsyncAPI spec HTTP status")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var spec struct {
		Channels map[string]struct {
			Address  string `yaml:"address"`
			Messages map[string]struct {
				Ref string `yaml:"$ref"`
			} `yaml:"messages"`
		} `yaml:"channels"`
		Components struct {
			Messages map[string]struct {
				Name string `yaml:"name"`
			} `yaml:"messages"`
		} `yaml:"components"`
	}
	require.NoError(t, yaml.Unmarshal(body, &spec))

	specChannels := make(map[string]bool)
	specMsgTypes := make(map[string]bool)

	for channelKey, channelDef := range spec.Channels {
		if channelKey == "root" || channelKey == "control_frames" {
			continue
		}
		address := channelDef.Address
		if address == "" {
			address = channelKey
		}
		specChannels[address] = true

		for _, msgRef := range channelDef.Messages {
			if msgRef.Ref == "" {
				continue
			}
			// Extract message name from $ref like "#/components/messages/OrderbookSnapshot"
			parts := strings.Split(msgRef.Ref, "/")
			msgName := parts[len(parts)-1]
			if resolved, ok := spec.Components.Messages[msgName]; ok && resolved.Name != "" {
				specMsgTypes[resolved.Name] = true
			} else {
				specMsgTypes[msgName] = true
			}
		}
	}

	// Verify every spec message type exists in MsgTypeToChannel.
	ourTypes := make(map[string]bool)
	for msgType := range MsgTypeToChannel {
		ourTypes[string(msgType)] = true
	}
	var missingTypes []string
	for msgType := range specMsgTypes {
		if !ourTypes[msgType] {
			missingTypes = append(missingTypes, msgType)
		}
	}

	// Verify every spec channel is referenced in MsgTypeToChannel values.
	ourChannels := make(map[string]bool)
	for _, channels := range MsgTypeToChannel {
		for _, ch := range channels {
			ourChannels[ch] = true
		}
	}
	var missingChannels []string
	for ch := range specChannels {
		if !ourChannels[ch] {
			missingChannels = append(missingChannels, ch)
		}
	}

	t.Logf("AsyncAPI channels: %d", len(specChannels))
	t.Logf("AsyncAPI message types: %d", len(specMsgTypes))
	t.Logf("MsgTypeToChannel entries: %d", len(MsgTypeToChannel))

	assert.Empty(t, missingTypes, "message types in spec but missing from MsgTypeToChannel")
	assert.Empty(t, missingChannels, "channels in spec but not referenced in MsgTypeToChannel")
}

// msgTypeToGoStruct maps WS message "name" → Go struct instance for reflection.
var msgTypeToGoStruct = map[string]any{
	"orderbook_snapshot":            OrderbookSnapshotData{},
	"orderbook_delta":               OrderbookDeltaData{},
	"ticker":                        TickerData{},
	"trade":                         TradeData{},
	"fill":                          FillData{},
	"market_position":               MarketPositionData{},
	"market_lifecycle_v2":           MarketLifecycleV2Data{},
	"event_lifecycle":               EventLifecycleData{},
	"multivariate_lookup":           MultivariateLookupData{},
	"user_order":                    UserOrderData{},
	"order_group_updates":           OrderGroupUpdateData{},
	"rfq_created":                   RFQCreatedData{},
	"rfq_deleted":                   RFQDeletedData{},
	"quote_created":                 QuoteCreatedData{},
	"quote_accepted":                QuoteAcceptedData{},
	"quote_executed":                QuoteExecutedData{},
	"event_fee_update":              EventFeeUpdateData{},
	"cfbenchmarks_value":            CfbenchmarksValueData{},
	"cfbenchmarks_value_indexlist":  CfbenchmarksValueIndexlistData{},
}

// structJSONFields returns the set of JSON field names for a Go struct type.
func structJSONFields(v any) map[string]bool {
	t := reflect.TypeOf(v)
	fields := make(map[string]bool)
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.SplitN(tag, ",", 2)[0]
		fields[name] = true
	}
	return fields
}

func TestAsyncAPIWSPayloadFieldCoverage(t *testing.T) {
	resp, err := http.Get("https://docs.kalshi.com/asyncapi.yaml")
	require.NoError(t, err, "fetch AsyncAPI spec")
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode, "AsyncAPI spec HTTP status")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Parse spec with enough depth to extract msg.properties field names.
	var spec struct {
		Channels map[string]struct {
			Messages map[string]struct {
				Ref string `yaml:"$ref"`
			} `yaml:"messages"`
		} `yaml:"channels"`
		Components struct {
			Messages map[string]struct {
				Name    string `yaml:"name"`
				Payload struct {
					Ref string `yaml:"$ref"`
				} `yaml:"payload"`
			} `yaml:"messages"`
			Schemas map[string]struct {
				AllOf []struct {
					Ref string `yaml:"$ref"`
				} `yaml:"allOf"`
				Properties map[string]struct {
					Properties map[string]any `yaml:"properties"`
					Required   []string       `yaml:"required"`
				} `yaml:"properties"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}
	require.NoError(t, yaml.Unmarshal(body, &spec))

	// Build message name → spec field names mapping.
	specFields := make(map[string][]string) // msg name → sorted field names

	for channelKey := range spec.Channels {
		if channelKey == "root" || channelKey == "control_frames" {
			continue
		}
		for _, msgRef := range spec.Channels[channelKey].Messages {
			if msgRef.Ref == "" {
				continue
			}
			parts := strings.Split(msgRef.Ref, "/")
			msgKey := parts[len(parts)-1]
			resolved, ok := spec.Components.Messages[msgKey]
			if !ok || resolved.Name == "" {
				continue
			}
			wsType := resolved.Name
			if _, done := specFields[wsType]; done {
				continue
			}

			// Resolve payload schema.
			payloadRef := resolved.Payload.Ref
			if payloadRef == "" {
				continue
			}
			schemaParts := strings.Split(payloadRef, "/")
			schemaName := schemaParts[len(schemaParts)-1]
			schema, ok := spec.Components.Schemas[schemaName]
			if !ok {
				// Try allOf (multivariate_market_lifecycle).
				continue
			}

			// Skip allOf schemas (type aliases).
			if len(schema.AllOf) > 0 {
				continue
			}

			msgProp, ok := schema.Properties["msg"]
			if !ok || msgProp.Properties == nil {
				continue
			}

			var fieldNames []string
			for fn := range msgProp.Properties {
				fieldNames = append(fieldNames, fn)
			}
			sort.Strings(fieldNames)
			specFields[wsType] = fieldNames
		}
	}

	// Compare spec fields against Go struct JSON tags.
	var allMissing []string
	for wsType, expectedFields := range specFields {
		goStruct, ok := msgTypeToGoStruct[wsType]
		if !ok {
			t.Logf("WARNING: no Go struct mapping for %q", wsType)
			continue
		}
		goFields := structJSONFields(goStruct)

		for _, field := range expectedFields {
			if !goFields[field] {
				allMissing = append(allMissing, wsType+"."+field)
			}
		}
	}
	sort.Strings(allMissing)

	t.Logf("Checked payload fields for %d message types", len(specFields))
	for _, m := range allMissing {
		t.Logf("  missing: %s", m)
	}
	assert.Empty(t, allMissing, "spec fields missing from Go structs")
}
