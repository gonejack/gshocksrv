package gshock

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

func TestEncodeTime(t *testing.T) {
	when := time.Date(2023, time.January, 2, 3, 4, 5, 500_000_000, time.UTC) // Monday
	want := []byte{0x09, 0xe7, 0x07, 1, 2, 3, 4, 5, 0, 128, 1}
	if got := encodeTime(when); !bytes.Equal(got, want) {
		t.Fatalf("encodeTime() = %x, want %x", got, want)
	}
}

func TestEncodeMIPTimeUsesCasioWeekday(t *testing.T) {
	sunday := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	if got := encodeMIPTime(sunday)[8]; got != 7 {
		t.Fatalf("Sunday = %d, want 7", got)
	}
}

func TestDecodeButton(t *testing.T) {
	for indicator, want := range map[byte]Button{
		0: ButtonLowerLeft,
		1: ButtonLowerLeft,
		3: ButtonNone,
		4: ButtonLowerRight,
		2: ButtonInvalid,
	} {
		data := make([]byte, 19)
		data[0], data[8] = featureBLE, indicator
		if got := decodeButton(data); got != want {
			t.Errorf("indicator %d = %d, want %d", indicator, got, want)
		}
	}
	if got := decodeButton([]byte{featureBLE}); got != ButtonInvalid {
		t.Errorf("short packet = %d, want invalid", got)
	}
}

func TestAnalogueResponseEnvelope(t *testing.T) {
	data := []byte{0x28, 0x01, 0xaa, 0xbb, featureDSTState, 0x02}
	key, err := responseKey(data, analogueProtocol)
	if err != nil || key != featureDSTState {
		t.Fatalf("responseKey() = 0x%02x, %v", key, err)
	}
	if got, want := unwrapResponse(data, key, analogueProtocol), data[4:]; !bytes.Equal(got, want) {
		t.Fatalf("unwrapResponse() = %x, want %x", got, want)
	}
}

func TestWorldCityRecords(t *testing.T) {
	zone := time.FixedZone("UTC+8", 8*60*60)
	records := worldCityRecords(time.Date(2026, 8, 18, 12, 0, 0, 0, zone))
	if len(records) != 66 {
		t.Fatalf("len(records) = %d, want 66", len(records))
	}
	longitude := math.Float64frombits(binary.BigEndian.Uint64(records[13:21]))
	if longitude != 120 {
		t.Fatalf("longitude = %v, want 120", longitude)
	}
}
