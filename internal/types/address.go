package types

import "fmt"

func NewAddress(scheme string, host string, port int) string {
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}
