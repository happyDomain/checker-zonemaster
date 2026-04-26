//go:build standalone

package checker

import (
	"errors"
	"net/http"
	"strings"

	sdk "git.happydns.org/checker-sdk-go/checker"
)

func (p *zonemasterProvider) RenderForm() []sdk.CheckerOptionField {
	return []sdk.CheckerOptionField{
		{
			Id:          "domainName",
			Type:        "string",
			Label:       "Domain name to check",
			Placeholder: "example.com",
			Required:    true,
			Description: "Fully-qualified domain name to submit to the Zonemaster engine.",
		},
		{
			Id:          "profile",
			Type:        "string",
			Label:       "Profile",
			Placeholder: "default",
			Description: "Zonemaster test profile to apply (engine-defined; usually \"default\").",
		},
		{
			Id:          "language",
			Type:        "string",
			Label:       "Result language",
			Placeholder: "en",
			Description: "Language for human-readable test messages (en, fr, de, es, sv, da, fi, nb, nl, pt).",
		},
		{
			Id:          "zonemasterAPIURL",
			Type:        "string",
			Label:       "Zonemaster API URL",
			Placeholder: "https://zonemaster.net/api",
			Description: "JSON-RPC endpoint of the Zonemaster backend to query.",
		},
	}
}

func (p *zonemasterProvider) ParseForm(r *http.Request) (sdk.CheckerOptions, error) {
	domainName := strings.TrimSpace(r.FormValue("domainName"))
	if domainName == "" {
		return nil, errors.New("domainName is required")
	}
	domainName = strings.TrimSuffix(domainName, ".")

	opts := sdk.CheckerOptions{
		"domainName": domainName,
	}

	if v := strings.TrimSpace(r.FormValue("profile")); v != "" {
		opts["profile"] = v
	}
	if v := strings.TrimSpace(r.FormValue("language")); v != "" {
		opts["language"] = v
	}
	if v := strings.TrimSpace(r.FormValue("zonemasterAPIURL")); v != "" {
		opts["zonemasterAPIURL"] = strings.TrimSuffix(v, "/")
	}

	return opts, nil
}
