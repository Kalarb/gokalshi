// generate_ws_types fetches the Kalshi AsyncAPI spec and generates Go types
// for all WebSocket messages (commands, responses, and data messages).
//
// Usage:
//
//	go run ./tools/generate_ws_types
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
	"sort"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

const asyncAPISpecURL = "https://docs.kalshi.com/asyncapi.yaml"

// ---------------------------------------------------------------------------
// Backward-compatible naming maps
// ---------------------------------------------------------------------------

// msgTypeToStructName maps WS message "name" → Go struct name for data messages.
var msgTypeToStructName = map[string]string{
	"orderbook_snapshot":             "OrderbookSnapshotData",
	"orderbook_delta":                "OrderbookDeltaData",
	"ticker":                         "TickerData",
	"trade":                          "TradeData",
	"fill":                           "FillData",
	"market_position":                "MarketPositionData",
	"market_lifecycle_v2":            "MarketLifecycleV2Data",
	"event_lifecycle":                "EventLifecycleData",
	"multivariate_market_lifecycle":  "MultivariateMarketLifecycleData",
	"multivariate_lookup":            "MultivariateLookupData",
	"user_order":                     "UserOrderData",
	"order_group_updates":            "OrderGroupUpdateData",
	"rfq_created":                    "RFQCreatedData",
	"rfq_deleted":                    "RFQDeletedData",
	"quote_created":                  "QuoteCreatedData",
	"quote_accepted":                 "QuoteAcceptedData",
	"quote_executed":                 "QuoteExecutedData",
	"event_fee_update":               "EventFeeUpdateData",
	"cfbenchmarks_value":             "CfbenchmarksValueData",
	"cfbenchmarks_value_indexlist":   "CfbenchmarksValueIndexlistData",
}

// msgTypeToConstName maps WS message "name" → Go constant name.
var msgTypeToConstName = map[string]string{
	"orderbook_snapshot":             "WSMsgOrderbookSnapshot",
	"orderbook_delta":                "WSMsgOrderbookDelta",
	"ticker":                         "WSMsgTicker",
	"trade":                          "WSMsgTrade",
	"fill":                           "WSMsgFill",
	"market_position":                "WSMsgMarketPosition",
	"market_lifecycle_v2":            "WSMsgMarketLifecycleV2",
	"event_lifecycle":                "WSMsgEventLifecycle",
	"multivariate_market_lifecycle":  "WSMsgMultivariateMarketLifecycle",
	"multivariate_lookup":            "WSMsgMultivariateLookup",
	"user_order":                     "WSMsgUserOrder",
	"order_group_updates":            "WSMsgOrderGroupUpdates",
	"rfq_created":                    "WSMsgRFQCreated",
	"rfq_deleted":                    "WSMsgRFQDeleted",
	"quote_created":                  "WSMsgQuoteCreated",
	"quote_accepted":                 "WSMsgQuoteAccepted",
	"quote_executed":                 "WSMsgQuoteExecuted",
	"event_fee_update":               "WSMsgEventFeeUpdate",
	"cfbenchmarks_value":             "WSMsgCfbenchmarksValue",
	"cfbenchmarks_value_indexlist":   "WSMsgCfbenchmarksValueIndexlist",
}

// refTypeOverrides maps schema $ref names to Go types for primitive/enum refs.
var refTypeOverrides = map[string]string{
	"marketSide":      "Side",
	"bookSide":        "BookSide",
	"orderAction":     "Action",
	"marketTicker":    "string",
	"marketId":        "string",
	"commandId":       "int",
	"subscriptionId":  "int",
	"sequenceNumber":  "int",
}

