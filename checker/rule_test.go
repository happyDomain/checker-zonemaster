package checker

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	sdk "git.happydns.org/checker-sdk-go/checker"
)

// fakeObs is a minimal ObservationGetter for tests. If err is non-nil, Get
// returns it; otherwise, it JSON-roundtrips data into dest.
type fakeObs struct {
	data any
	err  error
}

func (f *fakeObs) Get(_ context.Context, _ sdk.ObservationKey, dest any) error {
	if f.err != nil {
		return f.err
	}
	b, err := json.Marshal(f.data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

func (f *fakeObs) GetRelated(_ context.Context, _ sdk.ObservationKey) ([]sdk.RelatedObservation, error) {
	return nil, nil
}

func TestLevelToStatus(t *testing.T) {
	cases := []struct {
		level string
		want  sdk.Status
	}{
		{"CRITICAL", sdk.StatusCrit},
		{"ERROR", sdk.StatusCrit},
		{"critical", sdk.StatusCrit}, // case-insensitive
		{"WARNING", sdk.StatusWarn},
		{"NOTICE", sdk.StatusInfo},
		{"INFO", sdk.StatusInfo},
		{"DEBUG", sdk.StatusInfo},
		{"", sdk.StatusUnknown},
		{"BANANA", sdk.StatusUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			if got := levelToStatus(tc.level); got != tc.want {
				t.Errorf("levelToStatus(%q) = %v, want %v", tc.level, got, tc.want)
			}
		})
	}
}

func TestWorstStatus(t *testing.T) {
	// Severity ordering used by worstStatus:
	// Error > Crit > Warn > Info > OK > Unknown
	cases := []struct {
		a, b, want sdk.Status
	}{
		{sdk.StatusOK, sdk.StatusOK, sdk.StatusOK},
		{sdk.StatusOK, sdk.StatusInfo, sdk.StatusInfo},
		{sdk.StatusInfo, sdk.StatusWarn, sdk.StatusWarn},
		{sdk.StatusWarn, sdk.StatusCrit, sdk.StatusCrit},
		{sdk.StatusCrit, sdk.StatusError, sdk.StatusError},
		{sdk.StatusError, sdk.StatusCrit, sdk.StatusError},
		{sdk.StatusUnknown, sdk.StatusOK, sdk.StatusOK},
		{sdk.StatusUnknown, sdk.StatusUnknown, sdk.StatusUnknown},
	}
	for _, tc := range cases {
		if got := worstStatus(tc.a, tc.b); got != tc.want {
			t.Errorf("worstStatus(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestFilterByModules(t *testing.T) {
	results := []ZonemasterTestResult{
		{Module: "DNSSEC", Message: "a"},
		{Module: "Delegation", Message: "b"},
		{Module: "dnssec", Message: "c"},
		{Module: "Syntax", Message: "d"},
	}

	t.Run("matches case-insensitively", func(t *testing.T) {
		got := filterByModules(results, []string{"dnssec"})
		if len(got) != 2 {
			t.Fatalf("got %d results, want 2: %+v", len(got), got)
		}
		if got[0].Message != "a" || got[1].Message != "c" {
			t.Errorf("unexpected results: %+v", got)
		}
	})

	t.Run("multiple modules", func(t *testing.T) {
		got := filterByModules(results, []string{"delegation", "syntax"})
		if len(got) != 2 {
			t.Errorf("got %d, want 2", len(got))
		}
	})

	t.Run("empty modules returns nil", func(t *testing.T) {
		if got := filterByModules(results, nil); got != nil {
			t.Errorf("got %+v, want nil", got)
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		if got := filterByModules(results, []string{"nope"}); len(got) != 0 {
			t.Errorf("got %+v, want empty", got)
		}
	})
}

func TestValidateZonemasterOptions(t *testing.T) {
	cases := []struct {
		name    string
		opts    sdk.CheckerOptions
		wantErr string // substring; empty means no error expected
	}{
		{"empty opts", sdk.CheckerOptions{}, ""},
		{"empty url", sdk.CheckerOptions{"zonemasterAPIURL": ""}, ""},
		{"valid http", sdk.CheckerOptions{"zonemasterAPIURL": "http://localhost:5000/api"}, ""},
		{"valid https", sdk.CheckerOptions{"zonemasterAPIURL": "https://zonemaster.net/api"}, ""},
		{"non-string", sdk.CheckerOptions{"zonemasterAPIURL": 42}, "must be a string"},
		{"bad scheme", sdk.CheckerOptions{"zonemasterAPIURL": "ftp://x/api"}, "http or https"},
		{"no host", sdk.CheckerOptions{"zonemasterAPIURL": "http:///api"}, "must include a host"},
		{"unparseable", sdk.CheckerOptions{"zonemasterAPIURL": "http://[::1"}, "zonemasterAPIURL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateZonemasterOptions(tc.opts)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tc.wantErr != "" && err == nil:
				t.Errorf("expected error containing %q, got nil", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestNormLevel(t *testing.T) {
	cases := map[string]string{
		"":         "",
		"info":     "INFO",
		"WaRnInG":  "WARNING",
		"CRITICAL": "CRITICAL",
	}
	for in, want := range cases {
		if got := normLevel(in); got != want {
			t.Errorf("normLevel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCategoryRuleEvaluate_NoData(t *testing.T) {
	r := &categoryRule{name: "zonemaster.dnssec", modules: []string{"dnssec"}}
	obs := &fakeObs{data: ZonemasterData{Results: nil}}

	states := r.Evaluate(context.Background(), obs, nil)
	if len(states) != 1 {
		t.Fatalf("got %d states, want 1", len(states))
	}
	if states[0].Status != sdk.StatusUnknown {
		t.Errorf("status = %v, want StatusUnknown", states[0].Status)
	}
	if states[0].Code != "zonemaster.dnssec.not_tested" {
		t.Errorf("code = %q", states[0].Code)
	}
}

func TestCategoryRuleEvaluate_ObservationError(t *testing.T) {
	r := &categoryRule{name: "zonemaster.dnssec", modules: []string{"dnssec"}}
	obs := &fakeObs{err: errors.New("boom")}

	states := r.Evaluate(context.Background(), obs, nil)
	if len(states) != 1 {
		t.Fatalf("got %d states, want 1", len(states))
	}
	if states[0].Status != sdk.StatusError {
		t.Errorf("status = %v, want StatusError", states[0].Status)
	}
	if states[0].Code != "zonemaster.observation_error" {
		t.Errorf("code = %q", states[0].Code)
	}
}

func TestCategoryRuleEvaluate_AllOK(t *testing.T) {
	r := &categoryRule{name: "zonemaster.dnssec", modules: []string{"dnssec"}}
	obs := &fakeObs{data: ZonemasterData{Results: []ZonemasterTestResult{
		{Module: "dnssec", Level: "INFO", Message: "ok1"},
		{Module: "dnssec", Level: "NOTICE", Message: "ok2"},
		{Module: "delegation", Level: "ERROR", Message: "ignored, wrong module"},
	}}}

	states := r.Evaluate(context.Background(), obs, nil)
	if len(states) != 1 {
		t.Fatalf("got %d states, want 1 (summary only): %+v", len(states), states)
	}
	if states[0].Status != sdk.StatusOK {
		t.Errorf("status = %v, want StatusOK", states[0].Status)
	}
	if got, _ := states[0].Meta["total"].(int); got != 2 {
		t.Errorf("total = %d, want 2", got)
	}
}

func TestCategoryRuleEvaluate_MixedSeverities(t *testing.T) {
	r := &categoryRule{name: "zonemaster.dnssec", modules: []string{"dnssec"}}
	obs := &fakeObs{data: ZonemasterData{Results: []ZonemasterTestResult{
		{Module: "DNSSEC", Level: "INFO", Message: "i"},
		{Module: "dnssec", Level: "WARNING", Message: "w", Testcase: "tc-w"},
		{Module: "dnssec", Level: "ERROR", Message: "e", Testcase: "tc-e"},
		{Module: "dnssec", Level: "CRITICAL", Message: "c", Testcase: "tc-c"},
	}}}

	states := r.Evaluate(context.Background(), obs, nil)
	// Expect 1 summary + 3 issue states (warning + error + critical).
	if len(states) != 4 {
		t.Fatalf("got %d states, want 4: %+v", len(states), states)
	}

	summary := states[0]
	if summary.Status != sdk.StatusCrit {
		t.Errorf("summary status = %v, want StatusCrit", summary.Status)
	}
	if got, _ := summary.Meta["critical"].(int); got != 1 {
		t.Errorf("critical = %d, want 1", got)
	}
	if got, _ := summary.Meta["error"].(int); got != 1 {
		t.Errorf("error = %d, want 1", got)
	}
	if got, _ := summary.Meta["warning"].(int); got != 1 {
		t.Errorf("warning = %d, want 1", got)
	}
	if got, _ := summary.Meta["info"].(int); got != 1 {
		t.Errorf("info = %d, want 1", got)
	}

	// Issue states: codes should be dotted, lowercased levels.
	wantCodes := map[string]bool{
		"zonemaster.dnssec.warning":  false,
		"zonemaster.dnssec.error":    false,
		"zonemaster.dnssec.critical": false,
	}
	for _, s := range states[1:] {
		if _, ok := wantCodes[s.Code]; !ok {
			t.Errorf("unexpected issue code: %q", s.Code)
			continue
		}
		wantCodes[s.Code] = true
		if s.Subject == "" {
			t.Errorf("issue state %q missing Subject", s.Code)
		}
	}
	for code, seen := range wantCodes {
		if !seen {
			t.Errorf("missing issue state for %q", code)
		}
	}
}

func TestRulesContainsAllCategories(t *testing.T) {
	got := Rules()
	wantNames := []string{
		"zonemaster.dnssec",
		"zonemaster.delegation",
		"zonemaster.consistency",
		"zonemaster.connectivity",
		"zonemaster.nameserver",
		"zonemaster.syntax",
		"zonemaster.zone",
		"zonemaster.address",
		"zonemaster.basic",
	}
	if len(got) != len(wantNames) {
		t.Fatalf("Rules() returned %d rules, want %d", len(got), len(wantNames))
	}
	seen := map[string]bool{}
	for _, r := range got {
		seen[r.Name()] = true
		if r.Description() == "" {
			t.Errorf("rule %q has empty description", r.Name())
		}
	}
	for _, n := range wantNames {
		if !seen[n] {
			t.Errorf("Rules() missing %q", n)
		}
	}
}
