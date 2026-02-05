package types

type DependencyType string

const (
	DependencyTypeApt DependencyType = "apt"
	DependencyTypePip DependencyType = "pip"
)

type PackagingMode string

const (
	PackagingModeIndividual PackagingMode = "individual"
	PackagingModeMetaBundle PackagingMode = "meta-bundle"
	PackagingModeFatBundle  PackagingMode = "fat-bundle"
)

type SpecKind string

const (
	SpecKindProfile SpecKind = "profile"
	SpecKindProduct SpecKind = "product"
)

type ConstraintOp string

const (
	ConstraintOpNone   ConstraintOp = ""
	ConstraintOpEq     ConstraintOp = "="
	ConstraintOpEq2    ConstraintOp = "=="
	ConstraintOpNe     ConstraintOp = "!="
	ConstraintOpCompat ConstraintOp = "~="
	ConstraintOpGte    ConstraintOp = ">="
	ConstraintOpLte    ConstraintOp = "<="
	ConstraintOpGt     ConstraintOp = ">"
	ConstraintOpLt     ConstraintOp = "<"
)
