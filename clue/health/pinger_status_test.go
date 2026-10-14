package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPingerSuccessfulStatusRange(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent, 299, http.StatusMultipleChoices, http.StatusNotModified, http.StatusBadRequest, http.StatusInternalServerError} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			t.Cleanup(host.Close)
			err := NewPinger("dependency", host.Listener.Addr().String()).Ping(context.Background())
			if status >= http.StatusOK && status < http.StatusMultipleChoices {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "returned status "+strconv.Itoa(status))
			}
		})
	}
}
