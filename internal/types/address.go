package types

import (
	"fmt"
	"net/url"
)

// NewAddress creates a formatted address string from scheme, host, and port.
func NewAddress(scheme, host string, port int) string {
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

// ParsedAddress represents a parsed network address with host and port.
type ParsedAddress struct {
	Host string
	Port string
}

// ParseAddress parses a URL-formatted address string into host and port components.
func ParseAddress(address string) (ParsedAddress, error) {
	u, err := url.Parse(address)
	if err != nil {
		return ParsedAddress{}, fmt.Errorf("invalid address %q: %w", address, err)
	}

	host := u.Hostname()
	port := u.Port()

	if host == "" {
		return ParsedAddress{}, fmt.Errorf("address %q has no host", address)
	}

	return ParsedAddress{Host: host, Port: port}, nil
}

// ParseAddresses parses multiple address strings and returns the parsed results.
func ParseAddresses(addresses []string) ([]ParsedAddress, error) {
	result := make([]ParsedAddress, 0, len(addresses))
	for i, addr := range addresses {
		parsed, err := ParseAddress(addr)
		if err != nil {
			return nil, fmt.Errorf("address[%d]: %w", i, err)
		}
		result = append(result, parsed)
	}
	return result, nil
}
