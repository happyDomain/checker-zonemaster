package checker

import sdk "git.happydns.org/checker-sdk-go/checker"

// zoneRule covers Zonemaster's zone test module (SOA values, MX presence,
// mandatory records at the apex).
func zoneRule() sdk.CheckRule {
	return &categoryRule{
		name:        "zonemaster.zone",
		description: "Zonemaster zone tests (SOA values, MX presence, mandatory records).",
		modules:     []string{"zone"},
	}
}
