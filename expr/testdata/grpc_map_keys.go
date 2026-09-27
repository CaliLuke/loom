package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// GRPCMapKeys exercises every supported protobuf map key, primitive aliases,
// and recursive maps while retaining Any values.
func GRPCMapKeys() {
	keys := []expr.DataType{Boolean, String, Int, Int32, Int64, UInt, UInt32, UInt64}
	aliases := make([]expr.DataType, len(keys))
	for i, key := range keys {
		aliases[i] = Type("Alias"+key.Name(), Type("Base"+key.Name(), key))
	}
	node := Type("Node", func() {
		Field(1, "children", MapOf(String, "Node"))
		for i, key := range keys {
			Field(i+2, "by_"+key.Name(), MapOf(key, Any))
			Field(i+len(keys)+2, "by_alias_"+key.Name(), MapOf(aliases[i], Any))
		}
	})
	Service("maps", func() {
		Method("echo", func() {
			Payload(node)
			Result(node)
			GRPC(func() {})
		})
	})
}
