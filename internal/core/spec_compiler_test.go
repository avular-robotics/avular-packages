package core

import (
	"testing"

	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestSpecCompilerValidateSpecCases(t *testing.T) {
	compiler := NewSpecCompiler()

	tests := []struct {
		name    string
		build   func() types.Spec
		wantErr bool
	}{
		{
			name: "missing packaging group fields",
			build: func() types.Spec {
				spec := baseProfileSpec()
				spec.Packaging.Groups[0].Scope = ""
				spec.Packaging.Groups[0].Matches = nil
				return spec
			},
			wantErr: true,
		},
		{
			name: "invalid resolution directive fields",
			build: func() types.Spec {
				spec := baseProfileSpec()
				spec.Resolutions = []types.ResolutionDirective{
					{
						Dependency: "apt:libfoo",
						Action:     "force",
						Value:      "",
						Reason:     "",
						Owner:      "",
					},
				}
				return spec
			},
			wantErr: true,
		},
		{
			name: "resolution dependency missing type",
			build: func() types.Spec {
				spec := baseProfileSpec()
				spec.Resolutions = []types.ResolutionDirective{
					{
						Dependency: "libfoo",
						Action:     "force",
						Value:      "1.2.0",
						Reason:     "test",
						Owner:      "team",
					},
				}
				return spec
			},
			wantErr: true,
		},
		{
			name: "valid product spec",
			build: func() types.Spec {
				return baseProductSpec()
			},
			wantErr: false,
		},
		{
			name: "profile with compose",
			build: func() types.Spec {
				spec := baseProfileSpec()
				spec.Compose = []types.ComposeRef{
					{Name: "base", Version: "1.0.0", Source: "local", Path: "profile.yaml"},
				}
				return spec
			},
			wantErr: true,
		},
		{
			name: "unsupported target",
			build: func() types.Spec {
				spec := baseProfileSpec()
				spec.Packaging.Groups[0].Targets = []string{"20.04"}
				return spec
			},
			wantErr: true,
		},
		{
			name: "accept ubuntu-prefixed target",
			build: func() types.Spec {
				spec := baseProfileSpec()
				spec.Packaging.Groups[0].Targets = []string{"ubuntu-24.04"}
				return spec
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := compiler.ValidateSpec(t.Context(), tt.build())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func baseProfileSpec() types.Spec {
	return types.Spec{
		APIVersion: "v1",
		Kind:       types.SpecKindProfile,
		Metadata: types.Metadata{
			Name:    "profile",
			Version: "1.0.0",
			Owners:  []string{"team"},
		},
		Packaging: types.Packaging{
			Groups: []types.PackagingGroup{
				{
					Name:    "apt",
					Mode:    types.PackagingModeIndividual,
					Scope:   "runtime",
					Matches: []string{"apt:*"},
					Targets: []string{"24.04"},
				},
			},
		},
	}
}

func baseProductSpec() types.Spec {
	return types.Spec{
		APIVersion: "v1",
		Kind:       types.SpecKindProduct,
		Metadata: types.Metadata{
			Name:    "product",
			Version: "1.0.0",
			Owners:  []string{"team"},
		},
		Compose: []types.ComposeRef{
			{Name: "base", Version: "1.0.0", Source: "local", Path: "fixtures/profile-base.yaml"},
		},
		Inputs: types.Inputs{
			PackageXML: types.PackageXMLInput{Enabled: true, Tags: []string{"debian_depend"}},
		},
		Packaging: types.Packaging{
			Groups: []types.PackagingGroup{
				{
					Name:    "apt",
					Mode:    types.PackagingModeIndividual,
					Scope:   "runtime",
					Matches: []string{"apt:*"},
					Targets: []string{"24.04"},
				},
			},
		},
		Publish: types.Publish{
			Repository: types.PublishRepository{
				Name:           "avular",
				Channel:        "dev",
				SnapshotPrefix: "sample",
				SigningKey:     "key",
			},
		},
	}
}
