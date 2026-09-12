//go:build spec_validation

package gokalshi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The base URLs were hand-written while everything else came from the spec, so
// nothing caught it when Kalshi moved the WebSocket endpoint to its own
// external-api-ws host: the SDK kept dialling external-api, which answers 404,
// and every WS connection failed. Endpoint and schema coverage cannot see this
// — the spec declares its servers, so compare those too.

func TestOpenAPIServerURLs(t *testing.T) {
	body := loadVendoredSpec(t, "openapi")

	var spec struct {
		Servers []struct {
			URL string `yaml:"url"`
		} `yaml:"servers"`
	}
	require.NoError(t, yaml.Unmarshal(body, &spec))
	require.NotEmpty(t, spec.Servers, "spec declared no servers")

	var hosts []string
	for _, s := range spec.Servers {
		hosts = append(hosts, strings.TrimSuffix(s.URL, "/trade-api/v2"))
	}

	assert.Contains(t, hosts, prodHTTPBase, "prodHTTPBase is not a server the spec declares")
	assert.Contains(t, hosts, demoHTTPBase, "demoHTTPBase is not a server the spec declares")
}

func TestAsyncAPIServerURLs(t *testing.T) {
	body := loadVendoredSpec(t, "asyncapi")

	var spec struct {
		Servers map[string]struct {
			Host     string `yaml:"host"`
			Pathname string `yaml:"pathname"`
			Protocol string `yaml:"protocol"`
		} `yaml:"servers"`
	}
	require.NoError(t, yaml.Unmarshal(body, &spec))
	require.NotEmpty(t, spec.Servers, "spec declared no servers")

	var prod struct {
		host, pathname, protocol string
	}
	for _, s := range spec.Servers {
		prod.host, prod.pathname, prod.protocol = s.Host, s.Pathname, s.Protocol
		break
	}

	assert.Equal(t, prod.protocol+"://"+prod.host, prodWSBase,
		"prodWSBase does not match the AsyncAPI production server")
	assert.Equal(t, prod.pathname, wsPathSuffix,
		"wsPathSuffix does not match the AsyncAPI server pathname")

	// The spec only declares production. Demo mirrors it on the .co domain, so
	// hold the two to the same shape rather than leaving demo unchecked.
	assert.Equal(t, strings.Replace(prodWSBase, ".kalshi.com", ".demo.kalshi.co", 1), demoWSBase,
		"demoWSBase is not the demo form of prodWSBase")
}
