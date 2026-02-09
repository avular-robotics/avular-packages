package types

// SchemaMapping maps an abstract dependency key (as found in package.xml
// standard ROS tags like <exec_depend>, <build_depend>, or <depend>) to a
// concrete, typed, optionally versioned installable package.
//
// Keys are plain names such as "fmt", "rclcpp", or "numpy".  The schema
// resolver looks up each key in the mapping table and emits a typed
// Dependency (apt or pip) with the resolved package name and optional
// version constraint.
type SchemaMapping struct {
	// Type is the target dependency type: "apt" or "pip".
	Type DependencyType `yaml:"type"`

	// Package is the concrete package name in the target ecosystem.
	// For apt: e.g. "libfmt-dev", "ros-humble-rclcpp".
	// For pip: e.g. "numpy", "flask".
	Package string `yaml:"package"`

	// Version is an optional version constraint string.
	// Examples: ">=9.1.0", "==1.26.4", ">=1.0,<2.0".
	// If empty, no version constraint is applied.
	Version string `yaml:"version,omitempty"`
}

// SchemaFile is the top-level structure of a schema.yaml file.
// It provides a deterministic, versioned mapping from abstract dependency
// keys to concrete typed packages.
//
// Multiple schema files can be layered: workspace -> profile -> product.
// Later layers override earlier ones on a per-key basis.
type SchemaFile struct {
	// SchemaVersion identifies the file format version.
	SchemaVersion string `yaml:"schema_version"`

	// Target optionally constrains which OS/platform this schema applies to.
	// Example: "ubuntu-22.04".
	Target string `yaml:"target,omitempty"`

	// Mappings maps abstract keys to concrete typed packages.
	Mappings map[string]SchemaMapping `yaml:"mappings"`
}

// ROSDepScope describes which dependency scope a ROS tag represents.
type ROSDepScope string

const (
	ROSDepScopeExec      ROSDepScope = "exec"
	ROSDepScopeBuild     ROSDepScope = "build"
	ROSDepScopeBuildExec ROSDepScope = "build_exec"
	ROSDepScopeTest      ROSDepScope = "test"
	ROSDepScopeAll       ROSDepScope = "all"
)

// ROSTagDependency is a raw dependency key extracted from a standard ROS
// package.xml tag.  It carries the abstract name plus the scope derived
// from the XML element name.
type ROSTagDependency struct {
	// Key is the abstract dependency name (e.g. "rclcpp", "fmt").
	Key string

	// Scope indicates which lifecycle phase needs this dependency.
	Scope ROSDepScope
}
