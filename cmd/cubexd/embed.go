package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var embeddedWeb embed.FS

func getFileSystem() http.FileSystem {
	sub, err := fs.Sub(embeddedWeb, "dist")
	if err != nil {
		return http.FS(embeddedWeb)
	}
	return http.FS(sub)
}
