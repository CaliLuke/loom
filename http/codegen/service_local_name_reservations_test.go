package codegen

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// localNameMarker marks every name that the local name designs give to
// generated code, so that the names of the generated code alone remain.
const localNameMarker = "zz"

// localNameDesigns are the HTTP designs whose generated transport and example
// files the local name tests inspect and compile. A service whose methods
// all stream over WebSocket has client request builders that do not refer
// to the service package, so it covers the locals that would keep an unused
// service package import.
var localNameDesigns = []struct {
	Name   string
	Design func(services ...string)
}{
	{Name: "all-functions", Design: localNameDSL},
	{Name: "websocket", Design: localNameWebSocketDSL},
}

// TestTransportLocalNameReservationsCoverGeneratedLocals checks that
// transportGeneratedLocalNames lists every name that the generated HTTP
// transport and example files declare where a service package imported
// under the same name would not compile.
func TestTransportLocalNameReservationsCoverGeneratedLocals(t *testing.T) {
	for _, c := range localNameDesigns {
		t.Run(c.Name, func(t *testing.T) {
			const modulePath = "example.com/localnames"
			dir := renderLocalNameModule(t, modulePath, c.Design, localNameMarker+"svc")
			for _, name := range testingx.ServicePackageShadowNames(t, dir, modulePath+"/gen", localNameMarker) {
				assert.Truef(t, slices.Contains(transportGeneratedLocalNames, name) || slices.Contains(transportGeneratedImportNames, name),
					"generated local %q is missing from transportGeneratedLocalNames", name)
			}
		})
	}
}

// TestServicePackageNamedLikeLocalCompiles checks that services named after
// every reserved local name import their packages under an alias and that
// the generated module and example compile.
func TestServicePackageNamedLikeLocalCompiles(t *testing.T) {
	for _, c := range localNameDesigns {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				c.Design(transportGeneratedLocalNames...)
			})
			services := CreateHTTPServices(root)
			for _, name := range transportGeneratedLocalNames {
				require.Equal(t, name, services.ServicesData.Get(name).PkgName)
				assert.NotEqual(t, name, services.Get(name).Service.PkgName)
			}

			dir := renderLocalNameModule(t, "example.com/localnamesweep", c.Design, transportGeneratedLocalNames...)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
		})
	}
}

