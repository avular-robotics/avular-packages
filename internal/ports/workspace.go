package ports

type PackageXMLPort interface {
	ParseDependencies(paths []string, tags []string) ([]string, []string, error)
	ParsePackageNames(paths []string) ([]string, error)
}

type WorkspacePort interface {
	FindPackageXML(root string) ([]string, error)
}
