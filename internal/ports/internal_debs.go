package ports

type InternalDebsPort interface {
	CopyDebs(srcDir string, destDir string) error
	BuildInternalDebs(srcDirs []string, destDir string) error
}
