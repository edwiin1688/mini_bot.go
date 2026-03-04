package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Sandbox provides path validation to restrict operations inside a specific directory.
type Sandbox struct {
	Workspace string
}

func NewSandbox(workspacePath string) (*Sandbox, error) {
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, err
	}
	absPath = filepath.Clean(absPath)

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	return &Sandbox{
		Workspace: absPath,
	}, nil
}

func (s *Sandbox) isSubPath(path string) bool {
	checkPath := path
	for {
		if checkPath == s.Workspace {
			return true
		}
		if checkPath == filepath.Dir(checkPath) {
			return false
		}
		checkPath = filepath.Dir(checkPath)
	}
}

// CheckPath resolves input path to absolute and ensures it's inside Workspace.
func (s *Sandbox) CheckPath(inputPath string) (string, error) {
	var targetPath string
	if filepath.IsAbs(inputPath) {
		targetPath = inputPath
	} else {
		targetPath = filepath.Join(s.Workspace, inputPath)
	}

	absTargetPath, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		absTargetPath = filepath.Clean(targetPath)
	} else {
		absTargetPath = filepath.Clean(absTargetPath)
	}

	workspace := s.Workspace
	if runtime.GOOS == "windows" {
		workspace = strings.ToLower(workspace)
		absTargetPath = strings.ToLower(absTargetPath)
	}

	if absTargetPath != workspace && !strings.HasPrefix(absTargetPath, workspace+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace bounds: %s", inputPath)
	}

	if !s.isSubPath(absTargetPath) {
		return "", fmt.Errorf("path escapes workspace bounds: %s", inputPath)
	}

	return absTargetPath, nil
}
