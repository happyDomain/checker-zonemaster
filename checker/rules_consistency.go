package checker

import sdk "git.happydns.org/checker-sdk-go/checker"

// consistencyRule covers Zonemaster's consistency test module (SOA serial,
// NS set, zone content identical across authoritative servers).
func consistencyRule() sdk.CheckRule {
	return &categoryRule{
		name:        "zonemaster.consistency",
		description: "Zonemaster consistency tests (SOA serial, NS set, zone content across servers).",
		modules:     []string{"consistency"},
	}
}
