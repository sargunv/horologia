//go:build embedweb

package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distEmbed embed.FS

func init() {
	sub, err := fs.Sub(distEmbed, "dist")
	if err != nil {
		panic("webui: " + err.Error())
	}

	distFS = sub

	info, err := fs.Stat(distFS, "index.html")
	hasEmbeddedIndex = err == nil && !info.IsDir()
}
