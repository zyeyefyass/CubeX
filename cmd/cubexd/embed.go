package main

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embeddedWeb embed.FS

func getEmbeddedFS() fs.FS {
	sub, err := fs.Sub(embeddedWeb, "dist")
	if err != nil {
		return embeddedWeb
	}
	return sub
}
