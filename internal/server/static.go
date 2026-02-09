package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

// GetStaticHandler returns an http.Handler for the embedded static files.
func GetStaticHandler() http.Handler {
	// Get the "static" subdirectory from the embedded files
	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServerFS(static)
}
