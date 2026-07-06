package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func updateHeader(f *codegen.File) {
	header := f.HeaderSection()
	if header == nil {
		return
	}
	data := codegen.HeaderDataForSection(header)
	if data == nil {
		return
	}
	data.Title = strings.Replace(data.Title, "HTTP", "JSON-RPC", 1)
	for _, i := range data.Imports {
		i.Path = strings.Replace(i.Path, "gen/http", "gen/jsonrpc", 1)
	}
}
