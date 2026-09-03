package tooler

import (
	"golang.org/x/oauth2"

	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

// Connection's TokenSource is behavior rather than data: it never serializes,
// and no secret outlives the call.
type Connection struct {
	address domain.Address

	ca []byte

	tokens oauth2.TokenSource
}

func NewConnection(address domain.Address, ca []byte, tokens oauth2.TokenSource) (Connection, error) {
	if address.Host() == "" {
		return Connection{}, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "failed to create connection: no address is stated")
	}

	if len(ca) == 0 {
		return Connection{}, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "failed to create connection: no certificate authority is stated")
	}

	if tokens == nil {
		return Connection{}, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "failed to create connection: no token source is stated")
	}

	return Connection{address: address, ca: ca, tokens: tokens}, nil
}

func (c Connection) IsZero() bool {
	return c.address.Host() == ""
}

func (c Connection) Address() domain.Address {
	return c.address
}

func (c Connection) CA() []byte {
	return c.ca
}

func (c Connection) TokenSource() oauth2.TokenSource {
	return c.tokens
}
