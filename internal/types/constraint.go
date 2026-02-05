package types

type Constraint struct {
	Name    string
	Op      ConstraintOp
	Version string
	Source  string
}

type Dependency struct {
	Name        string
	Type        DependencyType
	Constraints []Constraint
}
