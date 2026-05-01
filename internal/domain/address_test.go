package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		name           string
		raw            string
		pass           bool
		expectedScheme string
		expectedHost   string
		expectedPort   int
	}{
		{
			name:           "Hostname_Valid",
			raw:            "tcp://query-service:9000",
			pass:           true,
			expectedScheme: "tcp",
			expectedHost:   "query-service",
			expectedPort:   9000,
		},
		{
			name:           "IPv4_Valid",
			raw:            "postgres://127.0.0.1:5432",
			pass:           true,
			expectedScheme: "postgres",
			expectedHost:   "127.0.0.1",
			expectedPort:   5432,
		},
		{
			name:           "IPv6_Valid",
			raw:            "tcp://[::1]:9000",
			pass:           true,
			expectedScheme: "tcp",
			expectedHost:   "::1",
			expectedPort:   9000,
		},
		{
			name:           "Hostname_NoPort_Valid",
			raw:            "tcp://signoz",
			pass:           true,
			expectedScheme: "tcp",
			expectedHost:   "signoz",
			expectedPort:   0,
		},
		{
			name:           "IPv6_NoPort_Valid",
			raw:            "tcp://[::1]",
			pass:           true,
			expectedScheme: "tcp",
			expectedHost:   "::1",
			expectedPort:   0,
		},
		{
			name: "MissingScheme_Invalid",
			raw:  "localhost:9000",
			pass: false,
		},
		{
			name: "Malformed_Invalid",
			raw:  "://bad",
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address, err := ParseAddress(tt.raw)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedScheme, address.Scheme())
			assert.Equal(t, tt.expectedHost, address.Host())
			assert.Equal(t, tt.expectedPort, address.Port())
			assert.Equal(t, tt.raw, address.String())
		})
	}
}
