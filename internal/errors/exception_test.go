package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A tool's last words survive wrapping: the account carries the outermost
// output in the chain, so a later wrap never drops them.
func TestExceptionOfOutput(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedOutput string
	}{
		{name: "Unset_Empty", err: Newf(TypeInternal, "failed to run terraform apply"), expectedOutput: ""},
		{
			name:           "Attached_Present",
			err:            Wrapf(Newf(TypeInternal, "exit status 1"), TypeInternal, "failed to run terraform apply").WithOutput("Error: no valid credential sources"),
			expectedOutput: "Error: no valid credential sources",
		},
		{
			name:           "OnTheCause_SurvivesTheWrap",
			err:            Wrapf(Wrapf(nil, TypeInternal, "inner").WithOutput("the tool said no"), TypeInternal, "outer"),
			expectedOutput: "the tool said no",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedOutput, ExceptionOf(tt.err).Output)
		})
	}
}

// The envelope is the machine rendering of the same value, so an agent reading
// --format json sees the tool's words where a human sees them streamed.
func TestEnvelopeOfCarriesOutput(t *testing.T) {
	err := Wrapf(Newf(TypeInternal, "exit status 1"), TypeInternal, "failed to run docker compose up").WithOutput("no such image")

	contents, marshalErr := EnvelopeOf(err).MarshalJSON()

	assert.NoError(t, marshalErr)
	assert.Contains(t, string(contents), `"output": "no such image"`)
}
