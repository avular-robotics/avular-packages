package core

import (
	"context"
	"fmt"

	"github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/rs/zerolog/log"

	"avular-packages/internal/types"
)

type ProductComposer struct{}

func NewProductComposer() ProductComposer {
	return ProductComposer{}
}

func (c ProductComposer) Compose(ctx context.Context, product types.Spec, profiles []types.Spec) (types.Spec, error) {
	if product.Kind != types.SpecKindProduct {
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("compose requires product spec")
	}
	if err := validateComposeOrder(product.Compose); err != nil {
		return types.Spec{}, err
	}

	composed := types.Spec{
		APIVersion:  product.APIVersion,
		Kind:        types.SpecKindProduct,
		Metadata:    product.Metadata,
		Compose:     product.Compose,
		Inputs:      types.Inputs{},
		Packaging:   types.Packaging{},
		Resolutions: []types.ResolutionDirective{},
		Publish:     types.Publish{},
	}

	for _, profile := range profiles {
		if profile.Kind != types.SpecKindProfile {
			return types.Spec{}, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg(fmt.Sprintf("invalid profile spec kind: %s", profile.Metadata.Name))
		}
		if err := mergeSpec(&composed, profile); err != nil {
			return types.Spec{}, err
		}
	}

	if err := mergeSpec(&composed, product); err != nil {
		return types.Spec{}, err
	}

	log.Ctx(ctx).Debug().Str("product", product.Metadata.Name).Msg("product composed")
	return composed, nil
}

func mergeSpec(target *types.Spec, incoming types.Spec) error {
	mergeInputs(&target.Inputs, incoming.Inputs)
	if err := mergePackagingGroups(&target.Packaging, incoming.Packaging); err != nil {
		return err
	}
	target.Resolutions = append(target.Resolutions, incoming.Resolutions...)
	if incoming.Publish.Repository.Name != "" {
		target.Publish = incoming.Publish
	}
	return nil
}

func mergeInputs(target *types.Inputs, incoming types.Inputs) {
	if incoming.PackageXML.Enabled {
		target.PackageXML.Enabled = true
	}
	if len(incoming.PackageXML.Tags) > 0 {
		target.PackageXML.Tags = append(target.PackageXML.Tags, incoming.PackageXML.Tags...)
	}
	if incoming.PackageXML.IncludeSrc {
		target.PackageXML.IncludeSrc = true
	}
	target.Manual.Apt = append(target.Manual.Apt, incoming.Manual.Apt...)
	target.Manual.Python = append(target.Manual.Python, incoming.Manual.Python...)
}

func mergePackagingGroups(target *types.Packaging, incoming types.Packaging) error {
	existing := map[string]struct{}{}
	for _, group := range target.Groups {
		existing[group.Name] = struct{}{}
	}
	for _, group := range incoming.Groups {
		if _, found := existing[group.Name]; found {
			return errbuilder.New().
				WithCode(errbuilder.CodeAlreadyExists).
				WithMsg(fmt.Sprintf("duplicate packaging group: %s", group.Name))
		}
		target.Groups = append(target.Groups, group)
	}
	return nil
}

func validateComposeOrder(compose []types.ComposeRef) error {
	seen := map[string]struct{}{}
	for _, ref := range compose {
		key := fmt.Sprintf("%s@%s", ref.Name, ref.Version)
		if _, ok := seen[key]; ok {
			return errbuilder.New().
				WithCode(errbuilder.CodeAlreadyExists).
				WithMsg(fmt.Sprintf("duplicate compose entry: %s", key))
		}
		seen[key] = struct{}{}
	}
	return nil
}
