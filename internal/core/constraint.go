package core

import (
	"fmt"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/types"
)

var opTokens = []types.ConstraintOp{
	types.ConstraintOpGte,
	types.ConstraintOpLte,
	types.ConstraintOpCompat,
	types.ConstraintOpNe,
	types.ConstraintOpEq2,
	types.ConstraintOpEq,
	types.ConstraintOpGt,
	types.ConstraintOpLt,
}

func ParseConstraint(raw string, source string) (types.Constraint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return types.Constraint{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("empty constraint")
	}
	for _, op := range opTokens {
		if strings.Contains(raw, string(op)) {
			parts := strings.SplitN(raw, string(op), 2)
			name := strings.TrimSpace(parts[0])
			version := strings.TrimSpace(parts[1])
			if name == "" || version == "" {
				return types.Constraint{}, errbuilder.New().
					WithCode(errbuilder.CodeInvalidArgument).
					WithMsg(fmt.Sprintf("invalid constraint: %s", raw))
			}
			return types.Constraint{
				Name:    name,
				Op:      op,
				Version: version,
				Source:  source,
			}, nil
		}
	}
	return types.Constraint{
		Name:    raw,
		Op:      types.ConstraintOpNone,
		Version: "",
		Source:  source,
	}, nil
}
