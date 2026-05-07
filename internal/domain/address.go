package domain

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/signoz/foundry/internal/errors"
)

type Address struct {
	scheme string
	host   string
	port   int
}

// NewAddress requires a non-empty scheme and host, and a port in [0, 65535] where 0 means no port.
func NewAddress(scheme, host string, port int) (Address, error) {
	if scheme == "" {
		return Address{}, errors.Newf(errors.TypeInvalidInput, "failed to create address: scheme is empty")
	}

	if host == "" {
		return Address{}, errors.Newf(errors.TypeInvalidInput, "failed to create address: host is empty")
	}

	if port < 0 || port > 65535 {
		return Address{}, errors.Newf(errors.TypeInvalidInput, "failed to create address: port %d is out of range", port)
	}

	return Address{scheme: scheme, host: host, port: port}, nil
}

func MustNewAddress(scheme, host string, port int) Address {
	address, err := NewAddress(scheme, host, port)
	if err != nil {
		panic(err)
	}

	return address
}

// ParseAddress accepts "scheme://host[:port]"; IPv6 literals must be bracketed
// (e.g. "tcp://[::1]:9000" or "tcp://[::1]").
func ParseAddress(raw string) (Address, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Address{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create address from %q: contents are not a valid URL", raw)
	}

	var port int
	if portStr := u.Port(); portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return Address{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create address from %q: port is not a valid integer", raw)
		}
	}

	return NewAddress(u.Scheme, u.Hostname(), port)
}

func ParseAddresses(raws []string) ([]Address, error) {
	result := make([]Address, 0, len(raws))

	for i, raw := range raws {
		address, err := ParseAddress(raw)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "addresses[%d]", i)
		}

		result = append(result, address)
	}

	return result, nil
}

func (a Address) Scheme() string {
	return a.scheme
}

func (a Address) Host() string {
	return a.host
}

func (a Address) Port() int {
	return a.port
}

// String renders the address as "scheme://host[:port]", bracketing IPv6 hosts and omitting the port suffix when port is zero.
func (a Address) String() string {
	host := a.host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}

	if a.port == 0 {
		return a.scheme + "://" + host
	}

	return a.scheme + "://" + host + ":" + strconv.Itoa(a.port)
}
