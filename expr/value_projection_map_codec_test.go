package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestValueProjectionMapMatchesNativeCodec(t *testing.T) {
	kinds := []struct {
		typ Primitive
		key any
	}{
		{Boolean, false}, {String, ""}, {Any, ""}, {Int, int(0)}, {Int32, int32(0)}, {Int64, int64(0)},
		{UInt, uint(0)}, {UInt32, uint32(0)}, {UInt64, uint64(0)}, {Float32, float32(0)}, {Float64, float64(0)},
	}
	wires := []string{`{"+1":"v"}`, `{"01":"v"}`, `{"0":"v"}`, `{"-0":"v"}`, `{"1.0":"v"}`, `{"1e0":"v"}`, `{"0.1":"v"}`, `{"true":"v"}`, `{"false":"v"}`, `{"0":"v","-0":"v"}`, `{"1":"v","1.0":"v"}`, `{"1":"v","1.0000000000000001":"v"}`}
	for _, kind := range kinds {
		t.Run(kind.typ.Name(), func(t *testing.T) {
			attribute := &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: kind.typ}, ElemType: &AttributeExpr{Type: String}}}
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanRuntime})
			require.NoError(t, err)
			for _, wire := range wires {
				t.Run(wire, func(t *testing.T) {
					keyType := reflect.TypeOf(kind.key)
					if kind.typ == Any {
						keyType = reflect.TypeFor[any]()
					}
					native := reflect.New(reflect.MapOf(keyType, reflect.TypeFor[string]()))
					err := json.Unmarshal([]byte(wire), native.Interface(), loom.JSONOptions())
					value, accepted := decodeJSON(plan.root, jsontext.Value(wire))
					require.Equal(t, err == nil, accepted, "actual native map decoder: %v", err)
					if !accepted {
						return
					}
					require.Len(t, value.Entries(), native.Elem().Len())
					for _, entry := range value.Entries() {
						key, ok := entry.Key.Scalar()
						require.True(t, ok)
						require.Equal(t, reflect.TypeOf(kind.key), reflect.TypeOf(key), "decoder must retain native key width")
						require.True(t, native.Elem().MapIndex(reflect.ValueOf(key)).IsValid())
					}
				})
			}
		})
	}
}
