package app

import (
	"context"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/core"
)

func (s Service) Validate(ctx context.Context, req ValidateRequest) (ValidateResult, error) {
	productPath := strings.TrimSpace(req.ProductPath)
	if productPath == "" {
		return ValidateResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("product spec path is required")
	}
	product, err := s.SpecLoader.LoadProduct(productPath)
	if err != nil {
		return ValidateResult{}, err
	}
	profiles, err := s.ProfileSource.LoadProfiles(product, req.Profiles)
	if err != nil {
		return ValidateResult{}, err
	}
	composer := core.NewProductComposer()
	compiler := core.NewSpecCompiler()
	composed, err := composer.Compose(ctx, product, profiles)
	if err != nil {
		return ValidateResult{}, err
	}
	if err := compiler.ValidateSpec(ctx, composed); err != nil {
		return ValidateResult{}, err
	}
	return ValidateResult{ProductName: composed.Metadata.Name}, nil
}
