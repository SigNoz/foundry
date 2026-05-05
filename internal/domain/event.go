package domain

import "github.com/signoz/foundry/internal/errors"

const (
	eventOutcomeSucceeded = "succeeded"
	eventOutcomeFailed    = "failed"
)

// Event names a foundryctl pipeline operation tracked by the ledger. An Event
// carries an optional past-tense outcome (succeeded or failed) so that ledger
// adapters see a finite, stable vocabulary like "forge succeeded" or
// "forge failed" rather than a single name disambiguated by a property.
type Event struct {
	name    string
	outcome string
}

var (
	EventGauge   = Event{name: "gauge"}
	EventForge   = Event{name: "forge"}
	EventCast    = Event{name: "cast"}
	EventCatalog = Event{name: "catalog"}
)

var allEvents = []Event{EventGauge, EventForge, EventCast, EventCatalog}

// NewEvent accepts only the names of declared base Event values. The returned
// Event has no outcome; use Succeeded or Failed to attach one.
func NewEvent(s string) (Event, error) {
	for _, e := range allEvents {
		if e.name == s {
			return e, nil
		}
	}

	return Event{}, errors.Newf(errors.TypeInvalidInput, "failed to create event from %q: name is not a known event", s)
}

func MustNewEvent(s string) Event {
	e, err := NewEvent(s)
	if err != nil {
		panic(err)
	}

	return e
}

// Succeeded returns a copy of e with the success outcome attached.
func (e Event) Succeeded() Event {
	return Event{name: e.name, outcome: eventOutcomeSucceeded}
}

// Failed returns a copy of e with the failure outcome attached.
func (e Event) Failed() Event {
	return Event{name: e.name, outcome: eventOutcomeFailed}
}

// String renders the event as "<name>" or "<name> <outcome>".
func (e Event) String() string {
	if e.outcome == "" {
		return e.name
	}

	return e.name + " " + e.outcome
}
