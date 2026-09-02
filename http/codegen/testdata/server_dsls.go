package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var ServerNoPayloadNoResultDSL = func() {
	Service("ServiceNoPayloadNoResult", func() {
		Method("MethodNoPayloadNoResult", func() {
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var ServerCORSPolicyDSL = func() {
	API("cors", func() {
		HTTP(func() {
			CORS(func() {
				Origin("https://app.example.com", func() {
					Methods("GET", "POST")
					Headers("Authorization", "Content-Type")
					ExposeHeaders("X-Request-Id")
					MaxAge(600)
					Credentials()
				})
			})
		})
	})
	Service("ServiceCORS", func() {
		Method("List", func() {
			HTTP(func() {
				GET("/items")
			})
		})
		Method("Create", func() {
			HTTP(func() {
				POST("/items")
			})
		})
	})
}

// ServerCORSOptionsPolicyDSL defines static CORS generation alongside an
// authored OPTIONS endpoint on the same path.
var ServerCORSOptionsPolicyDSL = func() {
	API("cors-options", func() {
		HTTP(func() {
			CORS(func() {
				Origin("https://app.example.com")
			})
		})
	})
	Service("cors-options-service", func() {
		Method("list", func() {
			HTTP(func() {
				GET("/items")
			})
		})
		Method("options", func() {
			HTTP(func() {
				OPTIONS("/items")
			})
		})
	})
}

// ServerRuntimeCORSPolicyDSL defines runtime-provided CORS policy generation.
var ServerRuntimeCORSPolicyDSL = func() {
	API("runtime-cors", func() {
		HTTP(func() {
			RuntimeCORS()
		})
	})
	Service("runtime-cors-service", func() {
		Method("list", func() {
			Result(String)
			HTTP(func() {
				GET("/items")
			})
		})
		Method("options", func() {
			Result(String)
			HTTP(func() {
				OPTIONS("/items")
				Response(StatusOK)
			})
		})
		Method("stream", func() {
			StreamingResult(func() {
				Attribute("value", String)
			})
			HTTP(func() {
				GET("/stream")
				ServerSentEvents()
			})
		})
	})
}

var ServerNoPayloadNoResultWithRedirectDSL = func() {
	Service("ServiceNoPayloadNoResult", func() {
		Method("MethodNoPayloadNoResult", func() {
			HTTP(func() {
				POST("/")
				Redirect("/redirect/dest", StatusMovedPermanently)
			})
		})
	})
}

var ServerPayloadNoResultDSL = func() {
	Service("ServicePayloadNoResult", func() {
		Method("MethodPayloadNoResult", func() {
			Payload(func() {
				Attribute("a", Boolean)
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var ServerPayloadNoResultWithRedirectDSL = func() {
	Service("ServicePayloadNoResult", func() {
		Method("MethodPayloadNoResult", func() {
			Payload(func() {
				Attribute("a", Boolean)
			})
			HTTP(func() {
				POST("/")
				Redirect("/redirect/dest", StatusMovedPermanently)
			})
		})
	})
}

var ServerNoPayloadResultDSL = func() {
	Service("ServiceNoPayloadResult", func() {
		Method("MethodNoPayloadResult", func() {
			Result(func() {
				Attribute("b", Boolean)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK)
			})
		})
	})
}

var ServerPayloadResultDSL = func() {
	Service("ServicePayloadResult", func() {
		Method("MethodPayloadResult", func() {
			Payload(func() {
				Attribute("a", Boolean)
			})
			Result(func() {
				Attribute("b", Boolean)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK)
			})
		})
	})
}

var ServerPayloadResultErrorDSL = func() {
	Service("ServicePayloadResultError", func() {
		Method("MethodPayloadResultError", func() {
			Payload(func() {
				Attribute("a", Boolean)
			})
			Result(func() {
				Attribute("b", Boolean)
			})
			Error("e", func() {
				Attribute("c", Boolean)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK)
				Response("e", func() {
					Code(StatusConflict)
				})
			})
		})
	})
}

var ServerMultiBasesDSL = func() {
	Service("ServiceMultiBases", func() {
		HTTP(func() {
			Path("/base_1")
			Path("/base_2")
		})
		Method("MethodMultiBases", func() {
			Payload(func() {
				Attribute("id", String)
			})
			HTTP(func() {
				GET("/{id}")
			})
		})
	})
}

var ServerMultiEndpointsDSL = func() {
	Service("ServiceMultiEndpoints", func() {
		HTTP(func() {
			Path("/server_multi_endpoints")
		})
		Method("MethodMultiEndpoints1", func() {
			Payload(func() {
				Attribute("id", String)
			})
			HTTP(func() {
				GET("/{id}")
			})
		})
		Method("MethodMultiEndpoints2", func() {
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var ServerFileServerDSL = func() {
	Service("ServiceFileServer", func() {
		HTTP(func() {
			Path("/server_file_server")
		})
		Files("/file1.json", "/path/to/file1.json")
		Files("/file2.json", "/path/to/file2.json")
		Files("/file3.json", "/path/to/file3.json")
	})
}

var ServerFileServerWithRedirectDSL = func() {
	Service("ServiceFileServer", func() {
		HTTP(func() {
			Path("/server_file_server")
		})
		Files("/file1.json", "/path/to/file1.json", func() {
			Redirect("/redirect/dest", StatusMovedPermanently)
		})
		Files("/file2.json", "/path/to/file2.json")
		Files("/file3.json", "/path/to/file3.json")
	})
}

var ServerFileServerRootPathDSL = func() {
	Service("ServiceFileServer", func() {
		HTTP(func() {
			Path("/")
		})
		Files("/file1.json", "file1.json")
		Files("/path/to/file1.json", "file1.json")
		Files("/file2.json", "file1.json")
		Files("/path/to/file2.json", "file1.json")
	})
}

var ServerMixedDSL = func() {
	Service("ServerMixed", func() {
		Method("MethodMixed1", func() {
			Payload(func() {
				Attribute("id", String)
			})
			HTTP(func() {
				GET("/resources1/{id}")
			})
		})
		Method("MethodMixed2", func() {
			Payload(func() {
				Attribute("id", String)
			})
			HTTP(func() {
				GET("/resources2/{id}")
				Redirect("/redirect/dest1", StatusMovedPermanently)
			})
		})
		Files("/file1.json", "/path/to/file1.json")
		Files("/file2.json", "/path/to/file2.json", func() {
			Redirect("/redirect/dest2", StatusMovedPermanently)
		})
	})
}

var ServerMultipartDSL = func() {
	Service("ServiceMultipart", func() {
		Method("MethodMultiBases", func() {
			Payload(String)
			HTTP(func() {
				GET("/")
				MultipartRequest()
			})
		})
	})
}

var ServerMultipleFilesDSL = func() {
	Service("ServiceFileServer", func() {
		Files("/file.json", "/path/to/file.json")
		Files("/", "/path/to/file.json")
		Files("/file.json", "file.json")
		Files("/{wildcard}", "/path/to/folder")
	})
}

var ServerMultipleFilesWithPrefixPathDSL = func() {
	Service("ServiceFileServer", func() {
		HTTP(func() {
			Path("/server_file_server")
		})
		Files("/file.json", "/path/to/file.json")
		Files("/", "/path/to/file.json")
		Files("/file.json", "file.json")
		Files("/{wildcard}", "/path/to/folder")
	})
}

var ServerMultipleFilesWithRedirectDSL = func() {
	Service("ServiceFileServer", func() {
		Files("/file.json", "/path/to/file.json", func() {
			Redirect("/redirect/dest", StatusMovedPermanently)
		})
		Files("/", "/path/to/file.json")
		Files("/file.json", "file.json")
		Files("/{wildcard}", "/path/to/folder")
	})
}

var ServerSimpleRoutingDSL = func() {
	Service("ServiceSimpleRoutingServer", func() {
		Method("server-simple-routing", func() {
			HTTP(func() {
				GET("/simple/routing")
			})
		})
	})
}

var ServerTrailingSlashRoutingDSL = func() {
	Service("ServiceTrailingSlashRoutingServer", func() {
		Method("server-trailing-slash-routing", func() {
			HTTP(func() {
				GET("/trailing/slash/")
			})
		})
	})
}

var ServerSimpleRoutingWithRedirectDSL = func() {
	Service("ServiceSimpleRoutingServer", func() {
		Method("server-simple-routing", func() {
			HTTP(func() {
				GET("/simple/routing")
				Redirect("/redirect/dest", StatusMovedPermanently)
			})
		})
	})
}

var ServerSkipResponseBodyEncodeDecodeDSL = func() {
	Service("ServiceSkipResponseBodyEncodeDecode", func() {
		Method("MethodSkipResponseBodyEncodeDecode", func() {
			Error("internal_error", ErrorResult, "Fault while processing download.")
			HTTP(func() {
				GET("/")
				SkipResponseBodyEncodeDecode()
				Response(StatusOK)
				Response("internal_error", StatusInternalServerError)
			})
		})
	})
}

var ResponseEncoderSkipResponseBodyEncodeDecodeDSL = func() {
	Service("ServiceResponseEncoderSkip", func() {
		Method("MethodResponseEncoderSkip", func() {
			HTTP(func() {
				GET("/")
				SkipResponseBodyEncodeDecode()
				Response(StatusCreated)
			})
		})
	})
}
