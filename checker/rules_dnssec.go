package checker

import sdk "git.happydns.org/checker-sdk-go/checker"

// dnssecRule covers Zonemaster's DNSSEC test module (signatures, NSEC/NSEC3,
// DS/DNSKEY coherence, algorithm posture).
func dnssecRule() sdk.CheckRule {
	return &categoryRule{
		name:        "zonemaster.dnssec",
		description: "Zonemaster DNSSEC tests (signatures, NSEC/NSEC3, DS/DNSKEY coherence).",
		modules:     []string{"dnssec"},
	}
}
