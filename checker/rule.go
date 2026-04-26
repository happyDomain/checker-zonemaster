package checker

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	sdk "git.happydns.org/checker-sdk-go/checker"
)

// Rules returns the full list of CheckRules exposed by the Zonemaster checker.
// Each rule narrows the Zonemaster results to a single test category so
// callers can see at a glance which category passed and which did not,
// instead of squashing every Zonemaster message into a single monolithic
// state. The Zonemaster-returned severity (INFO/NOTICE/WARNING/ERROR/
// CRITICAL) is treated as a raw input coming from Zonemaster's own
// judgement; each rule maps it onto happyDomain's CheckState.Status.
func Rules() []sdk.CheckRule {
	return []sdk.CheckRule{
		dnssecRule(),
		delegationRule(),
		consistencyRule(),
		connectivityRule(),
		nameserverRule(),
		syntaxRule(),
		zoneRule(),
		addressRule(),
		basicRule(),
	}
}

// Rule returns the legacy single-rule view of the Zonemaster checker.
//
// Deprecated: use Rules() for per-category CheckRules. This wrapper is kept
// so existing callers that only expect a single rule keep compiling.
func Rule() sdk.CheckRule { return &legacyRule{} }

type legacyRule struct{}

func (r *legacyRule) Name() string { return "zonemaster" }

func (r *legacyRule) Description() string {
	return "Runs Zonemaster DNS validation tests against the zone (aggregate view)."
}

func (r *legacyRule) ValidateOptions(opts sdk.CheckerOptions) error {
	return validateZonemasterOptions(opts)
}

func (r *legacyRule) Evaluate(ctx context.Context, obs sdk.ObservationGetter, _ sdk.CheckerOptions) []sdk.CheckState {
	data, errSt := loadZonemasterData(ctx, obs)
	if errSt != nil {
		return []sdk.CheckState{*errSt}
	}
	return []sdk.CheckState{summarizeAll(data)}
}

// ── shared helpers ────────────────────────────────────────────────────────────

// validateZonemasterOptions validates the options accepted by the Zonemaster
// checker. Shared across rules that implement OptionsValidator.
func validateZonemasterOptions(opts sdk.CheckerOptions) error {
	if v, ok := opts["zonemasterAPIURL"]; ok {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("zonemasterAPIURL must be a string")
		}
		if s != "" {
			u, err := url.Parse(s)
			if err != nil {
				return fmt.Errorf("zonemasterAPIURL: %w", err)
			}
			if u.Scheme != "http" && u.Scheme != "https" {
				return fmt.Errorf("zonemasterAPIURL must use http or https scheme")
			}
			if u.Host == "" {
				return fmt.Errorf("zonemasterAPIURL must include a host")
			}
		}
	}
	return nil
}

// loadZonemasterData fetches the Zonemaster observation. On error, returns a
// CheckState the caller should emit to short-circuit its rule.
func loadZonemasterData(ctx context.Context, obs sdk.ObservationGetter) (*ZonemasterData, *sdk.CheckState) {
	var data ZonemasterData
	if err := obs.Get(ctx, ObservationKeyZonemaster, &data); err != nil {
		return nil, &sdk.CheckState{
			Status:  sdk.StatusError,
			Message: fmt.Sprintf("failed to load Zonemaster observation: %v", err),
			Code:    "zonemaster.observation_error",
		}
	}
	return &data, nil
}

// levelToStatus maps a Zonemaster-returned severity to happyDomain's status.
// Zonemaster's own judgement is treated as raw input; this is happyDomain's
// own mapping onto the SDK status enum.
func levelToStatus(level string) sdk.Status {
	switch strings.ToUpper(level) {
	case "CRITICAL", "ERROR":
		return sdk.StatusCrit
	case "WARNING":
		return sdk.StatusWarn
	case "NOTICE", "INFO":
		return sdk.StatusInfo
	case "DEBUG":
		return sdk.StatusInfo
	default:
		return sdk.StatusUnknown
	}
}

// worstStatus returns the more severe of two statuses. StatusError always
// wins because it means "we could not evaluate".
func worstStatus(a, b sdk.Status) sdk.Status {
	rank := func(s sdk.Status) int {
		switch s {
		case sdk.StatusError:
			return 6
		case sdk.StatusCrit:
			return 5
		case sdk.StatusWarn:
			return 4
		case sdk.StatusInfo:
			return 2
		case sdk.StatusOK:
			return 1
		default:
			return 0
		}
	}
	if rank(a) >= rank(b) {
		return a
	}
	return b
}

// categoryRule is the common shape used by every per-category Zonemaster
// rule: load the observation, filter messages whose module matches one of
// the declared names, map Zonemaster severities onto CheckState.Status,
// and emit a summary state plus one state per WARNING-or-worse message.
// INFO/NOTICE messages are folded into the summary counts so the state
// list stays readable.
type categoryRule struct {
	name        string
	description string
	modules     []string // case-insensitive module names handled by this rule
}

