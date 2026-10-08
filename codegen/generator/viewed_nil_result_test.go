package generator

import (
	"testing"

	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestViewedNilResultBoundary(t *testing.T) {
	t.Run("objects", func(t *testing.T) {
		runDesignHarness(t, "example.com/customizedresults", customizedResultCopiesDSL, viewedNilObjectHarness)
	})
	t.Run("collections", func(t *testing.T) {
		runDesignHarness(t, "example.com/viewedcollections", testdata.ResultCollectionMultipleViewsMethodDSL, viewedNilCollectionHarness)
	})
}

const viewedNilObjectHarness = `package customizedresults

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	svc "example.com/customizedresults/gen/svc"
	svcserver "example.com/customizedresults/gen/http/svc/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

type missingService struct {
	view string
	err error
}

func (s missingService) M1(context.Context) (*svc.MenuM1Result, string, error) {
	return nil, s.view, s.err
}

func (s missingService) M2(context.Context) (*svc.MenuM2Result, error) {
	return nil, s.err
}

func (s missingService) M3(context.Context) (*svc.Menu, string, error) {
	return nil, s.view, s.err
}

func TestMissingObjectConstructors(t *testing.T) {
	for _, view := range []string{"", "default", "tiny", "unknown"} {
		t.Run(view, func(t *testing.T) {
			result, err := svc.NewViewedMenuM1Result(nil, view)
			require.Nil(t, result)
			var fault *loom.ServiceError
			require.ErrorAs(t, err, &fault)
			require.True(t, fault.Fault)
			require.Equal(t, "fault", fault.Name)
		})
	}
}

func TestValidObjectsAndInvalidViews(t *testing.T) {
	for _, view := range []string{"", "default", "tiny"} {
		t.Run(view, func(t *testing.T) {
			result, err := svc.NewViewedMenuM1Result(&svc.MenuM1Result{X: "one"}, view)
			require.NoError(t, err)
			require.Equal(t, "one", *result.Projected.X)
		})
	}
	_, err := svc.NewViewedMenuM1Result(&svc.MenuM1Result{X: "one"}, "unknown")
	var invalid *loom.ServiceError
	require.ErrorAs(t, err, &invalid)
	require.True(t, invalid.Fault)
	require.Equal(t, "fault", invalid.Name)
	var cause *loom.ServiceError
	require.ErrorAs(t, errors.Unwrap(invalid), &cause)
	require.Equal(t, loom.InvalidEnumValue, cause.Name)
}

func TestMissingObjectEndpoints(t *testing.T) {
	for _, view := range []string{"default", "tiny"} {
		t.Run(view, func(t *testing.T) {
			s := missingService{view: view}
			for _, endpoint := range []loom.Endpoint{svc.NewM1Endpoint(s), svc.NewM2Endpoint(s), svc.NewM3Endpoint(s)} {
				_, err := endpoint(context.Background(), nil)
				var fault *loom.ServiceError
				require.ErrorAs(t, err, &fault)
				require.True(t, fault.Fault)
			}
		})
	}
	sentinel := errors.New("service failed")
	s := missingService{err: sentinel}
	_, err := svc.NewM1Endpoint(s)(context.Background(), nil)
	require.ErrorIs(t, err, sentinel)
}

func TestMissingObjectHTTPResponse(t *testing.T) {
	mux := loomhttp.NewMuxer()
	svcserver.Mount(mux, svcserver.New(svc.NewEndpoints(missingService{view: "default"}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	for _, path := range []string{"/m1", "/m2", "/m3"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			require.Equal(t, http.StatusInternalServerError, response.Code)
			var problem loomhttp.ProblemResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
			require.Equal(t, "fault", problem.Code)
		})
	}
}
`

const viewedNilCollectionHarness = `package viewedcollections

import (
	"testing"

	"github.com/stretchr/testify/require"
	svc "example.com/viewedcollections/gen/result_collection_multiple_views_method"
)

func TestEmptyCollections(t *testing.T) {
	for _, view := range []string{"", "default", "tiny"} {
		t.Run(view, func(t *testing.T) {
			for _, items := range []svc.MultipleViewsCollection{nil, {}} {
				result, err := svc.NewViewedMultipleViewsCollection(items, view)
				require.NoError(t, err)
				require.Empty(t, result.Projected)
			}
		})
	}
}
`
