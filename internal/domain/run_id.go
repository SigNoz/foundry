package domain

import (
	"github.com/google/uuid"
)

// RunID identifies one foundryctl invocation. A command that carries several
// casting documents reports one event per document, and this is what ties them
// back to the single run that emitted them.
type RunID struct {
	value string
}

func NewRunID() RunID {
	return RunID{value: uuid.NewString()}
}

func (id RunID) String() string {
	return id.value
}
