package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/codegen/service/testdata"
)

func TestErrorRemedyInitData(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ErrorRemedyMethodDSL)
	services := NewServicesData(root)
	method := services.Get("ErrorRemedyMethod").Method("Show")
	require.NotNil(t, method)
	require.Len(t, method.Errors, 1)

	erro := method.Errors[0]
	assert.NotEmpty(t, erro.Name)
	assert.Equal(t, "bad_request.fix", erro.RemedyCode)
	assert.Equal(t, "The request is invalid.", erro.SafeMessage)
	assert.Equal(t, "Correct the payload and retry.", erro.RetryHint)
}
