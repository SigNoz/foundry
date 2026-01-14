package types

import (
	"fmt"
	"net/url"
	"strconv"
)

type Address struct {
	host string
	port int
}

func (address *Address) Host() string {
	return address.host
}

func (address *Address) Port() int {
	return address.port
}

// FormatAddress creates a formatted address string from scheme, host, and port.
func FormatAddress(scheme, host string, port int) string {
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

// NewAddress parses a URL-formatted address string into host and port components.
func NewAddress(address string) (Address, error) {
	u, err := url.Parse(address)
	if err != nil {
		return Address{}, fmt.Errorf("invalid address %q: %w", address, err)
	}

	host := u.Hostname()
	portStr := u.Port()

	if host == "" {
		return Address{}, fmt.Errorf("address %q has no host", address)
	}

	var port int
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return Address{}, fmt.Errorf("address %q has invalid port: %w", address, err)
		}
	}

	return Address{host: host, port: port}, nil
}

// NewAddresses parses multiple address strings and returns the parsed results.
func NewAddresses(addresses []string) ([]Address, error) {
	result := make([]Address, 0, len(addresses))
	for i, addr := range addresses {
		parsed, err := NewAddress(addr)
		if err != nil {
			return nil, fmt.Errorf("address[%d]: %w", i, err)
		}
		result = append(result, parsed)
	}
	return result, nil
}
