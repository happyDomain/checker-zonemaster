package checker

import sdk "git.happydns.org/checker-sdk-go/checker"

// nameserverRule covers Zonemaster's nameserver test module (server
// behaviour, EDNS posture, unknown RR handling).
func nameserverRule() sdk.CheckRule {
	return &categoryRule{
		name:        "zonemaster.nameserver",
		description: "Zonemaster nameserver tests (server behaviour, EDNS, unknown RR handling).",
		modules:     []string{"nameserver"},
	}
}
