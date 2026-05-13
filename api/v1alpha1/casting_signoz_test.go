package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSigNozCastingShape(t *testing.T) {
	t.Parallel()

	c := DefaultSigNozCasting()
	assert.Equal(t, KindSigNoz, c.Kind)
	assert.Equal(t, "v1alpha1", c.APIVersion)
	assert.NotNil(t, c.SigNozSpec())
	assert.NotNil(t, c.SigNozStatus())
}

func TestExampleSigNozCastingShape(t *testing.T) {
	t.Parallel()

	c := ExampleSigNozCasting()
	assert.Equal(t, KindSigNoz, c.Kind)
	assert.NotNil(t, c.SigNozSpec())
	assert.Nil(t, c.Status, "ExampleSigNozCasting omits Status so YAML serialization stays minimal")
}

func TestSigNozSpecPanicsOnWrongKind(t *testing.T) {
	t.Parallel()

	c := Casting{Kind: KindSigNoz, Spec: "not a SigNozCastingSpec"}
	assert.Panics(t, func() { c.SigNozSpec() }, "accessor must panic when Spec isn't *SigNozCastingSpec")
}

func TestSigNozStatusPanicsOnWrongKind(t *testing.T) {
	t.Parallel()

	c := Casting{Kind: KindSigNoz, Status: "not a SigNozCastingStatus"}
	assert.Panics(t, func() { c.SigNozStatus() }, "accessor must panic when Status isn't *SigNozCastingStatus")
}
