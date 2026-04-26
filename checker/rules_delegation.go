package checker

import sdk "git.happydns.org/checker-sdk-go/checker"

// delegationRule covers Zonemaster's delegation test module (parent/child NS
// agreement, glue correctness, referral integrity).
func delegationRule() sdk.CheckRule {
	return &categoryRule{
		name:        "zonemaster.delegation",
		description: "Zonemaster delegation tests (parent/child NS agreement, glue, referrals).",
		modules:     []string{"delegation"},
	}
}
