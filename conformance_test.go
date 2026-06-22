package unleash

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.klarlabs.de/rollops/pkg/flagconformance"
	"go.klarlabs.de/rollops/pkg/plugin"
)

// fakeUnleash returns a feature whose production environment already has a
// flexibleRollout strategy, then accepts strategy PUT and on/off POST.
func fakeUnleash(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"environments":[{"name":"production","strategies":[{"id":"s1","name":"flexibleRollout"}]}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestConformance(t *testing.T) {
	flagconformance.Run(t, func() (plugin.FlagProvider, error) {
		srv := fakeUnleash(t)
		return Provider{BaseURL: srv.URL, Token: "tok", Project: "default", HTTP: srv.Client()}, nil
	}, plugin.FlagChange{Flag: "checkout", Environment: "production"})
}
