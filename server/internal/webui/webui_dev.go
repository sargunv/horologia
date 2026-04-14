//go:build !embedweb

package webui

import "io/fs"

type emptyFS struct{}

func (emptyFS) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func init() {
	distFS = emptyFS{}
	hasEmbeddedIndex = false
}