// wsFieldTypeOverrides maps (struct, field) → Go type for enum overrides.
var wsFieldTypeOverrides = map[[2]string]string{
	// MarketLifecycleEventType
	{"MarketLifecycleV2Data", "event_type"}: "MarketLifecycleEventType",

	// OrderGroupEventType
	{"OrderGroupUpdateData", "event_type"}: "OrderGroupEventType",

	// CollateralReturnType
	{"EventLifecycleData", "collateral_return_type"}: "CollateralReturnType",

	// OrderStatus
	{"UserOrderData", "status"}: "OrderStatus",

	// STPType
	{"UserOrderData", "self_trade_prevention_type"}: "STPType",

	// Side for quote accepted
	{"QuoteAcceptedData", "accepted_side"}: "Side",

	// Side for multivariate lookup selected markets
	{"MultivariateLookupSelectedMarket", "side"}: "Side",

	// Fee type nullable enum
	{"EventFeeUpdateData", "fee_type_override"}: "FeeType",

	// WSUpdateAction for commands
	{"UpdateSubParams", "action"}: "WSUpdateAction",
}

// nullableOverrides forces pointer types for specific fields.
var nullableOverrides = map[[2]string]bool{
	{"MarketLifecycleV2Data", "is_deactivated"}: true,
}

// inlineSubObjectNames maps (parent struct, field) → Go struct name for inline objects.
var inlineSubObjectNames = map[[2]string]string{
	{"MarketLifecycleV2Data", "additional_metadata"}: "MarketLifecycleAdditionalMetadata",
	{"MultivariateLookupData", "selected_markets"}:   "MultivariateLookupSelectedMarket",
	{"RFQCreatedData", "mve_selected_legs"}:          "MveSelectedLeg",
}

// skipInlineGeneration lists inline sub-object struct names that already exist
// in types_generated.go and should NOT be re-generated.
var skipInlineGeneration = map[string]bool{
	"MveSelectedLeg": true,
}

// sharedSchemaNames maps schema ref names to Go struct names for shared schemas.
var sharedSchemaNames = map[string]string{
	"cfbenchmarksAvgData": "CfbenchmarksAvgData",
}

// skipChannels are channels that don't produce data message types.
var skipChannels = map[string]bool{
	"root":           true,
	"control_frames": true,
}

// ---------------------------------------------------------------------------
// Spec types
// ---------------------------------------------------------------------------

// AsyncAPISpec is a minimal representation of the AsyncAPI 3.0 spec.
type AsyncAPISpec struct {
	Channels   map[string]*Channel `yaml:"channels"`
	Components struct {
		Messages map[string]*Message `yaml:"messages"`
		Schemas  map[string]*Schema  `yaml:"schemas"`
	} `yaml:"components"`
	XErrorCodes *ErrorCodes `yaml:"x-error-codes"`
}

// Channel represents an AsyncAPI channel.
type Channel struct {
	Address  string              `yaml:"address"`
	Messages map[string]*MsgRef  `yaml:"messages"`
}

// MsgRef holds a $ref to a message.
type MsgRef struct {
	Ref string `yaml:"$ref"`
}

// Message represents an AsyncAPI message definition.
type Message struct {
	Name    string  `yaml:"name"`
	Payload *Schema `yaml:"payload"`
}

// Schema represents a JSON Schema used in AsyncAPI.
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
	Const                any                `yaml:"const"`
	AdditionalProperties any                `yaml:"additionalProperties"`
}

// ErrorCodes represents the x-error-codes extension.
type ErrorCodes struct {
	Codes []ErrorCode `yaml:"codes"`
}

