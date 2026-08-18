package gshock

import (
	"context"
	"fmt"
	"time"
)

func (w *Watch) setTimeStandard(ctx context.Context, now time.Time) error {
	states := []byte{0, 2, 4}
	for _, state := range states[:w.profile.dstStates] {
		if err := w.roundTrip(ctx, []byte{featureDSTState, state}, featureDSTState); err != nil {
			return err
		}
	}
	for city := 0; city < w.profile.worldCities; city++ {
		if err := w.roundTrip(ctx, []byte{featureDSTCity, byte(city)}, featureDSTCity); err != nil {
			return err
		}
	}
	if w.profile.hasWorldCities {
		for city := 0; city < w.profile.worldCities; city++ {
			if err := w.roundTrip(ctx, []byte{featureWorld, byte(city)}, featureWorld); err != nil {
				return err
			}
		}
	} else if w.profile.hasHomeTime {
		for city := 0; city < w.profile.worldCities; city++ {
			if err := w.roundTrip(ctx, []byte{featureHomeTime, byte(city)}, featureHomeTime); err != nil {
				return err
			}
		}
	}

	if err := w.writeCurrentTime(encodeTime(now)); err != nil {
		return fmt.Errorf("write current time: %w", err)
	}
	if w.profile.secondDial {
		return w.setSecondDial(ctx)
	}
	return nil
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
		for city := 0; city < 2; city++ {
			if err := w.roundTrip(ctx, []byte{feature, byte(city)}, feature); err != nil {
				return err
			}
		}
	}
	return w.write(w.allFeatures, []byte{0x21, 0x01, 0x01}, false)
}
