package ports

import "avular-packages/internal/types"

// PackageXMLPort parses package.xml files for dependency information.
type PackageXMLPort interface {
	// ParseDependencies extracts typed dependencies from <export> tags
	// (debian_depend, pip_depend).  Returns (apt deps, pip deps, error).
	ParseDependencies(paths []string, tags []string) ([]string, []string, error)

	// ParseROSTags extracts abstract dependency keys from standard ROS
	// tags: <depend>, <exec_depend>, <build_depend>, <build_export_depend>,
	// <run_depend>, <test_depend>.  These keys are abstract names that
	// must be resolved through a schema mapping before use.
	ParseROSTags(paths []string) ([]types.ROSTagDependency, error)

	// ParsePackageNames returns the <name> element from each package.xml.
	ParsePackageNames(paths []string) ([]string, error)
}

// WorkspacePort discovers package.xml files within workspace roots.
type WorkspacePort interface {
	FindPackageXML(root string) ([]string, error)
}
