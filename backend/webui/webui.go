// Package webui embeds the built React frontend so the kibo binary is
// fully self-contained — no Node, no separate file tree to deploy.
//
// Build the frontend first (npm run build in frontend/, which outputs
// to webui/dist), then go build.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var dist embed.FS

// FS returns the built frontend as an http-servable filesystem.
func FS() (http.FileSystem, error) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}
