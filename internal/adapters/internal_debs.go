package adapters

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
	"avular-packages/internal/shared"
)

type InternalDebsAdapter struct{}

func NewInternalDebsAdapter() InternalDebsAdapter {
	return InternalDebsAdapter{}
}

func (a InternalDebsAdapter) CopyDebs(srcDir string, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("internal deb dir not found").
			WithCause(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".deb") {
			continue
		}
		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())
		if err := copyDebFile(srcPath, destPath); err != nil {
			return err
		}
	}
	return nil
}

func (a InternalDebsAdapter) BuildInternalDebs(srcDirs []string, destDir string) error {
	for _, dir := range srcDirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		cmd := exec.Command("dpkg-buildpackage", "-b", "-us", "-uc")
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("dpkg-buildpackage failed").
				WithCause(shared.CommandError(output, err))
		}
		parent := filepath.Dir(dir)
		if err := a.CopyDebs(parent, destDir); err != nil {
			return err
		}
	}
	return nil
}

func copyDebFile(srcPath string, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("failed to open source deb").
			WithCause(err)
	}
	defer srcFile.Close()
	destFile, err := os.Create(destPath)
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create destination deb").
			WithCause(err)
	}
	defer destFile.Close()
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to copy deb").
			WithCause(err)
	}
	return nil
}

var _ ports.InternalDebsPort = InternalDebsAdapter{}
