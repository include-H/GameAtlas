package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrPathOutsideRoots = errors.New("path is outside allowed roots")
var ErrFileNotFound = errors.New("file not found")
var ErrNotAFile = errors.New("path is not a regular file")
var ErrNoAllowedRoots = errors.New("no allowed library roots configured")

type Guard struct {
	roots []string
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

func NewGuard(roots []string) *Guard {
	cleanRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		cleanRoots = append(cleanRoots, filepath.Clean(root))
	}

	return &Guard{roots: cleanRoots}
}

func (g *Guard) ValidateFile(path string) (*ResolvedFile, error) {
	if len(g.roots) == 0 {
		return nil, ErrNoAllowedRoots
	}

	requestedPath := filepath.Clean(strings.TrimSpace(path))
	if requestedPath == "" {
		return nil, ErrFileNotFound
	}

	resolvedPath, err := g.resolveWithinRoots(requestedPath)
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
	if len(g.roots) == 0 {
		return nil, ErrNoAllowedRoots
	}

	requestedPath := filepath.Clean(strings.TrimSpace(path))
	if requestedPath == "" {
		requestedPath = g.roots[0]
	}

	resolvedPath, err := g.resolveWithinRoots(requestedPath)
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
	if len(g.roots) == 0 {
		return "", ErrNoAllowedRoots
	}
	return g.roots[0], nil
}

func (g *Guard) ParentDirectory(path string) *string {
	cleaned := filepath.Clean(path)
	for _, root := range g.roots {
		root = filepath.Clean(root)
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
	}
	return nil
}

func (g *Guard) resolveWithinRoots(requestedPath string) (string, error) {
	resolvedPath, err := filepath.EvalSymlinks(requestedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrFileNotFound
		}
		return "", fmt.Errorf("resolve file symlink: %w", err)
	}

	allowed := false
	for _, root := range g.roots {
		resolvedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("resolve root symlink: %w", err)
		}
		if isWithinRoot(resolvedPath, resolvedRoot) {
			allowed = true
			break
		}
	}

	if !allowed {
		return "", ErrPathOutsideRoots
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
