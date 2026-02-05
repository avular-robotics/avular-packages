package app

import (
	"time"

	"avular-packages/internal/adapters"
	"avular-packages/internal/ports"
)

type Service struct {
	SpecLoader      ports.ProductSpecPort
	ProfileSource   ports.ProfileSourcePort
	Workspace       ports.WorkspacePort
	PackageXML      ports.PackageXMLPort
	OutputReader    ports.OutputReaderPort
	SBOMWriter      ports.SBOMPort
	RepoIndexBuild  ports.RepoIndexBuilderPort
	RepoIndexWriter ports.RepoIndexWriterPort
	InternalDebs    ports.InternalDebsPort
	Clock           func() time.Time
}

func NewService() Service {
	spec := adapters.NewSpecFileAdapter()
	return Service{
		SpecLoader:      spec,
		ProfileSource:   adapters.NewProfileSourceAdapter(spec),
		Workspace:       adapters.NewWorkspaceAdapter(),
		PackageXML:      adapters.NewPackageXMLAdapter(),
		OutputReader:    adapters.NewOutputReaderAdapter(),
		SBOMWriter:      adapters.NewSBOMWriterAdapter(),
		RepoIndexBuild:  adapters.NewRepoIndexBuilderAdapter(),
		RepoIndexWriter: adapters.NewRepoIndexWriterAdapter(),
		InternalDebs:    adapters.NewInternalDebsAdapter(),
		Clock:           time.Now,
	}
}
