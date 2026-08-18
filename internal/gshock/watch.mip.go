package gshock

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"
)

func (w *Watch) setTimeMIP(ctx context.Context, now time.Time) error {
	if w.spRequest == nil || w.spData == nil {
		return errors.New("watch lacks MIP protocol characteristics")
	}

	step1Request := []byte{0x05, 0x1d, 0x00, 0x1d, 0x00, 0x24, 0x00, 0x24, 0x01, 0x24, 0x02}
	step1, err := w.requestSP(ctx, step1Request, 101)
	if err != nil {
		return fmt.Errorf("MIP step 1: %w", err)
	}
	step1[0] = 0x02
	if err := w.write(*w.spData, step1, false); err != nil {
		return fmt.Errorf("MIP step 1 write: %w", err)
	}

	step2Request := []byte{0x03}
	for range (w.profile.worldCities + 1) / 2 {
		step2Request = append(step2Request, featureDSTCity, 0)
	}
	step2, err := w.requestSP(ctx, step2Request, 28)
	if err != nil {
		return fmt.Errorf("MIP step 2: %w", err)
	}
	step2[0] = 0x06
	step2 = append(step2, worldCityRecords(now)...)
	if err := w.write(*w.spData, step2, false); err != nil {
		return fmt.Errorf("MIP step 2 write: %w", err)
	}

	step3Request := []byte{0x06}
	for i := range w.profile.worldCities {
		index := i / 2
		if i%2 != 0 {
			index += 6
		}
		step3Request = append(step3Request, featureWorld, byte(index))
	}
	step3, err := w.requestSP(ctx, step3Request, 1+w.profile.worldCities*22)
	if err != nil {
		return fmt.Errorf("MIP step 3: %w", err)
	}
	if err := w.write(*w.spData, step3, false); err != nil {
		return fmt.Errorf("MIP step 3 write: %w", err)
	}

	return w.writeCurrentTime(encodeMIPTime(now))
}
func (w *Watch) requestSP(ctx context.Context, request []byte, expected int) ([]byte, error) {
	drain(w.spNotifications)
	if err := w.write(*w.spRequest, request, true); err != nil {
		return nil, err
	}

	waitCtx, cancel := context.WithTimeout(ctx, w.requestTimeout)
	defer cancel()
	data := make([]byte, 0, expected)
	for len(data) < expected {
		select {
		case <-waitCtx.Done():
			return nil, waitCtx.Err()
		case part := <-w.spNotifications:
			data = append(data, part...)
		}
	}
	return data, nil
}

func encodeMIPTime(t time.Time) []byte {
	b := encodeTime(t)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	b[8] = byte(weekday)
	return b
}
func worldCityRecords(now time.Time) []byte {
	_, offset := now.Zone()
	longitude := math.Max(-180, math.Min(180, float64(offset)/3600*15))
	dst := byte(0)
	if now.IsDST() {
		dst = 1
	}
	data := make([]byte, 0, 66)
	data = append(data, cityRecord(0, 0, longitude, dst)...)
	data = append(data, cityRecord(1, 0, 0, 0)...)
	data = append(data, cityRecord(2, 0, 0, 0)...)
	return data
}
func cityRecord(slot byte, latitude, longitude float64, trailing byte) []byte {
	b := []byte{0x14, 0x00, 0x24, slot, 0x01}
	b = binary.BigEndian.AppendUint64(b, math.Float64bits(latitude))
	b = binary.BigEndian.AppendUint64(b, math.Float64bits(longitude))
	return append(b, trailing)
}
