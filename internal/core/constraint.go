package core

import (
	"fmt"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/types"
)

// opTokens is the ordered list of constraint operators tried during
// parsing. Longer tokens must precede shorter ones to avoid false matches
// (e.g. ">=" before ">").
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

// ParseConstraint splits a raw "name>=version" string into a Constraint.
// When no operator is found the constraint is treated as a bare name
// reference with ConstraintOpNone.
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
