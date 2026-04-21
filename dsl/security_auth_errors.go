package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func ensureAuthError(scope eval.Expression, name string) {
	switch actual := scope.(type) {
	case *expr.RootExpr:
		if actual.Error(name) != nil {
			return
		}
		actual.Errors = append(actual.Errors, &expr.ErrorExpr{
			Name: name,
			AttributeExpr: &expr.AttributeExpr{
				Type: expr.ErrorResult,
			},
		})
	case *expr.ServiceExpr:
		if actual.Error(name) != nil {
			return
		}
		actual.Errors = append(actual.Errors, &expr.ErrorExpr{
			Name: name,
			AttributeExpr: &expr.AttributeExpr{
				Type: expr.ErrorResult,
			},
		})
	case *expr.MethodExpr:
		if actual.Error(name) != nil {
			return
		}
		actual.Errors = append(actual.Errors, &expr.ErrorExpr{
			Name: name,
			AttributeExpr: &expr.AttributeExpr{
				Type: expr.ErrorResult,
			},
		})
	}
}

func ensureHTTPAuthError(errors *[]*expr.HTTPErrorExpr, parent eval.Expression, name string, code int, description string) {
	for _, err := range *errors {
		if err.Name == name {
			return
		}
	}
	if inherited := inheritedCompatibleHTTPAuthError(parent, name, code); inherited != nil {
		dup := inherited.Dup()
		dup.Response.Parent = parent
		*errors = append(*errors, dup)
		return
	}
	*errors = append(*errors, &expr.HTTPErrorExpr{
		Name: name,
		Response: &expr.HTTPResponseExpr{
			StatusCode:  code,
			Description: description,
			Parent:      parent,
		},
	})
}

func inheritedCompatibleHTTPAuthError(parent eval.Expression, name string, code int) *expr.HTTPErrorExpr {
	switch actual := parent.(type) {
	case *expr.HTTPServiceExpr:
		return compatibleHTTPAuthError(actual.Root.Errors, name, code)
	case *expr.HTTPEndpointExpr:
		if err := compatibleHTTPAuthError(actual.Service.HTTPErrors, name, code); err != nil {
			return err
		}
		return compatibleHTTPAuthError(actual.Service.Root.Errors, name, code)
	default:
		return nil
	}
}

func compatibleHTTPAuthError(errors []*expr.HTTPErrorExpr, name string, code int) *expr.HTTPErrorExpr {
	for _, err := range errors {
		if err == nil || err.Name != name || err.Response == nil {
			continue
		}
		if err.Response.StatusCode == code {
			return err
		}
	}
	return nil
}

// APIKey defines the attribute used to provide the API key to an endpoint
// secured with API keys. The parameters and usage of APIKey are the same as the
// Attribute function except that it accepts an extra first argument
// corresponding to the name of the API key security scheme.
//
// The generated code produced by Loom uses the value of the corresponding
// payload field to set the API key value.
//
// APIKey must appear in Payload or Type.
//
// Example:
//
//	Method("secured_read", func() {
//	    Security(APIKeyAuth)
//	    Payload(func() {
//	        APIKey("api_key", "key", String, "API key used to perform authorization")
//	        Required("key")
//	    })
//	    Result(String)
//	    HTTP(func() {
//	        GET("/")
//	        Param("key:k") // Provide the key as a query string param "k"
//	    })
//	})
//
//	Method("secured_write", func() {
//	    Security(APIKeyAuth)
//	    Payload(func() {
//	        APIKey("api_key", "key", String, "API key used to perform authorization")
//	        Attribute("data", String, "Data to be written")
//	        Required("key", "data")
//	    })
//	    HTTP(func() {
//	        POST("/")
//	        Header("key:Authorization") // Provide the key in Authorization header (default)
//	    })