func (r *categoryRule) Name() string        { return r.name }
func (r *categoryRule) Description() string { return r.description }

func (r *categoryRule) Evaluate(ctx context.Context, obs sdk.ObservationGetter, _ sdk.CheckerOptions) []sdk.CheckState {
	data, errSt := loadZonemasterData(ctx, obs)
	if errSt != nil {
		return []sdk.CheckState{*errSt}
	}

	matched := filterByModules(data.Results, r.modules)
	if len(matched) == 0 {
		return []sdk.CheckState{{
			Status:  sdk.StatusUnknown,
			Message: fmt.Sprintf("No %s messages returned by Zonemaster for this zone.", r.name),
			Code:    r.name + ".not_tested",
		}}
	}

	var (
		critCount, errCount, warnCount, noticeCount, infoCount int
		worst                                                  = sdk.StatusOK
		issueStates                                            []sdk.CheckState
	)

	for _, res := range matched {
		lvl := strings.ToUpper(res.Level)
		st := levelToStatus(lvl)
		worst = worstStatus(worst, st)

		switch lvl {
		case "CRITICAL":
			critCount++
		case "ERROR":
			errCount++
		case "WARNING":
			warnCount++
		case "NOTICE":
			noticeCount++
		default:
			infoCount++
		}

		if st == sdk.StatusCrit || st == sdk.StatusWarn {
			issueStates = append(issueStates, sdk.CheckState{
				Status:  st,
				Message: res.Message,
				Code:    r.name + "." + strings.ToLower(lvl),
				Subject: res.Testcase,
				Meta: map[string]any{
					"module":   res.Module,
					"testcase": res.Testcase,
					"level":    lvl,
				},
			})
		}
	}

	summary := sdk.CheckState{
		Status: worst,
		Code:   r.name + ".summary",
		Meta: map[string]any{
			"total":    len(matched),
			"critical": critCount,
			"error":    errCount,
			"warning":  warnCount,
			"notice":   noticeCount,
			"info":     infoCount,
		},
	}

	switch {
	case critCount+errCount > 0:
		summary.Message = fmt.Sprintf("%d error(s), %d warning(s) reported by Zonemaster (%d checks).", critCount+errCount, warnCount, len(matched))
	case warnCount > 0:
		summary.Message = fmt.Sprintf("%d warning(s) reported by Zonemaster (%d checks).", warnCount, len(matched))
	default:
		summary.Status = sdk.StatusOK
		summary.Message = fmt.Sprintf("No issues reported by Zonemaster (%d checks).", len(matched))
	}

	return append([]sdk.CheckState{summary}, issueStates...)
}

// filterByModules returns the subset of results whose Module matches any of
// the given module names (case-insensitive).
func filterByModules(results []ZonemasterTestResult, modules []string) []ZonemasterTestResult {
	if len(modules) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(modules))
	for _, m := range modules {
		set[strings.ToLower(m)] = struct{}{}
	}
	var out []ZonemasterTestResult
	for _, r := range results {
		if _, ok := set[strings.ToLower(r.Module)]; ok {
			out = append(out, r)
		}
	}
	return out
}

// summarizeAll produces the legacy monolithic summary state. Preserved so
// Rule() keeps behaving as before for callers that still use it.
func summarizeAll(data *ZonemasterData) sdk.CheckState {
	var errorCount, warningCount int
	var criticalMsgs []string

	for _, res := range data.Results {
		switch strings.ToUpper(res.Level) {
		case "CRITICAL", "ERROR":
			errorCount++
			if len(criticalMsgs) < 5 {
				criticalMsgs = append(criticalMsgs, res.Message)
			}
		case "WARNING":
			warningCount++
		}
	}

	meta := map[string]any{
		"errorCount":   errorCount,
		"warningCount": warningCount,
		"totalChecks":  len(data.Results),
		"hashId":       data.HashID,
		"createdAt":    data.CreatedAt,
	}

	if errorCount > 0 {
		statusLine := fmt.Sprintf("%d error(s), %d warning(s) found", errorCount, warningCount)
		if len(criticalMsgs) > 0 {
			n := 2
			if len(criticalMsgs) < n {
				n = len(criticalMsgs)
			}
			statusLine += ": " + strings.Join(criticalMsgs[:n], "; ")
		}
		return sdk.CheckState{
			Status:  sdk.StatusCrit,
			Message: statusLine,
			Code:    "zonemaster_errors",
			Meta:    meta,
		}
	}
	if warningCount > 0 {
		return sdk.CheckState{
			Status:  sdk.StatusWarn,
			Message: fmt.Sprintf("%d warning(s) found", warningCount),
			Code:    "zonemaster_warnings",
			Meta:    meta,
		}
	}
	return sdk.CheckState{
		Status:  sdk.StatusOK,
		Message: fmt.Sprintf("All checks passed (%d checks)", len(data.Results)),
		Code:    "zonemaster_ok",
		Meta:    meta,
	}
}
