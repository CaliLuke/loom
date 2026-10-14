package openapiv3_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestFilesRejectsEquivalentPathTemplatesAcrossMethods(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("widgets", func() {
		})
		Service("widgets", func() {
			Method("get", func() {
				Payload(func() {
					Attribute("id", String)
					Required("id")
				})
				HTTP(func() {
					GET("/widgets/{id}")
					Param("id")
				})
			})
			Method("create", func() {
				Payload(func() {
					Attribute("slug", String)
					Required("slug")
				})
				HTTP(func() {
					POST("/widgets/{slug}")
					Param("slug")
				})
			})
		})
	})

	_, err := openapiv3.Files(root)
	require.ErrorContains(t, err, "path templates")
	require.Nil(t, openapiv3.New(root))
}

func TestFilesRejectsEquivalentParameterTemplatesWithStaticSuffixAcrossMethods(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("items", func() {
		})
		Service("items", func() {
			Method("json", func() {
				Payload(func() {
					Attribute("id", String)
					Required("id")
				})
				HTTP(func() {
					GET("/items/{id}.json")
					Param("id")
				})
			})
			Method("jsonAlt", func() {
				Payload(func() {
					Attribute("slug", String)
					Required("slug")
				})
				HTTP(func() {
					POST("/items/{slug}.json")
					Param("slug")
				})
			})
		})
	})

	_, err := openapiv3.Files(root)
	require.ErrorContains(t, err, "path templates")
}

func TestFilesRejectsDuplicateOperationIDs(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("widgets", func() {
		})
		Service("widgets", func() {
			Method("get", func() {
				Meta("openapi:operationId", "same")
				HTTP(func() {
					GET("/widgets")
				})
			})
			Method("list", func() {
				Meta("openapi:operationId", "same")
				HTTP(func() {
					GET("/gadgets")
				})
			})
		})
	})

	_, err := openapiv3.Files(root)
	require.ErrorContains(t, err, `operationId "same"`)
	require.Nil(t, openapiv3.New(root))
}

func TestFilesRejectsFileServerRouteCollision(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		API("assets", func() {
		})
		Service("assets", func() {
			Files("/assets", "dist")
			Method("get", func() {
				HTTP(func() {
					GET("/assets")
				})
			})
		})
	})

	require.ErrorContains(t, err, "GET /assets")
	require.ErrorContains(t, err, `HTTP endpoint "get"`)
	require.ErrorContains(t, err, "file server dist")
}

func TestFilesRejectsConflictingExplicitMediaTypeNames(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		first := Type("First", func() {
			Attribute("first", String)
		})
		second := Type("Second", func() {
			Attribute("second", Int)
		})
		API("media", func() {
		})
		Service("media", func() {
			Method("first", func() {
				Result(first, func() {
					Meta("openapi:component:mediaType", "Shared")
				})
				HTTP(func() {
					GET("/first")
					Response(StatusOK, func() {
						ContentType("application/json")
					})
				})
			})
			Method("second", func() {
				Result(second, func() {
					Meta("openapi:component:mediaType", "Shared")
				})
				HTTP(func() {
					GET("/second")
					Response(StatusOK, func() {
						ContentType("application/json")
					})
				})
			})
		})
	})
	root.API.Meta = map[string][]string{"openapi:version": {"3.2"}}

	_, err := openapiv3.Files(root)
	require.ErrorContains(t, err, `mediaTypes component "Shared"`)
	require.Nil(t, openapiv3.New(root))
}

func TestFiles31DoesNotRunOpenAPI32MediaIdentityPass(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		first := Type("First", func() {
			Attribute("first", String)
		})
		second := Type("Second", func() {
			Attribute("second", Int)
		})
		API("media", func() {
		})
		Service("media", func() {
			Method("first", func() {
				Result(first, func() {
					Meta("openapi:component:mediaType", "Shared")
				})
				HTTP(func() {
					GET("/first")
					Response(StatusOK, func() {
						ContentType("application/json")
					})
				})
			})
			Method("second", func() {
				Result(second, func() {
					Meta("openapi:component:mediaType", "Shared")
				})
				HTTP(func() {
					GET("/second")
					Response(StatusOK, func() {
						ContentType("application/json")
					})
				})
			})
		})
	})
	root.API.Meta = map[string][]string{"openapi:version": {"3.1"}}

	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	require.NotEmpty(t, files)
	var rendered bytes.Buffer
	require.NoError(t, files[0].Sections[0].Write(&rendered))
	require.NotContains(t, rendered.String(), "mediaTypes")
}

func TestFilesAcceptsSamePathAcrossMethodsAndEscapedBraceLiteral(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("widgets", func() {
		})
		Service("widgets", func() {
			Method("get", func() {
				Payload(func() {
					Attribute("id", String)
					Required("id")
				})
				HTTP(func() {
					GET("/widgets/{id}")
					Param("id")
				})
			})
			Method("purge", func() {
				Payload(func() {
					Attribute("id", String)
					Required("id")
				})
				HTTP(func() {
					Route("purge", "/widgets/{id}")
					Param("id")
				})
			})
			Method("literal", func() {
				HTTP(func() {
					GET("/widgets/%7Bid%7D")
				})
			})
		})
	})

	_, err := openapiv3.Files(root)
	require.NoError(t, err)
}
