package policies

import (
	"fmt"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/types"
)

const (
	ActionForce   = "force"
	ActionRelax   = "relax"
	ActionReplace = "replace"
	ActionBlock   = "block"
)

func ApplyResolution(dep types.Dependency, directive types.ResolutionDirective) (types.Dependency, types.ResolutionRecord, error) {
	record := types.ResolutionRecord(directive)

	switch strings.ToLower(directive.Action) {
	case ActionForce:
		if directive.Value == "" {
			return types.Dependency{}, record, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("force directive requires value")
		}
		dep.Constraints = []types.Constraint{{
			Name:    dep.Name,
			Op:      types.ConstraintOpEq,
			Version: directive.Value,
			Source:  "resolution:force",
		}}
		return dep, record, nil
	case ActionRelax:
		dep.Constraints = []types.Constraint{}
		return dep, record, nil
	case ActionReplace:
		if directive.Value == "" {
			return types.Dependency{}, record, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("replace directive requires value")
		}
		dep.Name = directive.Value
		dep.Constraints = []types.Constraint{}
		return dep, record, nil
	case ActionBlock:
		return types.Dependency{}, record, errbuilder.New().
			WithCode(errbuilder.CodePermissionDenied).
			WithMsg(fmt.Sprintf("dependency blocked by directive: %s", dep.Name))
	default:
		return types.Dependency{}, record, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("unknown resolution action: %s", directive.Action))
	}
}
