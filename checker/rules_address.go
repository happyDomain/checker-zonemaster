package checker

import sdk "git.happydns.org/checker-sdk-go/checker"

// addressRule covers Zonemaster's address test module (IP addresses of
// nameservers, private/reserved ranges, IPv6 coverage).
func addressRule() sdk.CheckRule {
	return &categoryRule{
		name:        "zonemaster.address",
		description: "Zonemaster address tests (IP addresses of nameservers, private/reserved ranges).",
		modules:     []string{"address"},
	}
}
