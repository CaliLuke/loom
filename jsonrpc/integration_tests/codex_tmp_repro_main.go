package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CaliLuke/loom/jsonrpc/integration_tests/framework"
)

func main() {
	methods := map[string]framework.MethodInfo{}
	for _, n := range []string{"stream_object_sse"} {
		mi, err := framework.ParseMethod(n)
		if err != nil {
			panic(err)
		}
		methods[mi.Name()] = mi
	}
	work := filepath.Join(os.TempDir(), "loom-jsonrpc-repro")
	_ = os.RemoveAll(work)
	if err := os.MkdirAll(work, 0o755); err != nil {
		panic(err)
	}
	g := framework.NewGenerator(work, methods)
	if err := g.Generate(); err != nil {
		panic(err)
	}
	data, err := os.ReadFile(filepath.Join(work, "testsse.go"))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