// renderLocalNameModule renders the HTTP transport and the example server
// and CLI of design with the given services to a new module.
func renderLocalNameModule(t *testing.T, modulePath string, design func(...string), services ...string) string {
	t.Helper()
	root := RunHTTPDSL(t, func() {
		design(services...)
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	data := CreateHTTPServices(root)
	renderGeneratedFiles(t, dir, ExampleServerFiles(modulePath+"/gen", data))
	renderGeneratedFiles(t, dir, ExampleCLIFiles(modulePath+"/gen", data))
	return dir
}

// localNameWebSocketDSL declares HTTP services with the given names whose
// methods stream over WebSocket: bidirectional, client and server streams
// without payloads. Every name it gives to generated code contains
// localNameMarker.
func localNameWebSocketDSL(services ...string) {
	var Frame = Type("ZzFrame", func() {
		Attribute("zzvalue", String)
	})
	API("zzapi", func() {})
	for _, name := range services {
		Service(name, func() {
			HTTP(func() {
				Path("/" + name)
			})
			Method("zzexchange", func() {
				StreamingPayload(Frame)
				StreamingResult(Frame)
				HTTP(func() {
					GET("/exchange")
				})
			})
			Method("zzpush", func() {
				StreamingPayload(Frame)
				Result(Frame)
				HTTP(func() {
					GET("/push")
				})
			})
			Method("zzfeed", func() {
				StreamingResult(Frame)
				HTTP(func() {
					GET("/feed")
				})
			})
		})
	}
}

// localNameDSL declares HTTP services with the given names that use every
// kind of generated transport function: path, query, header and cookie
// params, bodies, errors, security, views, skipped bodies, multipart
// requests, WebSocket and server-sent events streams and file servers. Every
// name it gives to generated code contains localNameMarker.
func localNameDSL(services ...string) {
	var Frame = Type("ZzFrame", func() {
		Attribute("zzvalue", String)
	})
	var Item = ResultType("application/vnd.zzitem", func() {
		TypeName("ZzItem")
		Attributes(func() {
			Attribute("zzid", String)
			Attribute("zzname", String)
		})
		View("default", func() {
			Attribute("zzid")
			Attribute("zzname")
		})
		View("zztiny", func() {
			Attribute("zzid")
		})
	})
	var JWTAuth = JWTSecurity("zzjwt", func() {
		Scope("zz:read")
	})
	var KeyAuth = APIKeySecurity("zzkey")
	var BasicAuth = BasicAuthSecurity("zzbasic")
	API("zzapi", func() {})
	for _, name := range services {
		Service(name, func() {
			HTTP(func() {
				Path("/" + name)
			})
			Error("zzunauthorized")
			Method("zzadd", func() {
				Payload(func() {
					Attribute("zzid", String)
					Attribute("zza", Int)
					Attribute("zzh", String)
					Attribute("zzq", ArrayOf(Int))
					Attribute("zzck", String)
					Attribute("zzm", MapOf(String, Int))
				})
				Result(func() {
					Attribute("zzsum", Int)
					Attribute("zzh", String)
					Attribute("zzck", String)
				})
				Error("zzbad")
				HTTP(func() {
					POST("/add/{zzid}")
					Header("zzh")
					Param("zzq")
					Cookie("zzck")
					Body(func() {
						Attribute("zza")
						Attribute("zzm")
					})
					Response(StatusOK, func() {
						Header("zzh")
						Cookie("zzck")
					})
					Response("zzbad", StatusBadRequest)
					Response("zzunauthorized", StatusUnauthorized)
				})
			})
			Method("zzsecure", func() {
				Security(JWTAuth, KeyAuth)
				Payload(func() {
					Token("zztoken", String)
					APIKey("zzkey", "zzkey", String)
					Attribute("zzn", Int)
				})
				Result(Item)
				HTTP(func() {
					GET("/secure")
					Header("zztoken:Authorization")
					Param("zzkey:k")
					Param("zzn")
				})
			})
			Method("zzlogin", func() {
				Security(BasicAuth)
				Payload(func() {
					Username("zzuser", String)
					Password("zzpass", String)
				})
				Result(CollectionOf(Item))
				HTTP(func() {
					POST("/login")
				})
			})
			Method("zzlist", func() {
				Result(ArrayOf(Frame))
				HTTP(func() {
					GET("/list")
				})
			})
			Method("zzraw", func() {
				Payload(func() {
					Attribute("zzn", Int)
				})
				HTTP(func() {
					PUT("/raw")
					SkipRequestBodyEncodeDecode()
					Param("zzn")
				})
			})
			Method("zzdownload", func() {
				Result(func() {
					Attribute("zzlength", Int64)
				})
				HTTP(func() {
					GET("/download")
					SkipResponseBodyEncodeDecode()
					Response(func() {
						Header("zzlength:Content-Length")
					})
				})
			})
			Method("zzupload", func() {
				Payload(func() {
					Attribute("zzname", String)
					Attribute("zzdata", Bytes)
				})
				HTTP(func() {
					POST("/upload")
					MultipartRequest()
				})
			})
			Method("zzexchange", func() {
				Payload(func() {
					Attribute("zzroom", String)
				})
				StreamingPayload(Frame)
				StreamingResult(Frame)
				HTTP(func() {
					GET("/exchange/{zzroom}")
				})
			})
			Method("zzpush", func() {
				StreamingPayload(Frame)
				Result(Frame)
				HTTP(func() {
					GET("/push")
				})
			})
			Method("zzfeed", func() {
				StreamingResult(Item)
				HTTP(func() {
					GET("/feed")
				})
			})
			Method("zzwatch", func() {
				Payload(func() {
					Attribute("zzsince", String)
				})
				StreamingResult(func() {
					Attribute("zzid", String)
					Attribute("zzevent", String)
					Attribute("zzvalue", String)
					Required("zzvalue")
				})
				HTTP(func() {
					GET("/watch")
					Param("zzsince")
					ServerSentEvents(func() {
						SSEEventID("zzid")
						SSEEventType("zzevent")
						SSEEventData("zzvalue")
					})
				})
			})
			Files("/static/{*path}", "zzstatic")
		})
	}
}
