//go:build spec_validation

package gokalshi

import (
	"io"
	"net/http"
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
