package checker

import (
	"time"

	sdk "git.happydns.org/checker-sdk-go/checker"
)

// Version is the checker version reported in CheckerDefinition.Version.
//
// It defaults to "built-in", which is appropriate when the checker package is
// imported directly (built-in or plugin mode). Standalone binaries (like
// main.go) should override this from their own Version variable at the start
// of main(), which makes it easy for CI to inject a version with a single
// -ldflags "-X main.Version=..." flag instead of targeting the nested
// package path.
var Version = "built-in"

// Definition returns the CheckerDefinition for the zonemaster checker.
func (p *zonemasterProvider) Definition() *sdk.CheckerDefinition {
	return &sdk.CheckerDefinition{
		ID:      "zonemaster",
		Name:    "Zonemaster",
		Version: Version,
		Availability: sdk.CheckerAvailability{
			ApplyToDomain: true,
		},
		HasHTMLReport:   true,
		ObservationKeys: []sdk.ObservationKey{ObservationKeyZonemaster},
		Options: sdk.CheckerOptionsDocumentation{
			RunOpts: []sdk.CheckerOptionDocumentation{
				{
					Id:          "domainName",
					Type:        "string",
					Label:       "Domain name to check",
					Required:    true,
					Placeholder: "example.com.",
					AutoFill:    sdk.AutoFillDomainName,
				},
				{
					Id:          "profile",
					Type:        "string",
					Label:       "Profile",
					Placeholder: "default",
					Default:     "default",
				},
			},
			UserOpts: []sdk.CheckerOptionDocumentation{
				{
					Id:      "language",
					Type:    "string",
					Label:   "Result language",
					Default: "en",
					Choices: []string{
						"en", // English
						"fr", // French
						"de", // German
						"es", // Spanish
						"sv", // Swedish
						"da", // Danish
						"fi", // Finnish
						"nb", // Norwegian Bokmål
						"nl", // Dutch
						"pt", // Portuguese
					},
				},
			},
			AdminOpts: []sdk.CheckerOptionDocumentation{
				{
					Id:          "zonemasterAPIURL",
					Type:        "string",
					Label:       "Zonemaster API URL",
					Placeholder: "https://zonemaster.net/api",
					Default:     "https://zonemaster.net/api",
				},
			},
		},
		Rules: Rules(),
		Interval: &sdk.CheckIntervalSpec{
			Min:     12 * time.Hour,
			Max:     30 * 24 * time.Hour,
			Default: 7 * 24 * time.Hour,
		},
	}
}
