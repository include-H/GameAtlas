package files

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGuardValidateFileAndDirectory(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "games")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	filePath := filepath.Join(subdir, "demo.iso")
	if err := os.WriteFile(filePath, []byte("demo"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	guard := NewGuard(root)

	file, err := guard.ValidateFile(filePath)
	if err != nil {
		t.Fatalf("ValidateFile returned error: %v", err)
	}
	if file.RequestedPath != filePath {
		t.Fatalf("RequestedPath = %q, want %q", file.RequestedPath, filePath)
	}
	if file.ResolvedPath != filePath {
		t.Fatalf("ResolvedPath = %q, want %q", file.ResolvedPath, filePath)
	}
	if file.SizeBytes != 4 {
		t.Fatalf("SizeBytes = %d, want 4", file.SizeBytes)
	}

	dir, err := guard.ValidateDirectory(subdir)
	if err != nil {
		t.Fatalf("ValidateDirectory returned error: %v", err)
	}
	if dir.ResolvedPath != subdir {
		t.Fatalf("ResolvedPath = %q, want %q", dir.ResolvedPath, subdir)
	}
}

func TestGuardRejectsPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "outside.iso")
	if err := os.WriteFile(outsideFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	guard := NewGuard(root)
	_, err := guard.ValidateFile(outsideFile)
	if !errors.Is(err, ErrPathOutsideRoot) {
		t.Fatalf("ValidateFile() error = %v, want %v", err, ErrPathOutsideRoot)
	}
}

func TestGuardParentDirectory(t *testing.T) {
	root := filepath.Clean("/roms")
	guard := NewGuard(root)

	if got := guard.ParentDirectory(root); got != nil {
		t.Fatalf("ParentDirectory(root) = %v, want nil", *got)
	}

	child := filepath.Join(root, "ps2", "game.iso")
	got := guard.ParentDirectory(child)
	if got == nil || *got != filepath.Join(root, "ps2") {
		t.Fatalf("ParentDirectory(child) = %v, want %q", got, filepath.Join(root, "ps2"))
	}

	outside := guard.ParentDirectory("/tmp/outside.iso")
	if outside != nil {
		t.Fatalf("ParentDirectory(outside) = %v, want nil", *outside)
	}
}

func TestIsWithinRoot(t *testing.T) {
	root := filepath.Clean("/roms")
	if !isWithinRoot(filepath.Join(root, "ps2", "game.iso"), root) {
		t.Fatalf("expected child path to be within root")
	}
	if isWithinRoot("/tmp/game.iso", root) {
		t.Fatalf("expected outside path to be rejected")
	}
}
