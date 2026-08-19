package gshock

import (
	"context"
	"fmt"
	"time"
)

func (w *Watch) setTimeStandard(ctx context.Context, adjustment time.Duration) (time.Time, error) {
	states := []byte{0, 2, 4}
	for _, state := range states[:w.profile.dstStates] {
		if err := w.roundTrip(ctx, []byte{featureDSTState, state}, featureDSTState); err != nil {
			return time.Time{}, err
		}
	}
	for city := range w.profile.worldCities {
		if err := w.roundTrip(ctx, []byte{featureDSTCity, byte(city)}, featureDSTCity); err != nil {
			return time.Time{}, err
		}
	}
	if w.profile.hasWorldCities {
		for city := range w.profile.worldCities {
			if err := w.roundTrip(ctx, []byte{featureWorld, byte(city)}, featureWorld); err != nil {
				return time.Time{}, err
			}
		}
	} else if w.profile.hasHomeTime {
		for city := range w.profile.worldCities {
			if err := w.roundTrip(ctx, []byte{featureHomeTime, byte(city)}, featureHomeTime); err != nil {
				return time.Time{}, err
			}
		}
	}
	wrt := time.Now().Add(adjustment)
	if err := w.writeTime(encodeTime(wrt)); err != nil {
		return wrt, fmt.Errorf("write current time: %w", err)
	}
	if w.profile.secondDial {
		if err := w.setSecondDial(ctx); err != nil {
			return wrt, err
		}
	}
	return wrt, nil
}
func (w *Watch) setSecondDial(ctx context.Context) error {
	if err := w.write(w.allFeatures, []byte{0x21, 0x00, 0x01}, false); err != nil {
		return err
	}
	if err := w.roundTrip(ctx, []byte{featureDSTState, 0}, featureDSTState); err != nil {
		return err
	}
	for _, feature := range []byte{featureDSTCity, featureWorld} {
		if feature == featureWorld && !w.profile.hasWorldCities {
			continue
		}
		for city := range 2 {
			if err := w.roundTrip(ctx, []byte{feature, byte(city)}, feature); err != nil {
				return err
			}
		}
	}
	return w.write(w.allFeatures, []byte{0x21, 0x01, 0x01}, false)
}
