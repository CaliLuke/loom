package expr_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// TestHTTPRouteOwnership checks physical mount admission and deliberate sharing
// across transport handlers, server scopes, synthetic routes and static files.
func TestHTTPRouteOwnership(t *testing.T) {
	cases := []struct {
		name     string
		design   func()
		conflict bool
	}{
		{
			name:     "http same service",
			conflict: true,
			design: func() {
				Service("one", func() {
					Method("add", func() {
						HTTP(func() {
							GET("/x")
						})
					})
					Method("sub", func() {
						HTTP(func() {
							GET("/x")
						})
					})
				})
			},
		},
		{
			name:     "http two services",
			conflict: true,
			design: func() {
				routeOwnershipHTTP("one", "/x", false)
				routeOwnershipHTTP("two", "/x", false)
			},
		},
		{
			name:     "jsonrpc two services",
			conflict: true,
			design: func() {
				routeOwnershipRPC("one", "add")
				routeOwnershipRPC("two", "sub")
			},
		},
		{
			name:     "http and jsonrpc",
			conflict: true,
			design: func() {
				routeOwnershipHTTP("one", "/rpc", true)
				routeOwnershipRPC("two", "sub")
			},
		},
		{
			name:     "wildcard names",
			conflict: true,
			design: func() {
				routeOwnershipParameter("one", "id")
				routeOwnershipParameter("two", "key")
			},
		},
		{
			name: "different methods",
			design: func() {
				routeOwnershipHTTP("one", "/x", false)
				routeOwnershipHTTP("two", "/x", true)
			},
		},
		{
			name: "shared jsonrpc dispatcher",
			design: func() {
				routeOwnershipRPC("one", "add", "sub")
			},
		},
		{
			name: "disjoint servers",
			design: func() {
				API("routes", func() {
					Server("first", func() {
						Services("one")
					})
					Server("second", func() {
						Services("two")
					})
				})
				routeOwnershipHTTP("one", "/x", false)
				routeOwnershipHTTP("two", "/x", false)
			},
		},
		{
			name:     "unselected service still owns its mount",
			conflict: true,
			design: func() {
				API("routes", func() {
					Server("selected", func() {
						Services("other")
					})
				})
				routeOwnershipHTTP("other", "/other", false)
				Service("one", func() {
					for _, method := range []string{"first", "second"} {
						Method(method, func() {
							HTTP(func() {
								GET("/x")
							})
						})
					}
				})
			},
		},
		{
			name:     "omitted server services hosts all",
			conflict: true,
			design: func() {
				API("routes", func() {
					Server("all", func() {
					})
				})
				routeOwnershipHTTP("one", "/x", false)
				routeOwnershipHTTP("two", "/x", false)
			},
		},
		{
			name:     "mixed dispatcher synthetic GET",
			conflict: true,
			design: func() {
				routeOwnershipHTTP("one", "/rpc", false)
				routeOwnershipSSE("events", true)
			},
		},
		{
			name:     "events stream synthetic GET",
			conflict: true,
			design: func() {
				routeOwnershipHTTP("one", "/rpc", false)
				routeOwnershipSSE("events/stream", false)
			},
		},
		{
			name: "ordinary SSE does not add GET",
			design: func() {
				routeOwnershipHTTP("one", "/rpc", false)
				routeOwnershipSSE("events", false)
			},
		},
		{
			name:     "different verbs collide through preflight",
			conflict: true,
			design: func() {
				routeOwnershipCORS("one", false)
				routeOwnershipCORS("two", true)
			},
		},
		{
			name:     "explicit OPTIONS belongs to another service",
			conflict: true,
			design: func() {
				routeOwnershipCORS("one", false)
				Service("two", func() {
					Method("options", func() {
						HTTP(func() {
							OPTIONS("/x")
						})
					})
				})
			},
		},
		{
			name: "explicit OPTIONS shares local preflight owner",
			design: func() {
				Service("one", func() {
					HTTP(func() {
						CORS(func() {
							Origin("*")
						})
					})
					Method("get", func() {
						HTTP(func() {
							GET("/x")
						})
					})
					Method("options", func() {
						HTTP(func() {
							OPTIONS("/x")
						})
					})
				})
			},
		},
		{
			name:     "static file versus endpoint",
			conflict: true,
			design: func() {
				routeOwnershipFiles("/asset", "asset.txt")
				routeOwnershipHTTP("one", "/asset", false)
			},
		},
		{
			name:     "directory synthetic root versus endpoint",
			conflict: true,
			design: func() {
				routeOwnershipFiles("/assets/{*path}", "assets")
				routeOwnershipHTTP("one", "/assets/", false)
			},
		},
		{
			name: "directory literal child has defined precedence",
			design: func() {
				routeOwnershipFiles("/assets/{*path}", "assets")
				routeOwnershipHTTP("one", "/assets/special", false)
			},
		},
		{
			name: "literal and parameter have defined precedence",
			design: func() {
				routeOwnershipParameter("one", "id")
				routeOwnershipHTTP("two", "/items/special", false)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr.SetupTestDSL(t)
			require.True(t, eval.Execute(tc.design, nil), eval.Context.Error())
			err := eval.RunDSL()
			if tc.conflict {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestHTTPRouteOwnershipPreparationPhase requires the paths used for admission
// to match finalized mount paths, including inherited and absolute paths.
func TestHTTPRouteOwnershipPreparationPhase(t *testing.T) {
	cases := []struct {
		name  string
		paths []string
		route string
		want  string
	}{
		{name: "omitted child path", route: "/child", want: "/api/parents/{id}/child"},
		{name: "explicit child root", paths: []string{"/"}, route: "/child", want: "/api/parents/{id}/child"},
		{name: "explicit child base", paths: []string{"/nested"}, route: "/child", want: "/api/parents/{id}/nested/child"},
		{name: "absolute service base", paths: []string{"//outside"}, route: "/child", want: "/outside/child"},
		{name: "absolute endpoint", route: "//outside", want: "/outside"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := &expr.HTTPExpr{Path: "/api"}
			parent := &expr.HTTPServiceExpr{
				ServiceExpr: &expr.ServiceExpr{Name: "parent"},
				Root:        root,
				Paths:       []string{"/parents"},
			}
			canonical := &expr.HTTPEndpointExpr{
				Service:    parent,
				MethodExpr: &expr.MethodExpr{Name: "show"},
			}
			canonical.Routes = []*expr.RouteExpr{{Method: "GET", Path: "/{id}", Endpoint: canonical}}
			parent.HTTPEndpoints = []*expr.HTTPEndpointExpr{canonical}
			child := &expr.HTTPServiceExpr{
				ServiceExpr: &expr.ServiceExpr{Name: "child"},
				Root:        root,
				ParentName:  "parent",
				Paths:       tc.paths,
			}
			root.Services = []*expr.HTTPServiceExpr{parent, child}
			route := &expr.RouteExpr{Method: "GET", Path: tc.route, Endpoint: &expr.HTTPEndpointExpr{Service: child}}
			parent.Prepare()
			child.Prepare()
			prepared := route.FullPaths()
			for _, service := range []any{parent, child} {
				if finalizer, ok := service.(eval.Finalizer); ok {
					finalizer.Finalize()
				}
			}
			finalized := route.FullPaths()
			require.Equal(t, []string{tc.want}, finalized, "finalized parent/absolute semantics")
			require.Equal(t, finalized, prepared, "admission and mounting must resolve the same paths")
		})
	}
}

// TestHTTPRouteOwnershipParentGraph exercises guarded path expansion during
// actual canonical-parameter inheritance, before root validation runs.
func TestHTTPRouteOwnershipParentGraph(t *testing.T) {
	for _, size := range []int{1, 2, 3} {
		t.Run(fmt.Sprintf("cycle_%d", size), func(t *testing.T) {
			expr.SetupTestDSL(t)
			require.True(t, eval.Execute(func() {
				for i := range size {
					Service(fmt.Sprintf("service%d", i), func() {
						HTTP(func() {
							Path("/base")
							Parent(fmt.Sprintf("service%d", (i+1)%size))
						})
						Method("show", func() {
							Payload(func() {
								Attribute("id", String)
							})
							HTTP(func() {
								GET("/{id}")
							})
						})
					})
				}
			}, nil))
			err := eval.RunDSL()
			require.ErrorContains(t, err, "HTTP parent cycle")
			require.NotContains(t, err.Error(), "panic")
		})
	}
}

// TestHTTPMountPlanBindings checks concrete synthetic registrations and retains
// distinct route bindings even when the same HTTP handler would be selected.
func TestHTTPMountPlanBindings(t *testing.T) {
	t.Run("multiple file bases", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			Service("files", func() {
				HTTP(func() {
					Path("/one")
					Path("/two")
					CORS(func() {
						Origin("*")
					})
				})
				Files("/assets/{*path}", "assets")
			})
		})
		plan, err := root.API.HTTP.Services[0].MountPlan()
		require.NoError(t, err)
		paths := make([]string, 0, len(plan.Routes))
		prefixes := make([]string, 0, len(plan.Routes))
		for _, route := range plan.Routes {
			paths = append(paths, route.Path)
			prefixes = append(prefixes, route.FilePrefix)
		}
		require.Equal(t, []string{"/one/assets/", "/one/assets/{*path}", "/two/assets/", "/two/assets/{*path}"}, paths)
		require.Equal(t, []string{"/one/assets", "/one/assets", "/two/assets", "/two/assets"}, prefixes)
		require.Empty(t, plan.Preflights, "file mounts do not acquire CORS implicitly")
	})
	t.Run("same handler different binding", func(t *testing.T) {
		expr.SetupTestDSL(t)
		require.True(t, eval.Execute(func() {
			Service("one", func() {
				Method("call", func() {
					Payload(func() {
						Attribute("id", String)
						Attribute("key", String)
					})
					HTTP(func() {
						GET("/{id}")
						GET("/{key}")
					})
				})
			})
		}, nil))
		require.ErrorContains(t, eval.RunDSL(), "HTTP route")
	})
	t.Run("identical handler registration", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			Service("one", func() {
				Method("call", func() {
					HTTP(func() {
						GET("/x")
						GET("/x")
					})
				})
			})
		})
		plan, err := root.API.HTTP.Services[0].MountPlan()
		require.NoError(t, err)
		require.Len(t, plan.Routes, 1)
	})
}

