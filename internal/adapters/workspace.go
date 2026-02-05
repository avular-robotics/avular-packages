package adapters

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
)

type WorkspaceAdapter struct{}

func NewWorkspaceAdapter() WorkspaceAdapter {
	return WorkspaceAdapter{}
}

func (a WorkspaceAdapter) FindPackageXML(root string) ([]string, error) {
	var paths []string
	if root == "" {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("workspace root is empty")
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipWorkspaceDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipWorkspacePath(path) {
			return nil
		}
		if filepath.Base(path) == "package.xml" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to scan workspace").
			WithCause(err)
	}
	return paths, nil
}

func shouldSkipWorkspaceDir(name string) bool {
	switch name {
	case "install", "build", "log", ".git", ".colcon", ".ros", "devel":
		return true
	default:
		return false
	}
}

func shouldSkipWorkspacePath(path string) bool {
	ignored := []string{
		string(filepath.Separator) + "install" + string(filepath.Separator),
		string(filepath.Separator) + "build" + string(filepath.Separator),
		string(filepath.Separator) + "log" + string(filepath.Separator),
		string(filepath.Separator) + "devel" + string(filepath.Separator),
		string(filepath.Separator) + ".git" + string(filepath.Separator),
		string(filepath.Separator) + ".colcon" + string(filepath.Separator),
		string(filepath.Separator) + ".ros" + string(filepath.Separator),
	}
	for _, marker := range ignored {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
}

var _ ports.WorkspacePort = WorkspaceAdapter{}
