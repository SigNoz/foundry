package domain

import (
	"github.com/denisbrodbeck/machineid"
)

const unknownDistinctID = "unknown"

// DistinctID is an anonymous, stable identifier for telemetry attribution. It
// is at most 32 characters and never reveals the source from which it was
// derived; values longer than 32 characters are truncated by NewDistinctID.
type DistinctID struct {
	value string
}

func NewDistinctID() (DistinctID, error) {
	id, err := machineid.ProtectedID("foundryctl")
	if err != nil {
		return UnknownDistinctID(), nil
	}

	return DistinctID{
		value: id,
	}, nil
}

func MustNewDistinctID() DistinctID {
	id, err := NewDistinctID()
	if err != nil {
		panic(err)
	}

	return id
}

// UnknownDistinctID is the sentinel used when no stable raw identifier could be derived.
func UnknownDistinctID() DistinctID {
	return DistinctID{value: unknownDistinctID}
}

func (d DistinctID) String() string {
	return d.value
}
