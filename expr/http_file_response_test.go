package expr_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestHTTPFileResponseValidation(t *testing.T) {
	tests := []struct {
		name    string
		dsl     func()
		wantErr string
	}{
		{
			name: "explicit GET and HEAD routes",
			dsl: fileResponseDSL(func() {
				GET("/files/{id}")
				HEAD("/files/{id}")
				FileResponse()
			}),
		},
		{
			name: "rejects POST",
			dsl: fileResponseDSL(func() {
				POST("/files/{id}")
				FileResponse()
			}),
			wantErr: `FileResponse supports only explicit GET or HEAD routes. Got "POST".`,
		},
		{
			name: "rejects encoded body",
			dsl: func() {
				Service("files", func() {
					Method("download", func() {
						Result(String)
						HTTP(func() {
							GET("/files")
							FileResponse()
							Response(StatusOK, func() {
								Body(func() {
									Attribute("value", String)
								})
							})
						})
					})
				})
			},
			wantErr: "Cannot define a response body when endpoint uses FileResponse.",
		},
		{
			name: "rejects raw stream escape hatch",
			dsl: fileResponseDSL(func() {
				GET("/files/{id}")
				FileResponse()
				SkipResponseBodyEncodeDecode()
			}),
			wantErr: "Endpoint cannot use FileResponse with SkipResponseBodyEncodeDecode.",
		},
		{
			name: "rejects raw request body escape hatch",
			dsl: fileResponseDSL(func() {
				GET("/files/{id}")
				FileResponse()
				SkipRequestBodyEncodeDecode()
			}),
			wantErr: "Endpoint cannot use FileResponse with SkipRequestBodyEncodeDecode.",
		},
		{
			name: "rejects application status owned by ServeContent",
			dsl: fileResponseDSL(func() {
				GET("/files/{id}")
				FileResponse()
				Response(StatusAccepted)
			}),
			wantErr: "FileResponse requires exactly one untagged 200 application response",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.wantErr == "" {
				root := expr.RunDSL(t, test.dsl)
				endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
				require.True(t, endpoint.FileResponse)
				require.Len(t, endpoint.Routes, 2)
				require.Equal(t, []string{"GET", "HEAD"}, []string{
					endpoint.Routes[0].Method,
					endpoint.Routes[1].Method,
				})
				require.Len(t, endpoint.Responses, 1)
				require.Equal(t, StatusOK, endpoint.Responses[0].StatusCode)
				return
			}
			err := expr.RunInvalidDSL(t, test.dsl)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestHTTPFileResponseRejectsViewedResultTypes(t *testing.T) {
	tests := []struct {
		name string
		dsl  func()
	}{
		{name: "implicit default view", dsl: fileResponseViewedResultDSL(false, "")},
		{name: "explicit default view", dsl: fileResponseViewedResultDSL(false, "default")},
		{name: "explicit alternate view", dsl: fileResponseViewedResultDSL(true, "summary")},
		{name: "multiple views without selection", dsl: fileResponseViewedResultDSL(true, "")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, test.dsl)
			require.ErrorContains(t, err, "Endpoint cannot use FileResponse when method result type defines views.")
		})
	}
}

func TestHTTPFileResponseAllowsTextContentTypeForMetadataResult(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("files", func() {
			Method("download", func() {
				Result(func() {
					Attribute("etag", String)
					Required("etag")
				})
				HTTP(func() {
					GET("/files")
					FileResponse()
					Response(func() {
						ContentType("text/plain")
						Header("etag:ETag")
					})
				})
			})
		})
	})
	require.Equal(t, "text/plain", root.API.HTTP.Services[0].HTTPEndpoints[0].Responses[0].ContentType)
}

func TestHTTPFileResponseRejectsTransportOwnedResultHeaders(t *testing.T) {
	for _, header := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Last-Modified"} {
		t.Run(header, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, fileResponseResultHeaderDSL(header))
			require.ErrorContains(t, err, `FileResponse result attribute "metadata" cannot map to transport-owned response header "`+header+`".`)
		})
	}
}

func fileResponseResultHeaderDSL(header string) func() {
	return func() {
		Service("files", func() {
			Method("download", func() {
				Result(func() {
					Attribute("metadata", String)
					Required("metadata")
				})
				HTTP(func() {
					GET("/files")
					FileResponse()
					Response(func() {
						Header("metadata:" + header)
					})
				})
			})
		})
	}
}

func fileResponseDSL(httpDSL func()) func() {
	return func() {
		Service("files", func() {
			Method("download", func() {
				Payload(func() {
					Attribute("id", String)
				})
				Result(func() {
					Attribute("etag", String)
				})
				HTTP(httpDSL)
			})
		})
	}
}

func fileResponseViewedResultDSL(multipleViews bool, selectedView string) func() {
	return func() {
		metadata := ResultType("application/vnd.loom.file-metadata+json", func() {
			Attributes(func() {
				Attribute("etag", String)
				Attribute("name", String)
			})
			View("default", func() {
				Attribute("etag")
			})
			if multipleViews {
				View("summary", func() {
					Attribute("name")
				})
			}
		})

		Service("files", func() {
			Method("download", func() {
				if selectedView == "" {
					Result(metadata)
				} else {
					Result(metadata, func() {
						View(selectedView)
					})
				}
				HTTP(func() {
					GET("/files")
					FileResponse()
				})
			})
		})
	}
}
