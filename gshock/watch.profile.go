package gshock

import "strings"

type protocol uint8

const (
	standardProtocol protocol = iota
	analogueProtocol
	mipProtocol
)

type profile struct {
	protocol        protocol
	worldCities     int
	dstStates       int
	hasWorldCities  bool
	hasHomeTime     bool
	secondDial      bool
	alwaysConnected bool
}

func profileFor(name string) profile {
	model := strings.TrimSpace(strings.TrimPrefix(name, "CASIO "))
	p := profile{worldCities: 2, dstStates: 1, hasWorldCities: true}

	switch model {
	case "GW-BX5600":
		p.protocol, p.worldCities, p.dstStates = mipProtocol, 6, 3
	case "MTG-B1000":
		p.protocol, p.worldCities, p.dstStates, p.secondDial = analogueProtocol, 6, 3, true
	case "MTG-B3000", "MTG-B3100":
		p.protocol, p.hasWorldCities, p.hasHomeTime, p.secondDial = analogueProtocol, false, true, true
	}

	if hasSixWorldCities(model) {
		p.worldCities, p.dstStates = 6, 3
	}
	if hasNoWorldCities(model) {
		p.hasWorldCities = false
	}
	if isAlwaysConnected(model) {
		p.alwaysConnected = true
	}
	return p
}
func hasSixWorldCities(model string) bool {
	switch model {
	case "GW-B5000", "GW-B5600", "GW-B5600#", "GW-BX5600",
		"GMW-B5000", "GMW-B5000#", "GMW-BZ5000",
		"MRG-B5000", "MRG-B5000#", "GCW-B5000", "TRN-50", "PRJ-BW002",
		"DW-B5600", "GWR-B1000", "GWR-B3000", "GWG-B1000":
		return true
	}
	return false
}
func hasNoWorldCities(model string) bool {
	prefixes := []string{"ABL-", "GBD-", "GBX-", "GMD-B800", "EQB-"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}
func isAlwaysConnected(model string) bool {
	switch model {
	case "DW-H5600", "GBD-H2000", "DW-GH5600", "GM-H5600":
		return true
	default:
		return strings.HasPrefix(model, "ECB-")
	}
}
func IsAlwaysConnected(name string) bool {
	model := strings.TrimSpace(strings.TrimPrefix(name, "CASIO "))
	return isAlwaysConnected(model)
}
