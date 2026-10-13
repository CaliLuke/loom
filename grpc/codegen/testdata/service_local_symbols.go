package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

type grpcLocalSymbolTypes struct {
	frame, value, choice, failure expr.UserType
	key                           *expr.SchemeExpr
	view                          *expr.ResultTypeExpr
}

// GRPCLocalSymbols declares named services with request/response metadata and
// each stream direction. Authored names contain zz to distinguish them from
// generator-owned local identifiers in the scope inventory.
func GRPCLocalSymbols(names ...string) {
	API("zzlocals", func() {
	})
	types := defineGRPCLocalSymbolTypes()
	for _, name := range names {
		Service(name, func() {
			Method("zzchoose", func() {
				Payload(types.choice)
				Result(types.choice)
				Error("zzfailure", types.failure)
				GRPC(func() {
					Response("zzfailure", CodeInvalidArgument)
				})
			})
			Method("zzsecure", func() {
				Security(types.key)
				Payload(func() {
					APIKey("zzkey", "zzkey", String)
					Field(1, "zzvalue", String)
					Required("zzkey")
				})
				StreamingPayload(types.frame)
				StreamingResult(types.view)
				GRPC(func() {
				})
			})
			Method("zzecho", func() {
				Payload(types.value)
				Result(types.value)
				GRPC(func() {
					Metadata(func() {
						Attribute("zztoken")
						Attribute("zzheader")
						Attribute("zztrailer")
					})
					Response(CodeOK, func() {
						Headers(func() {
							Attribute("zztoken")
							Attribute("zzheader")
						})
						Trailers(func() {
							Attribute("zztrailer")
						})
					})
				})
			})
			Method("zzwatch", func() {
				Payload(types.frame)
				StreamingResult(types.frame)
				GRPC(func() {
				})
			})
			Method("zzpush", func() {
				StreamingPayload(types.frame)
				Result(types.frame)
				GRPC(func() {
				})
			})
			Method("zzexchange", func() {
				StreamingPayload(types.frame)
				StreamingResult(types.frame)
				GRPC(func() {
				})
			})
		})
	}
}

// GRPCLocalSymbolsCustom couples a newly escaped service alias with a relocated
// body package and metadata fields that request the same and suffixed names.
func GRPCLocalSymbolsCustom() {
	API("zzcustom", func() {
	})
	body := Type("ZzBody", func() {
		Meta("struct:pkg:path", "mdsvc")
		Field(1, "zzvalue", String)
	})
	value := Type("ZzValue", func() {
		Field(1, "zzbody", body)
		Attribute("mdsvc", String)
		Attribute("mdsvc2", String)
		Attribute("mdsvc3", String)
	})
	Service("md", func() {
		Method("zzecho", func() {
			Payload(value)
			Result(value)
			GRPC(func() {
				Metadata(func() {
					Attribute("mdsvc")
					Attribute("mdsvc2")
					Attribute("mdsvc3")
				})
				Response(CodeOK, func() {
					Headers(func() {
						Attribute("mdsvc")
						Attribute("mdsvc2")
					})
					Trailers(func() {
						Attribute("mdsvc3")
					})
				})
			})
		})
	})
}

// GRPCLocalSymbolsAggregate isolates example-file aliases from payload types.
// Use p/psvcsvr as a parent-compiling control and ordinary/ordinarysvr as the
// existing header/reference disagreement witness.
func GRPCLocalSymbolsAggregate(names ...string) {
	API("zzaggregate", func() {
	})
	grpcLocalSymbolsServices(names...)
}

// GRPCLocalSymbolsHosts exercises file-local bindings for distinct server
// subsets without changing the shared transport identities between renders.
func GRPCLocalSymbolsHosts() {
	API("zzhosts", func() {
		Server("first", func() {
			Services("p", "psvcsvr")
			Host("local", func() {
				URI("grpc://localhost:8081")
			})
		})
		Server("second", func() {
			Services("req")
			Host("local", func() {
				URI("grpc://localhost:8082")
			})
		})
	})
	grpcLocalSymbolsServices("p", "psvcsvr", "req")
}

func grpcLocalSymbolsServices(names ...string) {
	for _, name := range names {
		Service(name, func() {
			Method("zzecho", func() {
				Payload(String)
				Result(String)
				GRPC(func() {
				})
			})
		})
	}
}

func defineGRPCLocalSymbolTypes() grpcLocalSymbolTypes {
	frame := Type("ZzFrame", func() {
		Field(1, "zzvalue", String, func() {
			MinLength(1)
		})
	})
	value := Type("ZzValue", func() {
		Field(1, "zzbody", frame)
		Attribute("zztoken", String)
		Attribute("zzheader", String)
		Attribute("zztrailer", String)
	})
	key := APIKeySecurity("zzkey", func() {
	})
	other := Type("ZzOther", func() {
		Field(1, "zzcount", Int)
	})
	choice := Type("ZzChoice", OneOf(frame, other))
	failure := Type("ZzFailure", func() {
		ErrorName(1, "zzname", String)
		Field(2, "zzdetail", String)
		Required("zzname")
	})
	view := ResultType("application/vnd.zzview", func() {
		TypeName("ZzViewed")
		Field(1, "zzframe", frame)
		View("default", func() {
			Attribute("zzframe")
		})
	})
	return grpcLocalSymbolTypes{frame: frame, value: value, key: key, choice: choice, failure: failure, view: view}
}
