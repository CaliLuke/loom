package debug

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDebugLogToggleConcurrentAccess(t *testing.T) {
	debugLogs.Store(false)
	t.Cleanup(func() {
		debugLogs.Store(false)
	})

	mux := http.NewServeMux()
	MountDebugLogEnabler(mux)
	mux.Handle("/work", HTTP()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	server := httptest.NewTestServer(t, mux)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 50 {
				resp, err := server.Client().Get(server.URL + "/debug?debug-logs=on")
				if err != nil {
					t.Errorf("toggle on failed: %v", err)
					return
				}
				if err := resp.Body.Close(); err != nil {
					t.Errorf("close toggle on response body: %v", err)
					return
				}
				resp, err = server.Client().Get(server.URL + "/debug?debug-logs=off")
				if err != nil {
					t.Errorf("toggle off failed: %v", err)
					return
				}
				if err := resp.Body.Close(); err != nil {
					t.Errorf("close toggle off response body: %v", err)
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for range 50 {
				resp, err := server.Client().Get(server.URL + "/work")
				if err != nil {
					t.Errorf("work request failed: %v", err)
					return
				}
				if err := resp.Body.Close(); err != nil {
					t.Errorf("close work response body: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestMountPprofHandlersIncludesGoroutineLeakProfile(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	MountPprofHandlers(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/pprof/goroutineleak?debug=1", nil)
	mux.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "goroutineleak profile")
}
