package core

import (
	"context"
	"fmt"
	"strings"

	assert "github.com/ZanzyTHEbar/assert-lib"
	"github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/rs/zerolog/log"

	"avular-packages/internal/policies"
	"avular-packages/internal/types"
)

type SpecCompiler struct{}

var ubuntuLTS = map[string]struct{}{
	"22.04": {},
	"24.04": {},
}

var validPackagingModes = map[types.PackagingMode]struct{}{
	types.PackagingModeIndividual: {},
	types.PackagingModeMetaBundle: {},
	types.PackagingModeFatBundle:  {},
}

var validPackagingScopes = map[string]struct{}{
	"runtime": {},
	"dev":     {},
	"test":    {},
	"doc":     {},
}

func NewSpecCompiler() SpecCompiler {
	return SpecCompiler{}
}

func (c SpecCompiler) ValidateSpec(ctx context.Context, spec types.Spec) error {
	assert.NotEmpty(ctx, spec.APIVersion, "api_version must be set")
	assert.NotEmpty(ctx, string(spec.Kind), "kind must be set")
	assert.NotEmpty(ctx, spec.Metadata.Name, "metadata.name must be set")
	assert.NotEmpty(ctx, spec.Metadata.Version, "metadata.version must be set")
	if len(spec.Metadata.Owners) == 0 {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("metadata.owners must not be empty")
	}
	if spec.Kind != types.SpecKindProduct && spec.Kind != types.SpecKindProfile {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("spec kind must be product or profile")
	}
	if spec.Kind == types.SpecKindProduct && len(spec.Compose) == 0 {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("product spec must include compose list")
	}
	if spec.Kind == types.SpecKindProfile && len(spec.Compose) > 0 {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("profile spec must not include compose")
	}
	if len(spec.Packaging.Groups) == 0 {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("packaging.groups must not be empty")
	}
	for _, group := range spec.Packaging.Groups {
		if err := validatePackagingGroup(group); err != nil {
			return err
		}
		if err := validateTargets(group.Targets); err != nil {
			return err
		}
	}
	if err := validateResolutions(spec.Resolutions); err != nil {
		return err
	}
	if spec.Inputs.PackageXML.Enabled && len(spec.Inputs.PackageXML.Tags) == 0 {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("package_xml enabled but tags are empty")
	}
	if spec.Kind == types.SpecKindProduct {
		if err := validatePublish(spec.Publish.Repository); err != nil {
			return err
		}
	}
	log.Ctx(ctx).Debug().Str("spec", spec.Metadata.Name).Msg("spec validated")
	return nil
}

func validatePackagingGroup(group types.PackagingGroup) error {
	if group.Name == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("packaging.groups.name must not be empty")
	}
	if group.Mode == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("packaging group %s missing mode", group.Name))
	}
	if _, ok := validPackagingModes[group.Mode]; !ok {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("packaging group %s has invalid mode %s", group.Name, group.Mode))
	}
	if strings.TrimSpace(group.Scope) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("packaging group %s missing scope", group.Name))
	}
	if _, ok := validPackagingScopes[group.Scope]; !ok {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("packaging group %s has invalid scope %s", group.Name, group.Scope))
	}
	if len(group.Matches) == 0 {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("packaging group %s missing matches", group.Name))
	}
	if len(group.Targets) == 0 {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("packaging group %s missing targets", group.Name))
	}
	return nil
}

func validateTargets(targets []string) error {
	for _, target := range targets {
		normalized := normalizeUbuntuTarget(target)
		if _, ok := ubuntuLTS[normalized]; !ok {
			return errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg(fmt.Sprintf("unsupported Ubuntu target: %s", target))
		}
	}
	return nil
}

func validatePublish(repo types.PublishRepository) error {
	if repo.Name == "" || repo.Channel == "" || repo.SnapshotPrefix == "" || repo.SigningKey == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("publish.repository fields are required for product specs")
	}
	return nil
}

func validateResolutions(resolutions []types.ResolutionDirective) error {
	for _, directive := range resolutions {
		if err := validateResolutionDirective(directive); err != nil {
			return err
		}
	}
	return nil
}

func validateResolutionDirective(directive types.ResolutionDirective) error {
	if strings.TrimSpace(directive.Dependency) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("resolution directive dependency must not be empty")
	}
	if !isTypedDependency(directive.Dependency) {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("resolution directive dependency must be typed (apt:name or pip:name): %s", directive.Dependency))
	}
	action := strings.ToLower(strings.TrimSpace(directive.Action))
	if action == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("resolution directive action must not be empty")
	}
	switch action {
	case policies.ActionForce, policies.ActionRelax, policies.ActionReplace, policies.ActionBlock:
	default:
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("resolution directive has invalid action: %s", directive.Action))
	}
	if strings.TrimSpace(directive.Reason) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("resolution directive reason must not be empty")
	}
	if strings.TrimSpace(directive.Owner) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("resolution directive owner must not be empty")
	}
	if (action == policies.ActionForce || action == policies.ActionReplace) && strings.TrimSpace(directive.Value) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("resolution directive value must not be empty for force/replace actions")
	}
	return nil
}

func isTypedDependency(value string) bool {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return false
	}
	switch strings.ToLower(parts[0]) {
	case "apt", "pip":
		return strings.TrimSpace(parts[1]) != ""
	default:
		return false
	}
}

func normalizeUbuntuTarget(value string) string {
	normalized := strings.TrimSpace(value)
	lower := strings.ToLower(normalized)
	if strings.HasPrefix(lower, "ubuntu-") {
		return strings.TrimSpace(normalized[len("ubuntu-"):])
	}
	return normalized
}
