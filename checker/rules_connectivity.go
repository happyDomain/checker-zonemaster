package checker

import sdk "git.happydns.org/checker-sdk-go/checker"

// connectivityRule covers Zonemaster's connectivity test module (UDP/TCP
// reachability of authoritative servers, AS diversity).
func connectivityRule() sdk.CheckRule {
	return &categoryRule{
		name:        "zonemaster.connectivity",
		description: "Zonemaster connectivity tests (reachability of authoritative servers over UDP/TCP, AS diversity).",
		modules:     []string{"connectivity"},
	}
}
