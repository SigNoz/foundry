package v1alpha1

import "github.com/signoz/foundry/internal/domain"

// Machinery is the marker every per-Kind casting type satisfies.
type Machinery interface {
	Kind() Kind
	TrackableProperties() domain.Properties
}
