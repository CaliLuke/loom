package debug

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
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

	server := httptest.NewServer(mux)
	defer server.Close()

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 50 {
				resp, err := http.Get(server.URL + "/debug?debug-logs=on")
				if err != nil {
					t.Errorf("toggle on failed: %v", err)
					return
				}
				if err := resp.Body.Close(); err != nil {
					t.Errorf("close toggle on response body: %v", err)
					return
				}
				resp, err = http.Get(server.URL + "/debug?debug-logs=off")
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
				resp, err := http.Get(server.URL + "/work")
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
