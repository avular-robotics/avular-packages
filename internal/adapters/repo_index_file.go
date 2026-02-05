package adapters

import (
	"os"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"
	"gopkg.in/yaml.v3"

	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

type RepoIndexFileAdapter struct {
	Path   string
	cached types.RepoIndexFile
	loaded bool
}

func NewRepoIndexFileAdapter(path string) *RepoIndexFileAdapter {
	return &RepoIndexFileAdapter{Path: path}
}

func (a *RepoIndexFileAdapter) AvailableVersions(depType types.DependencyType, name string) ([]string, error) {
	index, err := a.load()
	if err != nil {
		return nil, err
	}
	switch depType {
	case types.DependencyTypeApt:
		return index.Apt[name], nil
	case types.DependencyTypePip:
		if versions, ok := index.Pip[name]; ok && len(versions) > 0 {
			return versions, nil
		}
		normalized := normalizePipName(name)
		if normalized != name {
			return index.Pip[normalized], nil
		}
		return index.Pip[name], nil
	default:
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("unknown dependency type")
	}
}

func (a *RepoIndexFileAdapter) load() (types.RepoIndexFile, error) {
	if a.loaded {
		return a.cached, nil
	}
	data, err := os.ReadFile(a.Path)
	if err != nil {
		return types.RepoIndexFile{}, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("repo index file not found").
			WithCause(err)
	}
	var idx types.RepoIndexFile
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return types.RepoIndexFile{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("invalid repo index format").
			WithCause(err)
	}
	if idx.Apt == nil {
		idx.Apt = map[string][]string{}
	}
	if idx.Pip == nil {
		idx.Pip = map[string][]string{}
	}
	a.cached = idx
	a.loaded = true
	return idx, nil
}

func normalizePipName(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "-", ".", "-")
	return replacer.Replace(lower)
}

var _ ports.RepoIndexPort = (*RepoIndexFileAdapter)(nil)
