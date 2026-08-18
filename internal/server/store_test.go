package server

import (
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	want := state{LastConnected: "08/18 12:34", WatchName: "CASIO GW-B5600"}
	if err := s.update(want); err != nil {
		t.Fatal(err)
	}
	loaded, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.data != want {
		t.Fatalf("loaded state = %+v, want %+v", loaded.data, want)
	}
}
