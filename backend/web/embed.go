package web

import (
	"embed"
	"io/fs"
)

// Files stores the embedded frontend bundle for release builds.
//
//go:embed all:dist
var Files embed.FS

func DistFS() (fs.FS, error) {
	return fs.Sub(Files, "dist")
}
