package ports

type PackageBuildPort interface {
	BuildDebs(inputDir string, outputDir string) error
}