// ErrorCode represents a single WS error code.
type ErrorCode struct {
	Code        int    `yaml:"code"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// dataMsg holds parsed info for a data message type.
type dataMsg struct {
	Name       string   // WS type value (e.g., "trade")
	StructName string   // Go struct name (e.g., "TradeData")
	MsgSchema  *Schema  // The "msg" sub-schema
	Channels   []string // Channel addresses this message appears on
}

// inlineSub holds info for an inline sub-object to generate.
type inlineSub struct {
	StructName string
	Schema     *Schema
}

func main() {
	spec, err := fetchSpec()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch spec: %v\n", err)
		os.Exit(1)
	}
	schemas := spec.Components.Schemas

	// Collect data messages from non-control channels.
	dataMsgs, msgToChannels := collectDataMessages(spec)
	fmt.Printf("Loaded %d data message types from %d channels\n", len(dataMsgs), countDataChannels(spec))

	// Collect shared schemas that need standalone structs.
	sharedStructs := collectSharedSchemas(schemas)

	// Collect inline sub-objects from data messages.
	inlineSubs := collectInlineSubObjects(dataMsgs, schemas)

	// Build MsgTypeToChannel map.
	msgTypeToChannel := buildMsgTypeToChannel(msgToChannels)

	// Collect command param schemas.
	subParams := extractParamsSchema(schemas, "subscribeCommandPayload")
	unsubParams := extractParamsSchema(schemas, "unsubscribeCommandPayload")
	updateSubParams := mergeUpdateSubParams(schemas)

	// Collect response body schemas.
	subscribedBody := extractMsgSchema(schemas, "subscribedResponsePayload")
	okBody := extractMsgSchema(schemas, "okResponsePayload")
	errorBody := extractMsgSchema(schemas, "errorResponsePayload")

	// Generate code.
	var buf bytes.Buffer
	buf.WriteString(fileHeader())
	buf.WriteString("package gokalshi\n\n")

	// Section: Outgoing commands
	buf.WriteString("// ---------------------------------------------------------------------------\n")
	buf.WriteString("// Outgoing commands\n")
	buf.WriteString("// ---------------------------------------------------------------------------\n\n")

	buf.WriteString("// WSCommand is the generic envelope for outgoing WebSocket commands.\n")
	buf.WriteString("// Cmd values: see WSCmd constants.\n")
	buf.WriteString("type WSCommand struct {\n")
	buf.WriteString("\tID     int   `json:\"id\"`\n")
	buf.WriteString("\tCmd    WSCmd `json:\"cmd\"`\n")
	buf.WriteString("\tParams any   `json:\"params,omitempty\"` // nil for list_subscriptions\n")
	buf.WriteString("}\n\n")

	writeStruct(&buf, "SubscribeParams", "SubscribeParams are the parameters for a \"subscribe\" command.", subParams, schemas)
	writeStruct(&buf, "UnsubscribeParams", "UnsubscribeParams are the parameters for an \"unsubscribe\" command.", unsubParams, schemas)
	writeStruct(&buf, "UpdateSubParams", "UpdateSubParams are the parameters for an \"update_subscription\" command.", updateSubParams, schemas)

	// Section: Control responses
	buf.WriteString("// ---------------------------------------------------------------------------\n")
	buf.WriteString("// Incoming response msg bodies (parsed from WSMessage.Msg)\n")
	buf.WriteString("// ---------------------------------------------------------------------------\n\n")

	writeStruct(&buf, "SubscribedBody", "SubscribedBody is the msg body for a \"subscribed\" response.", subscribedBody, schemas)
	writeStruct(&buf, "OkBody", "OkBody is the msg body for an \"ok\" response.", okBody, schemas)
	writeStruct(&buf, "WSErrorBody", "WSErrorBody is the msg body for an \"error\" response.", errorBody, schemas)

	// Section: WS error codes
	if spec.XErrorCodes != nil && len(spec.XErrorCodes.Codes) > 0 {
		buf.WriteString("// ---------------------------------------------------------------------------\n")
		buf.WriteString("// WS error codes (from Kalshi API spec)\n")
		buf.WriteString("// ---------------------------------------------------------------------------\n\n")
		writeErrorCodes(&buf, spec.XErrorCodes.Codes)
	}

	// Section: Shared sub-types
	if len(sharedStructs) > 0 {
		buf.WriteString("// ---------------------------------------------------------------------------\n")
		buf.WriteString("// Shared sub-types\n")
		buf.WriteString("// ---------------------------------------------------------------------------\n\n")
		for _, name := range sortedKeys(sharedStructs) {
			s := sharedStructs[name]
			goName := sharedSchemaNames[name]
			desc := fmt.Sprintf("%s is a shared sub-type used by multiple WS messages.", goName)
			if s.Description != "" {
				desc = goName + " " + cleanDescription(s.Description)
			}
			writeStruct(&buf, goName, desc, s, schemas)
		}
	}

	// Section: Inline sub-types
	if len(inlineSubs) > 0 {
		buf.WriteString("// ---------------------------------------------------------------------------\n")
		buf.WriteString("// Inline sub-types\n")
		buf.WriteString("// ---------------------------------------------------------------------------\n\n")
		// Deduplicate, sort by name, and skip types that exist in types_generated.go.
		seen := make(map[string]inlineSub)
		for _, sub := range inlineSubs {
			if skipInlineGeneration[sub.StructName] {
				continue
			}
			if _, exists := seen[sub.StructName]; !exists {
				seen[sub.StructName] = sub
			}
		}
		sortedSubNames := sortedKeys(seen)
		for _, name := range sortedSubNames {
			sub := seen[name]
			desc := fmt.Sprintf("%s is a nested object type within a WS data message.", sub.StructName)
			writeStruct(&buf, sub.StructName, desc, sub.Schema, schemas)
		}
	}

	// Section: Channel data messages
	buf.WriteString("// ---------------------------------------------------------------------------\n")
	buf.WriteString("// Incoming channel data messages\n")
	buf.WriteString("// ---------------------------------------------------------------------------\n\n")

	// Sort by struct name for deterministic output.
	sortedMsgs := make([]dataMsg, len(dataMsgs))
	copy(sortedMsgs, dataMsgs)
	sort.Slice(sortedMsgs, func(i, j int) bool {
		return sortedMsgs[i].StructName < sortedMsgs[j].StructName
	})

	for _, dm := range sortedMsgs {
		// Skip allOf alias types (handled below).
		if dm.MsgSchema == nil {
			continue
		}
		desc := fmt.Sprintf("%s is the msg body for a %q message.", dm.StructName, dm.Name)
		writeStruct(&buf, dm.StructName, desc, dm.MsgSchema, schemas)
	}

	// Type aliases (multivariate_market_lifecycle = MarketLifecycleV2Data).
	for _, dm := range sortedMsgs {
		if dm.MsgSchema == nil && dm.StructName == "MultivariateMarketLifecycleData" {
			buf.WriteString("// MultivariateMarketLifecycleData has the same shape as MarketLifecycleV2Data.\n")
			buf.WriteString("type MultivariateMarketLifecycleData = MarketLifecycleV2Data\n\n")
		}
	}

	// Section: MsgTypeToChannel
	buf.WriteString("// ---------------------------------------------------------------------------\n")
	buf.WriteString("// Channel mapping\n")
	buf.WriteString("// ---------------------------------------------------------------------------\n\n")
	writeMsgTypeToChannel(&buf, msgTypeToChannel)

	// Write output file.
	outDir := findPackageRoot()
	writeFile(outDir, "ws_messages_generated.go", buf.String())
	fmt.Println("Done!")
}

