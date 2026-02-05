package adapters

import (
	"os"

	"github.com/ZanzyTHEbar/errbuilder-go"
	"gopkg.in/yaml.v3"

	"avular-packages/internal/types"
)

type SpecFileAdapter struct{}

func NewSpecFileAdapter() SpecFileAdapter {
	return SpecFileAdapter{}
}

func (a SpecFileAdapter) LoadProduct(path string) (types.Spec, error) {
	spec, err := a.load(path)
	if err != nil {
		return types.Spec{}, err
	}
	if spec.Kind != types.SpecKindProduct {
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("spec kind is not product")
	}
	return spec, nil
}

func (a SpecFileAdapter) LoadProfile(path string) (types.Spec, error) {
	spec, err := a.load(path)
	if err != nil {
		return types.Spec{}, err
	}
	if spec.Kind != types.SpecKindProfile {
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("spec kind is not profile")
	}
	return spec, nil
}

func (a SpecFileAdapter) load(path string) (types.Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("spec file not found").
			WithCause(err)
	}
	var spec types.Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("failed to parse spec yaml").
			WithCause(err)
	}
	return spec, nil
}
