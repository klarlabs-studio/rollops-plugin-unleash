package unleash

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.klarlabs.de/rollops/pkg/plugin"
)

func TestApplyFlag_UpdatesStrategyAndEnables(t *testing.T) {
	var calls []string
	var putBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet:
			// Feature with a production env that already has a flexibleRollout strategy.
			json.NewEncoder(w).Encode(feature{Environments: []struct {
				Name       string `json:"name"`
				Strategies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"strategies"`
			}{
				{Name: "production", Strategies: []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{{ID: "strat-1", Name: "flexibleRollout"}}},
			}})
		case r.Method == http.MethodPut:
			b, _ := io.ReadAll(r.Body)
			json.Unmarshal(b, &putBody)
			w.WriteHeader(200)
		default:
			w.WriteHeader(200)
		}
	}))
	defer srv.Close()

	p := Provider{BaseURL: srv.URL, Token: "tok", Project: "default", HTTP: srv.Client()}
	err := p.ApplyFlag(context.Background(), plugin.FlagChange{Flag: "checkout", Environment: "production", Percentage: 25})
	if err != nil {
		t.Fatalf("ApplyFlag: %v", err)
	}
	// Strategy PUT must carry rollout=25; enable POST must hit .../production/on.
	params, _ := putBody["parameters"].(map[string]any)
	if params["rollout"] != "25" {
		t.Errorf("rollout param = %v, want 25", params["rollout"])
	}
	joined := strings.Join(calls, " | ")
	if !strings.Contains(joined, "PUT") || !strings.Contains(joined, "/environments/production/strategies/strat-1") {
		t.Errorf("expected strategy PUT, calls: %s", joined)
	}
	if !strings.HasSuffix(calls[len(calls)-1], "/environments/production/on") {
		t.Errorf("expected enable on, last call: %s", calls[len(calls)-1])
	}
}

func TestApplyFlag_CreatesStrategyWhenMissing(t *testing.T) {
	var posted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(feature{Environments: []struct {
				Name       string `json:"name"`
				Strategies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"strategies"`
			}{{Name: "production"}}}) // env present, no strategies
			return
		}
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/strategies") {
			posted = true
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := Provider{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	if err := p.ApplyFlag(context.Background(), plugin.FlagChange{Flag: "checkout", Environment: "production", Percentage: 50}); err != nil {
		t.Fatalf("ApplyFlag: %v", err)
	}
	if !posted {
		t.Error("expected a strategy POST when the environment has no flexibleRollout strategy")
	}
}

func TestApplyFlag_UnknownEnvironmentErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(feature{}) // no environments
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := Provider{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	if err := p.ApplyFlag(context.Background(), plugin.FlagChange{Flag: "checkout", Environment: "production", Percentage: 10}); err == nil {
		t.Fatal("missing environment must error")
	}
}

func TestApplyFlag_RequiresToken(t *testing.T) {
	p := Provider{BaseURL: "http://x"}
	if err := p.ApplyFlag(context.Background(), plugin.FlagChange{Flag: "f", Environment: "e"}); err == nil {
		t.Fatal("missing token must error")
	}
}
