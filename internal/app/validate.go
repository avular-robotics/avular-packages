package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/core"
	"avular-packages/internal/types"
)

func (s Service) Validate(ctx context.Context, req ValidateRequest) (ValidateResult, error) {
	productPath := strings.TrimSpace(req.ProductPath)
	if productPath == "" {
		productPath = discoverProduct()
	}
	if productPath == "" {
		return ValidateResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("product spec path is required (provide --product or place product.yaml in current directory)")
	}
	product, err := s.SpecLoader.LoadProduct(productPath)
	if err != nil {
		return ValidateResult{}, err
	}

	// Validate inline schema structure before composition
	if product.Schema != nil {
		if err := validateInlineSchema(*product.Schema); err != nil {
			return ValidateResult{}, err
		}
	}

	// Validate inline profile definitions in compose refs
	for _, ref := range product.Compose {
		if ref.Source == "inline" && ref.Profile != nil {
			if err := validateInlineProfile(ref.Name, *ref.Profile); err != nil {
				return ValidateResult{}, err
			}
		}
	}

	profiles, err := s.ProfileSource.LoadProfiles(product, req.Profiles)
	if err != nil {
		return ValidateResult{}, err
	}

	// Validate inline schemas on loaded profile specs (file-based profiles
	// can also carry inline schema definitions that need structural checks).
	for _, profile := range profiles {
		if profile.Schema != nil {
			if err := validateInlineSchema(*profile.Schema); err != nil {
				return ValidateResult{}, errbuilder.New().
					WithCode(errbuilder.CodeInvalidArgument).
					WithMsg(fmt.Sprintf("profile '%s': %s", profile.Metadata.Name, err.Error()))
			}
		}
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

// validateInlineSchema checks structural validity of an inline schema
// definition embedded in a product spec.
func validateInlineSchema(schema types.SchemaFile) error {
	if schema.SchemaVersion == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("inline schema missing schema_version")
	}
	for key, mapping := range schema.Mappings {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		if mapping.Package == "" {
			return errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg(fmt.Sprintf("inline schema key '%s' has empty package", normalizedKey))
		}
		if mapping.Type != types.DependencyTypeApt && mapping.Type != types.DependencyTypePip {
			return errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg(fmt.Sprintf("inline schema key '%s' has invalid type '%s'", normalizedKey, mapping.Type))
		}
	}
	return nil
}

// validateInlineProfile checks that an inline profile definition has
// the minimum required fields to produce a valid composed spec.
func validateInlineProfile(composeName string, profile types.InlineProfile) error {
	if len(profile.Packaging.Groups) == 0 {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("inline profile '%s' must define at least one packaging group", composeName))
	}
	for _, group := range profile.Packaging.Groups {
		if group.Name == "" {
			return errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg(fmt.Sprintf("inline profile '%s' has a packaging group with empty name", composeName))
		}
	}
	return nil
}
