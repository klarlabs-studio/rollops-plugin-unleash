// Package unleash is a Rollops feature-flag provider plugin backed by Unleash's
// Admin API. It drives a feature toggle's enabled state and the rollout
// percentage of its flexibleRollout (gradual rollout) strategy to match a
// rollout's progressive steps, so an Unleash flag tracks a Rollops canary in
// lockstep.
package unleash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"go.klarlabs.de/rollops/pkg/plugin"
)

// Provider talks to Unleash's Admin API. BaseURL, Token, and Project come from
// the plugin's environment (see Config); Environment is supplied per call by
// Rollops as the Unleash environment name (e.g. "production").
type Provider struct {
	BaseURL string // e.g. https://unleash.example.com
	Token   string // Admin API token (Authorization: <token>)
	Project string // Unleash project id (default "default")
	HTTP    *http.Client
}

func (p Provider) client() *http.Client {
	if p.HTTP != nil {
		return p.HTTP
	}
	return http.DefaultClient
}

func (p Provider) project() string {
	if p.Project != "" {
		return p.Project
	}
	return "default"
}

// ApplyFlag sets the feature's enabled state in the environment and writes the
// rollout percentage into its flexibleRollout strategy, creating that strategy
// if the environment does not yet have one.
func (p Provider) ApplyFlag(ctx context.Context, c plugin.FlagChange) error {
	if p.Token == "" {
		return fmt.Errorf("unleash: UNLEASH_TOKEN is required")
	}
	if err := p.setRollout(ctx, c.Flag, c.Environment, c.Percentage); err != nil {
		return err
	}
	return p.setEnabled(ctx, c.Flag, c.Environment, !c.Disabled)
}

type feature struct {
	Environments []struct {
		Name       string `json:"name"`
		Strategies []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"strategies"`
	} `json:"environments"`
}

// setRollout upserts the flexibleRollout strategy's rollout parameter for the
// feature in the given environment.
func (p Provider) setRollout(ctx context.Context, flag, env string, pct int) error {
	base := fmt.Sprintf("%s/api/admin/projects/%s/features/%s", p.BaseURL, p.project(), url.PathEscape(flag))
	var f feature
	if err := p.do(ctx, http.MethodGet, base, nil, &f); err != nil {
		return fmt.Errorf("unleash: lookup feature %q: %w", flag, err)
	}
	body := map[string]any{
		"name":       "flexibleRollout",
		"parameters": map[string]string{"rollout": strconv.Itoa(pct), "stickiness": "default", "groupId": flag},
	}
	for _, e := range f.Environments {
		if e.Name != env {
			continue
		}
		for _, s := range e.Strategies {
			if s.Name == "flexibleRollout" {
				u := fmt.Sprintf("%s/environments/%s/strategies/%s", base, url.PathEscape(env), url.PathEscape(s.ID))
				if err := p.do(ctx, http.MethodPut, u, body, nil); err != nil {
					return fmt.Errorf("unleash: update rollout strategy: %w", err)
				}
				return nil
			}
		}
		// Environment exists but has no flexibleRollout strategy yet: create one.
		u := fmt.Sprintf("%s/environments/%s/strategies", base, url.PathEscape(env))
		if err := p.do(ctx, http.MethodPost, u, body, nil); err != nil {
			return fmt.Errorf("unleash: create rollout strategy: %w", err)
		}
		return nil
	}
	return fmt.Errorf("unleash: feature %q has no environment %q", flag, env)
}

// setEnabled turns the feature on or off in the environment.
func (p Provider) setEnabled(ctx context.Context, flag, env string, on bool) error {
	state := "off"
	if on {
		state = "on"
	}
	u := fmt.Sprintf("%s/api/admin/projects/%s/features/%s/environments/%s/%s",
		p.BaseURL, p.project(), url.PathEscape(flag), url.PathEscape(env), state)
	if err := p.do(ctx, http.MethodPost, u, nil, nil); err != nil {
		return fmt.Errorf("unleash: set feature %s in %q: %w", state, env, err)
	}
	return nil
}

func (p Provider) do(ctx context.Context, method, u string, body, out any) error {
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", p.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
