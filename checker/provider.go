package checker

import (
	sdk "git.happydns.org/checker-sdk-go/checker"
)

// Provider returns a new zonemaster observation provider.
func Provider() sdk.ObservationProvider {
	return &zonemasterProvider{}
}

type zonemasterProvider struct{}

func (p *zonemasterProvider) Key() sdk.ObservationKey {
	return ObservationKeyZonemaster
}

// Definition implements sdk.CheckerDefinitionProvider.
func (p *zonemasterProvider) Definition() *sdk.CheckerDefinition {
	return Definition()
}
