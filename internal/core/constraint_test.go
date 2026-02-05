package core

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestParseConstraint(t *testing.T) {
	tests := []struct {
		raw     string
		op      types.ConstraintOp
		name    string
		version string
	}{
		{"libfoo=1.2.3", types.ConstraintOpEq, "libfoo", "1.2.3"},
		{"libfoo==1.2.3", types.ConstraintOpEq2, "libfoo", "1.2.3"},
		{"libfoo>=1.2.3", types.ConstraintOpGte, "libfoo", "1.2.3"},
		{"libfoo<=1.2.3", types.ConstraintOpLte, "libfoo", "1.2.3"},
		{"libfoo>1.2.3", types.ConstraintOpGt, "libfoo", "1.2.3"},
		{"libfoo<1.2.3", types.ConstraintOpLt, "libfoo", "1.2.3"},
		{"libfoo!=1.2.3", types.ConstraintOpNe, "libfoo", "1.2.3"},
		{"libfoo~=1.2.3", types.ConstraintOpCompat, "libfoo", "1.2.3"},
		{"libfoo", types.ConstraintOpNone, "libfoo", ""},
	}

	for _, tt := range tests {
		constraint, err := ParseConstraint(tt.raw, "test")
		require.NoError(t, err)
		if diff := cmp.Diff(tt.op, constraint.Op); diff != "" {
			t.Fatalf("unexpected op (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(tt.name, constraint.Name); diff != "" {
			t.Fatalf("unexpected name (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(tt.version, constraint.Version); diff != "" {
			t.Fatalf("unexpected version (-want +got):\n%s", diff)
		}
	}
}
