package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrPathOutsideRoot = errors.New("path is outside primary ROM root")
var ErrFileNotFound = errors.New("file not found")
var ErrNotAFile = errors.New("path is not a regular file")
var ErrNoPrimaryRoot = errors.New("PRIMARY_ROM_ROOT is not configured")

type Guard struct {
	root string
}

type ResolvedFile struct {
	RequestedPath string
	ResolvedPath  string
	SizeBytes     int64
	ModTime       int64
}

type ResolvedDirectory struct {
	RequestedPath string
	ResolvedPath  string
}

func NewGuard(root string) *Guard {
	root = strings.TrimSpace(root)
	if root == "" {
		return &Guard{}
	}

	return &Guard{root: filepath.Clean(root)}
}

func (g *Guard) ValidateFile(path string) (*ResolvedFile, error) {
	if g.root == "" {
		return nil, ErrNoPrimaryRoot
	}

	requestedPath := filepath.Clean(strings.TrimSpace(path))
	if requestedPath == "" {
		return nil, ErrFileNotFound
	}

	resolvedPath, err := g.resolveWithinRoot(requestedPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("stat file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, ErrNotAFile
	}

	return &ResolvedFile{
		RequestedPath: requestedPath,
		ResolvedPath:  resolvedPath,
		SizeBytes:     info.Size(),
		ModTime:       info.ModTime().Unix(),
	}, nil
}

func (g *Guard) ValidateDirectory(path string) (*ResolvedDirectory, error) {
	if g.root == "" {
		return nil, ErrNoPrimaryRoot
	}

	requestedPath := filepath.Clean(strings.TrimSpace(path))
	if requestedPath == "" {
		requestedPath = g.root
	}

	resolvedPath, err := g.resolveWithinRoot(requestedPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("stat directory: %w", err)
	}
	if !info.IsDir() {
		return nil, ErrNotAFile
	}

	return &ResolvedDirectory{
		RequestedPath: requestedPath,
		ResolvedPath:  resolvedPath,
	}, nil
}

func (g *Guard) DefaultDirectory() (string, error) {
	if g.root == "" {
		return "", ErrNoPrimaryRoot
	}
	return g.root, nil
}

func (g *Guard) ParentDirectory(path string) *string {
	cleaned := filepath.Clean(path)
	root := filepath.Clean(g.root)
	if root == "." || root == "" {
		return nil
	}
	if cleaned == root {
		return nil
	}
	if isWithinRoot(cleaned, root) {
		parent := filepath.Dir(cleaned)
		if !isWithinRoot(parent, root) {
			return &root
		}
		return &parent
	}
	return nil
}

func (g *Guard) resolveWithinRoot(requestedPath string) (string, error) {
	resolvedPath, err := filepath.EvalSymlinks(requestedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrFileNotFound
		}
		return "", fmt.Errorf("resolve file symlink: %w", err)
	}

	resolvedRoot, err := filepath.EvalSymlinks(g.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNoPrimaryRoot
		}
		return "", fmt.Errorf("resolve root symlink: %w", err)
	}
	if !isWithinRoot(resolvedPath, resolvedRoot) {
		return "", ErrPathOutsideRoot
	}

	return resolvedPath, nil
}

func isWithinRoot(path, root string) bool {
	if path == root {
		return true
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
