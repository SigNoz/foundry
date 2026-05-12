package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		base     TypeMetadata
		override TypeMetadata
		want     TypeMetadata
	}{
		{
			name:     "override Name wins over base",
			base:     TypeMetadata{Name: "base"},
			override: TypeMetadata{Name: "override"},
			want:     TypeMetadata{Name: "override"},
		},
		{
			name:     "override fills in unset base fields",
			base:     TypeMetadata{},
			override: TypeMetadata{Name: "fresh"},
			want:     TypeMetadata{Name: "fresh"},
		},
		{
			name:     "override Annotations (map with omitempty) does not clobber base when unset",
			base:     TypeMetadata{Name: "base", Annotations: map[string]string{"a": "1"}},
			override: TypeMetadata{Name: "base"},
			want:     TypeMetadata{Name: "base", Annotations: map[string]string{"a": "1"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base := tc.base
			override := tc.override
			require.NoError(t, Merge(&base, &override))
			assert.Equal(t, tc.want, base)
		})
	}
}

func TestMerge_RejectsAnyTypedFields(t *testing.T) {
	t.Parallel()

	base := DefaultCasting()
	override := Casting{
		Spec: &SigNozCastingSpec{
			Deployment: TypeDeployment{Mode: ModeDocker, Flavor: FlavorCompose},
		},
	}

	err := Merge(&base, &override)
	require.Error(t, err, "Merge should refuse Casting because Spec/Status are any-typed; use MergeCasting")
	assert.Contains(t, err.Error(), "interface")
}

func TestMergeCasting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		base     Casting
		override Casting
		assert   func(t *testing.T, c Casting)
	}{
		{
			name: "override Spec field wins over base",
			base: Casting{
				Kind: KindSigNoz,
				Spec: &SigNozCastingSpec{
					Deployment: TypeDeployment{Mode: ModeDocker, Flavor: FlavorCompose},
				},
			},
			override: Casting{
				Spec: &SigNozCastingSpec{
					Deployment: TypeDeployment{Mode: ModeDocker, Flavor: FlavorSwarm},
				},
			},
			assert: func(t *testing.T, c Casting) {
				spec := c.SigNozSpec()
				assert.Equal(t, ModeDocker, spec.Deployment.Mode)
				assert.Equal(t, FlavorSwarm, spec.Deployment.Flavor)
			},
		},
		{
			name: "empty override Spec preserves base Spec fields",
			base: Casting{
				Kind: KindSigNoz,
				Spec: &SigNozCastingSpec{
					Deployment: TypeDeployment{Mode: ModeDocker, Flavor: FlavorCompose},
				},
			},
			override: Casting{Spec: &SigNozCastingSpec{}},
			assert: func(t *testing.T, c Casting) {
				spec := c.SigNozSpec()
				assert.Equal(t, ModeDocker, spec.Deployment.Mode)
				assert.Equal(t, FlavorCompose, spec.Deployment.Flavor)
			},
		},
		{
			name: "override APIVersion wins over base",
			base: Casting{
				TypeVersion: TypeVersion{APIVersion: "v1alpha1"},
				Kind:        KindSigNoz,
				Spec:        &SigNozCastingSpec{},
			},
			override: Casting{
				TypeVersion: TypeVersion{APIVersion: "v1alpha2"},
				Spec:        &SigNozCastingSpec{},
			},
			assert: func(t *testing.T, c Casting) {
				assert.Equal(t, "v1alpha2", c.APIVersion)
			},
		},
		{
			name: "empty override APIVersion preserves base",
			base: Casting{
				TypeVersion: TypeVersion{APIVersion: "v1alpha1"},
				Kind:        KindSigNoz,
				Spec:        &SigNozCastingSpec{},
			},
			override: Casting{Spec: &SigNozCastingSpec{}},
			assert: func(t *testing.T, c Casting) {
				assert.Equal(t, "v1alpha1", c.APIVersion)
			},
		},
		{
			name: "override Metadata Name wins via strategic merge",
			base: Casting{
				Kind:     KindSigNoz,
				Metadata: TypeMetadata{Name: "base"},
				Spec:     &SigNozCastingSpec{},
			},
			override: Casting{
				Metadata: TypeMetadata{Name: "override"},
				Spec:     &SigNozCastingSpec{},
			},
			assert: func(t *testing.T, c Casting) {
				assert.Equal(t, "override", c.Metadata.Name)
			},
		},
		{
			name: "nil override Spec preserves base Spec untouched",
			base: Casting{
				Kind: KindSigNoz,
				Spec: &SigNozCastingSpec{
					Deployment: TypeDeployment{Mode: ModeDocker},
				},
			},
			override: Casting{Kind: KindSigNoz},
			assert: func(t *testing.T, c Casting) {
				spec := c.SigNozSpec()
				assert.Equal(t, ModeDocker, spec.Deployment.Mode)
			},
		},
		{
			name: "override Spec mutations propagate through pointer",
			base: Casting{
				Kind: KindSigNoz,
				Spec: &SigNozCastingSpec{},
			},
			override: Casting{
				Spec: &SigNozCastingSpec{
					Deployment: TypeDeployment{Mode: ModeDocker},
				},
			},
			assert: func(t *testing.T, c Casting) {
				assert.Equal(t, ModeDocker, c.SigNozSpec().Deployment.Mode,
					"override field reachable via accessor after merge")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base := tc.base
			override := tc.override
			require.NoError(t, MergeCasting(&base, &override))
			tc.assert(t, base)
		})
	}
}

func TestMergeCastingSpecAndStatus(t *testing.T) {
	t.Parallel()

	t.Run("SigNoz folds Status.Env into Spec.Env with user keys winning", func(t *testing.T) {
		t.Parallel()

		c := Casting{
			Kind: KindSigNoz,
			Spec: &SigNozCastingSpec{
				Ingester: Ingester{
					Spec: MoldingSpec{Env: map[string]string{"FOO": "user"}},
					Status: IngesterStatus{
						MoldingStatus: MoldingStatus{
							Env: map[string]string{"FOO": "enricher", "BAR": "enricher"},
						},
					},
				},
			},
		}

		require.NoError(t, MergeCastingSpecAndStatus(&c))

		env := c.SigNozSpec().Ingester.Spec.Env
		assert.Equal(t, "user", env["FOO"], "user-set keys win over enricher values")
		assert.Equal(t, "enricher", env["BAR"], "non-overlapping enricher keys fill in")
	})

	t.Run("nil Spec is a no-op", func(t *testing.T) {
		t.Parallel()

		c := Casting{Kind: KindSigNoz}
		require.NoError(t, MergeCastingSpecAndStatus(&c))
	})
}
