package tooler

import (
	"testing"

	"golang.org/x/oauth2"

	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnection(t *testing.T) {
	address := domain.MustNewAddress("https", "cluster.example", 0)
	tokens := oauth2.StaticTokenSource(&oauth2.Token{})

	tests := []struct {
		name    string
		address domain.Address
		ca      []byte
		tokens  oauth2.TokenSource
		pass    bool
	}{
		{name: "Whole_Valid", address: address, ca: []byte("ca"), tokens: tokens, pass: true},
		{name: "UnstatedAddress_Invalid", ca: []byte("ca"), tokens: tokens},
		{name: "UnstatedCertificateAuthority_Invalid", address: address, tokens: tokens},
		{name: "UnstatedTokenSource_Invalid", address: address, ca: []byte("ca")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connection, err := NewConnection(tt.address, tt.ca, tt.tokens)

			if !tt.pass {
				assert.Error(t, err)
				assert.True(t, connection.IsZero())

				return
			}

			require.NoError(t, err)
			assert.False(t, connection.IsZero())
			assert.Equal(t, "https://cluster.example", connection.Address().String())
			assert.Equal(t, []byte("ca"), connection.CA())
			assert.NotNil(t, connection.TokenSource())
		})
	}
}

func TestConnectionZeroIsAmbient(t *testing.T) {
	assert.True(t, Connection{}.IsZero())
}