// ---------------------------------------------------------------------------
// Spec fetching
// ---------------------------------------------------------------------------

func fetchSpec() (*AsyncAPISpec, error) {
	fmt.Printf("Fetching %s...\n", asyncAPISpecURL)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(asyncAPISpecURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, asyncAPISpecURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var spec AsyncAPISpec
	if err := yaml.Unmarshal(body, &spec); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	return &spec, nil
}

// ---------------------------------------------------------------------------
// Data message collection
// ---------------------------------------------------------------------------

func collectDataMessages(spec *AsyncAPISpec) ([]dataMsg, map[string][]string) {
	msgToChannels := make(map[string][]string)
	msgSchemas := make(map[string]*Schema)

	for channelKey, channelDef := range spec.Channels {
		if skipChannels[channelKey] {
			continue
		}
		address := channelDef.Address
		if address == "" {
			address = channelKey
		}

		for _, msgRef := range channelDef.Messages {
			if msgRef.Ref == "" {
				continue
			}
			msgName := refToName(msgRef.Ref)
			resolved := spec.Components.Messages[msgName]
			if resolved == nil || resolved.Name == "" {
				continue
			}
			wsType := resolved.Name
			msgToChannels[wsType] = appendUnique(msgToChannels[wsType], address)

			// Resolve payload schema.
			if _, ok := msgSchemas[wsType]; ok {
				continue // Already resolved.
			}
			payloadSchema := resolvePayloadSchema(resolved, spec.Components.Schemas)
			if payloadSchema == nil {
				continue
			}
			msgSchema := extractMsgFromPayload(payloadSchema, spec.Components.Schemas)
			msgSchemas[wsType] = msgSchema
		}
	}

	var result []dataMsg
	for wsType, channels := range msgToChannels {
		sort.Strings(channels)
		structName, ok := msgTypeToStructName[wsType]
		if !ok {
			structName = toGoFieldName(wsType) + "Data"
			fmt.Fprintf(os.Stderr, "WARNING: unknown message type %q, using struct name %s\n", wsType, structName)
		}
		result = append(result, dataMsg{
			Name:       wsType,
			StructName: structName,
			MsgSchema:  msgSchemas[wsType], // nil for allOf aliases
			Channels:   channels,
		})
	}
	return result, msgToChannels
}

func resolvePayloadSchema(msg *Message, schemas map[string]*Schema) *Schema {
	if msg.Payload == nil {
		return nil
	}
	if msg.Payload.Ref != "" {
		name := refToName(msg.Payload.Ref)
		return schemas[name]
	}
	return msg.Payload
}

func extractMsgFromPayload(payload *Schema, schemas map[string]*Schema) *Schema {
	// Handle allOf (e.g., multivariateMarketLifecyclePayload).
	if len(payload.AllOf) > 0 {
		// If allOf only overrides "type" const, treat as alias — return nil.
		return nil
	}

	if payload.Properties == nil {
		return nil
	}
	msgProp := payload.Properties["msg"]
	if msgProp == nil {
		return nil
	}
	// Resolve $ref on msg itself.
	if msgProp.Ref != "" {
		name := refToName(msgProp.Ref)
		if s, ok := schemas[name]; ok {
			return s
		}
	}
	return msgProp
}

func countDataChannels(spec *AsyncAPISpec) int {
	count := 0
	for k := range spec.Channels {
		if !skipChannels[k] {
			count++
		}
	}
	return count
}

// ---------------------------------------------------------------------------
// Command/response schema extraction
// ---------------------------------------------------------------------------

func extractParamsSchema(schemas map[string]*Schema, payloadName string) *Schema {
	payload := schemas[payloadName]
	if payload == nil || payload.Properties == nil {
		return nil
	}
	params := payload.Properties["params"]
	if params == nil {
		return nil
	}
	if params.Ref != "" {
		name := refToName(params.Ref)
		if s, ok := schemas[name]; ok {
			return s
		}
	}
	return params
}

func extractMsgSchema(schemas map[string]*Schema, payloadName string) *Schema {
	payload := schemas[payloadName]
	if payload == nil || payload.Properties == nil {
		return nil
	}
	msg := payload.Properties["msg"]
	if msg == nil {
		return nil
	}
	if msg.Ref != "" {
		name := refToName(msg.Ref)
		if s, ok := schemas[name]; ok {
			return s
		}
	}
	return msg
}

// mergeUpdateSubParams merges the regular and cfbenchmarks update_subscription
// command variants into a single params schema with all possible fields.
func mergeUpdateSubParams(schemas map[string]*Schema) *Schema {
	regular := extractParamsSchema(schemas, "updateSubscriptionCommandPayload")
	cfb := extractParamsSchema(schemas, "cfbenchmarksUpdateSubscriptionCommandPayload")

	if regular == nil {
		return cfb
	}
	if cfb == nil {
		return regular
	}

	// Merge: start with regular, add any fields from cfb that aren't in regular.
	merged := &Schema{
		Type:       regular.Type,
		Required:   regular.Required,
		Properties: make(map[string]*Schema),
	}
	for k, v := range regular.Properties {
		merged.Properties[k] = v
	}
	for k, v := range cfb.Properties {
		if _, exists := merged.Properties[k]; !exists {
			merged.Properties[k] = v
		}
	}

	// Merge required arrays (union, deduplicated).
	reqSet := make(map[string]bool)
	for _, r := range regular.Required {
		reqSet[r] = true
	}
	for _, r := range cfb.Required {
		reqSet[r] = true
	}
	var mergedReq []string
	for r := range reqSet {
		mergedReq = append(mergedReq, r)
	}
	sort.Strings(mergedReq)
	merged.Required = mergedReq

	// The action field should accept all enum values from both variants.
	// The generator uses WSUpdateAction override, so enum values don't matter here.

	return merged
}

// ---------------------------------------------------------------------------
// Shared and inline sub-object collection
// ---------------------------------------------------------------------------

func collectSharedSchemas(schemas map[string]*Schema) map[string]*Schema {
	result := make(map[string]*Schema)
	for name := range sharedSchemaNames {
		if s, ok := schemas[name]; ok {
			result[name] = s
		}
	}
	return result
}

func collectInlineSubObjects(dataMsgs []dataMsg, schemas map[string]*Schema) []inlineSub {
	var result []inlineSub
	seen := make(map[string]bool)

	for _, dm := range dataMsgs {
		if dm.MsgSchema == nil || dm.MsgSchema.Properties == nil {
			continue
		}
		for fieldName, prop := range dm.MsgSchema.Properties {
			key := [2]string{dm.StructName, fieldName}
			goName, ok := inlineSubObjectNames[key]
			if !ok {
				continue
			}
			if seen[goName] {
				continue
			}
			seen[goName] = true

			// The inline object is either the property itself (type: object)
			// or the items of an array property.
			var subSchema *Schema
			if prop.Type == "array" && prop.Items != nil {
				subSchema = prop.Items
				if subSchema.Ref != "" {
					// Points to a schema — don't generate inline, will be resolved by type.
					continue
				}
			} else if prop.Type == "object" && len(prop.Properties) > 0 {
				subSchema = prop
			}
			if subSchema != nil {
				result = append(result, inlineSub{StructName: goName, Schema: subSchema})
			}
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Code generation: structs
// ---------------------------------------------------------------------------

func writeStruct(buf *bytes.Buffer, structName, comment string, schema *Schema, allSchemas map[string]*Schema) {
	if schema == nil {
		return
	}

	buf.WriteString(fmt.Sprintf("// %s\n", comment))
	buf.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	if len(schema.Properties) == 0 {
		buf.WriteString("}\n\n")
		return
	}

	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	propNames := sortedKeys(schema.Properties)
	for _, propName := range propNames {
		prop := schema.Properties[propName]
		isRequired := requiredSet[propName]

		goType := resolveGoType(structName, propName, prop, allSchemas)

		// Apply nullable/pointer overrides.
		forceNullable := nullableOverrides[[2]string{structName, propName}]
		if (prop.Nullable || forceNullable) && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map[") && !strings.HasPrefix(goType, "*") {
			goType = "*" + goType
		}

		goFieldName := toGoFieldName(propName)

		jsonTag := propName
		if !isRequired {
			jsonTag += ",omitempty"
		}

		// Field description comment.
		if prop.Description != "" {
			desc := cleanDescription(prop.Description)
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

func resolveGoType(structName, fieldName string, prop *Schema, allSchemas map[string]*Schema) string {
	// Check field-level type override first.
	if override, ok := wsFieldTypeOverrides[[2]string{structName, fieldName}]; ok {
		return override
	}

	return resolveGoTypeInner(structName, fieldName, prop, allSchemas)
}

func resolveGoTypeInner(structName, fieldName string, prop *Schema, allSchemas map[string]*Schema) string {
	// Handle allOf wrapper (common for nullable refs).
	if len(prop.AllOf) == 1 {
		return resolveGoTypeInner(structName, fieldName, prop.AllOf[0], allSchemas)
	}

	// Handle $ref.
	if prop.Ref != "" {
		refName := refToName(prop.Ref)
		if override, ok := refTypeOverrides[refName]; ok {
			return override
		}
		if goName, ok := sharedSchemaNames[refName]; ok {
			return goName
		}
		// Resolve to see if it's a primitive.
		if resolved, ok := allSchemas[refName]; ok {
			return resolveGoTypeInner(structName, fieldName, resolved, allSchemas)
		}
		return "any"
	}

	// Handle inline object with properties → check if it has a named sub-struct.
	if prop.Type == "object" && len(prop.Properties) > 0 {
		key := [2]string{structName, fieldName}
		if goName, ok := inlineSubObjectNames[key]; ok {
			return "*" + goName
		}
		return "map[string]any"
	}

	switch prop.Type {
	case "string":
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
			// Check for inline sub-object array items.
			if prop.Items.Type == "object" && len(prop.Items.Properties) > 0 {
				key := [2]string{structName, fieldName}
				if goName, ok := inlineSubObjectNames[key]; ok {
					return "[]" + goName
				}
			}
			// Check for array of arrays (orderbook levels).
			if prop.Items.Type == "array" {
				if prop.Items.Items != nil {
					innerType := resolveGoTypeInner(structName, fieldName, prop.Items.Items, allSchemas)
					return "[][]" + innerType
				}
				return "[][]any"
			}
			itemType := resolveGoTypeInner(structName, fieldName, prop.Items, allSchemas)
			return "[]" + itemType
		}
		return "[]any"
	case "object":
		if ap, ok := prop.AdditionalProperties.(*Schema); ok && ap != nil {
			valType := resolveGoTypeInner(structName, fieldName, ap, allSchemas)
			return "map[string]" + valType
		}
		return "map[string]any"
	default:
		return "any"
	}
}

// ---------------------------------------------------------------------------
// Code generation: error codes
// ---------------------------------------------------------------------------

// errorCodeNames maps error code numbers to Go constant names for backward compat.
var errorCodeNames = map[int]string{
	1:  "WSErrUnableToProcess",
	2:  "WSErrParamsRequired",
	3:  "WSErrChannelsRequired",
	4:  "WSErrSIDsRequired",
	5:  "WSErrUnknownCommand",
	6:  "WSErrAlreadySubscribed",
	7:  "WSErrUnknownSID",
	8:  "WSErrUnknownChannel",
	9:  "WSErrAuthRequired",
	10: "WSErrChannelError",
	11: "WSErrInvalidParam",
	12: "WSErrExactlyOneSID",
	13: "WSErrUnsupportedAction",
	14: "WSErrMarketTickerReq",
	15: "WSErrActionRequired",
	16: "WSErrMarketNotFound",
	17: "WSErrInternal",
	18: "WSErrCommandTimeout",
	19: "WSErrShardFactorPositive",
	20: "WSErrShardFactorRequired",
	21: "WSErrShardKeyRange",
	22: "WSErrShardFactorTooLarge",
	23: "WSErrMatchIDsRequired",
	24: "WSErrIndexIDsRequired",
	25: "WSErrSubBufferOverflow",
}

func writeErrorCodes(buf *bytes.Buffer, codes []ErrorCode) {
	buf.WriteString("const (\n")
	for _, ec := range codes {
		constName, ok := errorCodeNames[ec.Code]
		if !ok {
			constName = "WSErr" + toGoFieldName(ec.Name)
			fmt.Fprintf(os.Stderr, "WARNING: unknown error code %d %q, using %s\n", ec.Code, ec.Name, constName)
		}
		buf.WriteString(fmt.Sprintf("\t%s = %d\n", constName, ec.Code))
	}
	buf.WriteString(")\n\n")
}

// ---------------------------------------------------------------------------
// Code generation: MsgTypeToChannel
// ---------------------------------------------------------------------------

func buildMsgTypeToChannel(msgToChannels map[string][]string) map[string][]string {
	// Sort channel lists.
	for k := range msgToChannels {
		sort.Strings(msgToChannels[k])
	}
	return msgToChannels
}

func writeMsgTypeToChannel(buf *bytes.Buffer, msgTypeToChannel map[string][]string) {
	buf.WriteString("// MsgTypeToChannel maps incoming WS message types to the channel(s) they can arrive on.\n")
	buf.WriteString("// Also used as a set of known message types — actual routing uses SID.\n")
	buf.WriteString("var MsgTypeToChannel = map[WSMessageType][]string{\n")

	// Sort by constant name for deterministic output.
	type entry struct {
		constName string
		channels  []string
	}
	var entries []entry
	for wsType, channels := range msgTypeToChannel {
		constName, ok := msgTypeToConstName[wsType]
		if !ok {
			constName = "WSMsg" + toGoFieldName(wsType)
			fmt.Fprintf(os.Stderr, "WARNING: unknown message type %q for MsgTypeToChannel, using %s\n", wsType, constName)
		}
		entries = append(entries, entry{constName: constName, channels: channels})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].constName < entries[j].constName
	})

	for _, e := range entries {
		channelStrs := make([]string, len(e.channels))
		for i, ch := range e.channels {
			channelStrs[i] = fmt.Sprintf("%q", ch)
		}
		buf.WriteString(fmt.Sprintf("\t%s: {%s},\n", e.constName, strings.Join(channelStrs, ", ")))
	}
	buf.WriteString("}\n")
}

// ---------------------------------------------------------------------------
// Helpers (duplicated from generate_types for self-contained tool)
// ---------------------------------------------------------------------------

func refToName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

// goFieldNameOverrides maps specific JSON field names to Go field names.
var goFieldNameOverrides = map[string]string{
	"sids": "SIDs",
}

func toGoFieldName(s string) string {
	if override, ok := goFieldNameOverrides[s]; ok {
		return override
	}
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		upper := strings.ToUpper(part)
		switch upper {
		case "ID", "URL", "API", "FP", "TS", "IP", "RFQ", "FCM", "MVE", "OHLC", "STP", "SID":
			result.WriteString(upper)
		default:
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			result.WriteString(string(runes))
		}
	}
	return result.String()
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var multiSpaceRe = regexp.MustCompile(`\s+`)

func cleanDescription(s string) string {
	s = strings.TrimSpace(s)
	lines := strings.SplitN(s, "\n", 2)
	s = strings.TrimSpace(lines[0])
	s = strings.TrimLeft(s, "# ")
	s = multiSpaceRe.ReplaceAllString(s, " ")
	return s
}

func fileHeader() string {
	return fmt.Sprintf("// Code generated by tools/generate_ws_types from %s — DO NOT EDIT.\n// Source: ws_messages_generated.go\n\n", asyncAPISpecURL)
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

func writeFile(dir, filename, content string) {
	formatted, err := format.Source([]byte(content))
	if err != nil {
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
	fmt.Printf("  Wrote %s (%d lines)\n", filename, lines)
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
