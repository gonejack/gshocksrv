package gshock

import "testing"

func TestProfiles(t *testing.T) {
	tests := []struct {
		name       string
		protocol   protocol
		cities     int
		dst        int
		world      bool
		always     bool
		secondDial bool
	}{
		{"CASIO GW-B5600", standardProtocol, 6, 3, true, false, false},
		{"CASIO GW-BX5600", mipProtocol, 6, 3, true, false, false},
		{"CASIO MTG-B3000", analogueProtocol, 2, 1, false, false, true},
		{"CASIO GBD-H1000", standardProtocol, 2, 1, false, false, false},
		{"CASIO DW-H5600", standardProtocol, 2, 1, true, true, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := profileFor(test.name)
			if got.protocol != test.protocol || got.worldCities != test.cities ||
				got.dstStates != test.dst || got.hasWorldCities != test.world ||
				got.alwaysConnected != test.always || got.secondDial != test.secondDial {
				t.Fatalf("profileFor() = %+v", got)
			}
		})
	}
}
