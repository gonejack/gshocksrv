package gshock

import (
	"encoding/binary"
	"errors"
	"time"
)

const (
	featureBLE      byte = 0x10
	featureDSTState byte = 0x1d
	featureDSTCity  byte = 0x1e
	featureWorld    byte = 0x1f
	featureHomeTime byte = 0x24
	featureTime     byte = 0x09
)

type Button uint8

const (
	ButtonInvalid Button = iota
	ButtonLowerLeft
	ButtonLowerRight
	ButtonNone
)

func decodeButton(data []byte) Button {
	if len(data) < 19 || data[0] != featureBLE {
		return ButtonInvalid
	}
	switch data[8] {
	case 0, 1:
		return ButtonLowerLeft
	case 3:
		return ButtonNone
	case 4:
		return ButtonLowerRight
	default:
		return ButtonInvalid
	}
}
func encodeTime(t time.Time) []byte {
	b := make([]byte, 11)
	b[0] = featureTime
	binary.LittleEndian.PutUint16(b[1:3], uint16(t.Year()))
	b[3] = byte(t.Month())
	b[4] = byte(t.Day())
	b[5] = byte(t.Hour())
	b[6] = byte(t.Minute())
	b[7] = byte(t.Second())
	b[8] = byte((int(t.Weekday()) + 6) % 7) // Python datetime.weekday(): Monday = 0.
	b[9] = byte((t.Nanosecond() * 256) / int(time.Second))
	b[10] = 1
	return b
}
func responseKey(data []byte, p protocol) (byte, error) {
	if len(data) == 0 {
		return 0, errors.New("empty notification")
	}
	if p != analogueProtocol || data[0] != 0x28 {
		return data[0], nil
	}
	if len(data) > 4 && data[1] == 1 {
		return data[4], nil
	}
	if len(data) > 3 && data[1] == 0 {
		return data[3], nil
	}
	return data[0], nil
}
func unwrapResponse(data []byte, key byte, p protocol) []byte {
	if p != analogueProtocol || len(data) < 2 || data[0] != 0x28 || key == 0x28 {
		return data
	}
	if data[1] == 1 && len(data) >= 4 {
		return data[4:]
	}
	if len(data) >= 3 {
		return data[3:]
	}
	return data
}
