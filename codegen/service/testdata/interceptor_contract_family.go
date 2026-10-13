package testdata

import (
	"strings"

	"github.com/CaliLuke/loom/dsl"
)

// InterceptorContractStorageDSL exercises shared signatures with method-local storage.
func InterceptorContractStorageDSL(mode string) func() {
	return func() {
		firstType, secondType := interceptorContractStorageTypes(mode)
		audit := dsl.Interceptor("Audit", func() {
			dsl.ReadPayload(func() {
				dsl.Attribute("value")
			})
			dsl.WritePayload(func() {
				dsl.Attribute("value")
			})
			dsl.ReadResult(func() {
				dsl.Attribute("value")
			})
			dsl.WriteResult(func() {
				dsl.Attribute("value")
			})
		})
		dsl.Service("contracts", func() {
			if !strings.Contains(mode, "disjoint") {
				dsl.ServerInterceptor(audit)
				dsl.ClientInterceptor(audit)
			}
			for _, name := range []string{"first", "second"} {
				dsl.Method(name, func() {
					if strings.Contains(mode, "disjoint") {
						if name == "first" {
							dsl.ServerInterceptor(audit)
						} else {
							dsl.ClientInterceptor(audit)
						}
					}
					fields := interceptorContractStorageFields(mode, name, firstType, secondType)
					dsl.Payload(fields)
					dsl.Result(fields)
					dsl.HTTP(func() {
						dsl.POST("/" + name)
					})
				})
			}
		})
	}
}

func interceptorContractStorageTypes(mode string) (any, any) {
	var firstType, secondType any = dsl.String, dsl.String
	if strings.Contains(mode, "nominal") {
		firstType = dsl.Type("FirstValue", func() {
			dsl.Attribute("text", dsl.String)
		})
		secondType = firstType
		if strings.Contains(mode, "incompatible") {
			secondType = dsl.Type("SecondValue", func() {
				dsl.Attribute("text", dsl.String)
			})
		}
	}
	if mode == "union" {
		firstType = dsl.Type("Choice", dsl.OneOf(dsl.String, dsl.Int))
		secondType = firstType
	}
	if mode == "custom" {
		firstType, secondType = dsl.Int64, dsl.Int64
	}
	return firstType, secondType
}

func interceptorContractStorageFields(mode, name string, firstType, secondType any) func() {
	return func() {
		typ := firstType
		if name == "second" {
			typ = secondType
		}
		if name == "second" && (mode == "incompatible" || mode == "incompatible-disjoint") {
			typ = dsl.Int
		}
		dsl.Attribute("value", typ, func() {
			if mode == "custom" {
				dsl.Meta("struct:field:type", "time.Duration", "time")
			}
			if mode == "nullable" || (strings.Contains(mode, "nullable-incompatible") && name == "second") {
				dsl.Nullable()
			}
			if name == "second" && mode == "renamed" {
				dsl.Meta("struct:field:name", "StoredValue")
			}
		})
		if (name == "second" && (mode == "required" || mode == "custom" || mode == "nominal" || mode == "nullable" || mode == "union")) || (name == "first" && mode == "required-reversed") {
			dsl.Required("value")
		}
	}
}
