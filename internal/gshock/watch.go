package gshock

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"tinygo.org/x/bluetooth"
)

type Watch struct {
	Name            string
	Address         string
	AlwaysConnected bool

	device          bluetooth.Device
	profile         profile
	requestTimeout  time.Duration
	logger          *slog.Logger
	readRequest     bluetooth.DeviceCharacteristic
	allFeatures     bluetooth.DeviceCharacteristic
	spRequest       *bluetooth.DeviceCharacteristic
	spData          *bluetooth.DeviceCharacteristic
	notifications   chan []byte
	spNotifications chan []byte
}

func (w *Watch) enqueue(ch chan []byte, data []byte) {
	copyOfData := slices.Clone(data)
	select {
	case ch <- copyOfData:
	default:
		w.logger.Warn("dropping BLE notification", "watch", w.Name)
	}
}
func (w *Watch) PressedButton(ctx context.Context) (Button, error) {
	data, err := w.request(ctx, []byte{featureBLE}, featureBLE)
	if err != nil {
		return ButtonInvalid, fmt.Errorf("read pressed button: %w", err)
	}
	return decodeButton(data), nil
}
func (w *Watch) SetTime(ctx context.Context, now time.Time) error {
	if w.profile.protocol == mipProtocol {
		return w.setTimeMIP(ctx, now)
	}
	return w.setTimeStandard(ctx, now)
}
func (w *Watch) Disconnect() error {
	return w.device.Disconnect()
}
func (w *Watch) roundTrip(ctx context.Context, request []byte, key byte) error {
	response, err := w.request(ctx, request, key)
	if err != nil {
		return fmt.Errorf("request feature 0x%02x: %w", key, err)
	}
	if err := w.write(w.allFeatures, response, false); err != nil {
		return fmt.Errorf("write feature 0x%02x: %w", key, err)
	}
	return nil
}
func (w *Watch) request(ctx context.Context, request []byte, expectedKey byte) ([]byte, error) {
	drain(w.notifications)
	if err := w.write(w.readRequest, request, true); err != nil {
		return nil, err
	}

	waitCtx, cancel := context.WithTimeout(ctx, w.requestTimeout)
	defer cancel()
	for {
		select {
		case <-waitCtx.Done():
			return nil, waitCtx.Err()
		case data := <-w.notifications:
			key, err := responseKey(data, w.profile.protocol)
			if err != nil || key != expectedKey {
				continue
			}
			return slices.Clone(unwrapResponse(data, key, w.profile.protocol)), nil
		}
	}
}
func (w *Watch) write(characteristic bluetooth.DeviceCharacteristic, data []byte, withoutResponse bool) error {
	var err error
	if withoutResponse {
		_, err = characteristic.WriteWithoutResponse(data)
	} else {
		_, err = characteristic.Write(data)
	}
	return err
}
func (w *Watch) writeCurrentTime(data []byte) error {
	err := w.write(w.allFeatures, data, false)
	if err == nil {
		return nil
	}
	connected, stateErr := w.device.Connected()
	if stateErr == nil && !connected {
		w.logger.Debug("watch disconnected after receiving the time packet", "watch", w.Name)
		return nil
	}
	return err
}

func drain(ch chan []byte) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
