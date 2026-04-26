package checker

import sdk "git.happydns.org/checker-sdk-go/checker"

// syntaxRule covers Zonemaster's syntax test module (domain name syntax,
// hostname legality).
func syntaxRule() sdk.CheckRule {
	return &categoryRule{
		name:        "zonemaster.syntax",
		description: "Zonemaster syntax tests (domain name syntax, hostname legality).",
		modules:     []string{"syntax"},
	}
}
