package checker

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	sdk "git.happydns.org/checker-sdk-go/checker"
)

// Rule returns a new zonemaster check rule.
func Rule() sdk.CheckRule {
	return &zonemasterRule{}
}

type zonemasterRule struct{}

func (r *zonemasterRule) Name() string { return "zonemaster" }

func (r *zonemasterRule) Description() string {
	return "Runs Zonemaster DNS validation tests against the zone"
}

func (r *zonemasterRule) ValidateOptions(opts sdk.CheckerOptions) error {
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

func (r *zonemasterRule) Evaluate(ctx context.Context, obs sdk.ObservationGetter, opts sdk.CheckerOptions) []sdk.CheckState {
	var data ZonemasterData
	if err := obs.Get(ctx, ObservationKeyZonemaster, &data); err != nil {
		return []sdk.CheckState{{
			Status:  sdk.StatusError,
			Message: fmt.Sprintf("Failed to get Zonemaster data: %v", err),
			Code:    "zonemaster_error",
		}}
	}

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
		return []sdk.CheckState{{
			Status:  sdk.StatusCrit,
			Message: statusLine,
			Code:    "zonemaster_errors",
			Meta:    meta,
		}}
	}

	if warningCount > 0 {
		return []sdk.CheckState{{
			Status:  sdk.StatusWarn,
			Message: fmt.Sprintf("%d warning(s) found", warningCount),
			Code:    "zonemaster_warnings",
			Meta:    meta,
		}}
	}

	return []sdk.CheckState{{
		Status:  sdk.StatusOK,
		Message: fmt.Sprintf("All checks passed (%d checks)", len(data.Results)),
		Code:    "zonemaster_ok",
		Meta:    meta,
	}}
}
