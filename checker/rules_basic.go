package checker

import sdk "git.happydns.org/checker-sdk-go/checker"

// basicRule covers Zonemaster's basic/system test modules (initial
// reachability and fundamental pre-conditions for running the other test
// categories).
func basicRule() sdk.CheckRule {
	return &categoryRule{
		name:        "zonemaster.basic",
		description: "Zonemaster basic tests (initial reachability and fundamental requirements).",
		modules:     []string{"basic", "system"},
	}
}