func routeOwnershipHTTP(service, route string, post bool) {
	Service(service, func() {
		Method("call", func() {
			HTTP(func() {
				if post {
					POST(route)
				} else {
					GET(route)
				}
			})
		})
	})
}

func routeOwnershipRPC(service string, methods ...string) {
	Service(service, func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		for _, method := range methods {
			Method(method, func() {
				JSONRPC(func() {
				})
			})
		}
	})
}

func routeOwnershipParameter(service, parameter string) {
	Service(service, func() {
		Method("call", func() {
			Payload(func() {
				Attribute(parameter, String)
			})
			HTTP(func() {
				GET("/items/{" + parameter + "}")
				Param(parameter)
			})
		})
	})
}

func routeOwnershipSSE(method string, mixed bool) {
	Service("stream", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method(method, func() {
			StreamingResult(String)
			JSONRPC(func() {
				ServerSentEvents()
			})
		})
		if mixed {
			Method("ordinary", func() {
				JSONRPC(func() {
				})
			})
		}
	})
}

func routeOwnershipCORS(service string, post bool) {
	Service(service, func() {
		HTTP(func() {
			CORS(func() {
				Origin("*")
			})
		})
		Method("call", func() {
			HTTP(func() {
				if post {
					POST("/x")
				} else {
					GET("/x")
				}
			})
		})
	})
}

func routeOwnershipFiles(route, filename string) {
	Service("files", func() {
		Files(route, filename)
	})
}
