package app

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestValidateApp(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	productPath := filepath.Join(root, "fixtures", "product-sample.yaml")
	profilePath := filepath.Join(root, "fixtures", "profile-base.yaml")

	service := NewService()
	result, err := service.Validate(t.Context(), ValidateRequest{
		ProductPath: productPath,
		Profiles:    []string{profilePath},
	})
	require.NoError(t, err)
	if diff := cmp.Diff("sample-product", result.ProductName); diff != "" {
		t.Fatalf("unexpected product name (-want +got):\n%s", diff)
	}
}
