package gshock

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cmp"
	"tinygo.org/x/bluetooth"
)

var ErrNotFound = errors.New("no matching G-Shock found")

const (
	casioServiceUUID  = "00001804-0000-1000-8000-00805f9b34fb"
	readRequestUUID   = "26eb002c-b012-49a8-b1f8-394fb2032b0f"
	allFeaturesUUID   = "26eb002d-b012-49a8-b1f8-394fb2032b0f"
	spRequestUUID     = "26eb002e-b012-49a8-b1f8-394fb2032b0f"
	spDataUUID        = "26eb002f-b012-49a8-b1f8-394fb2032b0f"
	notificationQueue = 64
)

type WatchConnector struct {
	adapt *bluetooth.Adapter

	requestTimeout time.Duration
	casioUUID      bluetooth.UUID

	g *slog.Logger
}

type candidate struct {
	address bluetooth.Address
	name    string
}

func (c *WatchConnector) ScanAndConnect(ctx context.Context, accept func(string) bool) (*Watch, error) {
	found, err := c.scan(ctx, accept)
	if err != nil {
		return nil, err
	}
	device, err := c.adapt.Connect(found.address, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", found.address.String(), err)
	}
	watch, err := c.prepareWatch(device, found.name, found.address.String())
	if err != nil {
		_ = device.Disconnect()
		return nil, err
	}
	return watch, nil
}

func (c *WatchConnector) scan(ctx context.Context, accept func(string) bool) (*candidate, error) {
	var found *candidate
	var err = c.scanWithContext(ctx, func(stop func()) error {
		return c.adapt.Scan(func(adapt *bluetooth.Adapter, res bluetooth.ScanResult) {
			if found != nil || !res.HasServiceUUID(c.casioUUID) {
				return
			}
			name := res.LocalName()
			if accept != nil && !accept(name) {
				return
			}
			found = &candidate{
				address: res.Address,
				name:    name,
			}
			stop()
		})
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("scan error: %w", err)
	case found == nil:
		return nil, ErrNotFound
	}
	return found, nil
}
func (c *WatchConnector) scanWithContext(ctx context.Context, scan func(func()) error) (err error) {
	defer func() { err = cmp.Or(ctx.Err(), err) }()
	scanErr := make(chan error, 1)
	stopReq := make(chan int, 1)
	go func() {
		scanErr <- scan(func() {
			select {
			case stopReq <- 0:
			default:
			}
		})
	}()
	select {
	case exx := <-scanErr:
		return exx
	case <-ctx.Done():
	case <-stopReq:
	}
	for {
		// If no scan is in progress, an error will be returned.
		if c.adapt.StopScan() != nil {
			select {
			case exx := <-scanErr: // read error if finished
				return exx
			case <-time.After(time.Second / 4): // wait start if not started
				continue
			}
		}
		return <-scanErr
	}
}
func (c *WatchConnector) prepareWatch(device bluetooth.Device, name, address string) (*Watch, error) {
	services, err := device.DiscoverServices(nil)
	if err != nil {
		return nil, fmt.Errorf("discover services: %w", err)
	}

	chars := make(map[string]bluetooth.DeviceCharacteristic)
	for _, service := range services {
		discovered, discoverErr := service.DiscoverCharacteristics(nil)
		if discoverErr != nil {
			c.g.Debug("characteristic discovery failed", "service", service.UUID().String(), "error", discoverErr)
			continue
		}
		for _, characteristic := range discovered {
			chars[strings.ToLower(characteristic.UUID().String())] = characteristic
		}
	}

	readRequest, ok := chars[readRequestUUID]
	if !ok {
		return nil, fmt.Errorf("watch lacks read-request characteristic %s", readRequestUUID)
	}
	allFeatures, ok := chars[allFeaturesUUID]
	if !ok {
		return nil, fmt.Errorf("watch lacks all-features characteristic %s", allFeaturesUUID)
	}

	p := profileFor(name)
	w := &Watch{
		Name:            name,
		Address:         address,
		AlwaysConnected: p.alwaysConnected,
		device:          device,
		profile:         p,
		requestTimeout:  c.requestTimeout,
		readRequest:     readRequest,
		allFeatures:     allFeatures,
		notifications:   make(chan []byte, notificationQueue),
		spNotifications: make(chan []byte, notificationQueue),

		g: c.g,
	}

	if characteristic, exists := chars[spRequestUUID]; exists {
		w.spRequest = &characteristic
	}

	notifications := 0
	for uuid, characteristic := range chars {
		switch uuid {
		case spDataUUID:
			w.spData = &characteristic
			if p.protocol == mipProtocol {
				err := characteristic.EnableNotifications(func(data []byte) { w.enqueue(w.spNotifications, data) })
				if err != nil {
					return nil, fmt.Errorf("enable SP notifications: %w", err)
				}
			}
		default:
			err := characteristic.EnableNotifications(func(data []byte) { w.enqueue(w.notifications, data) })
			if err == nil {
				notifications++
				c.g.Debug("subscribed to notifications", "uuid", uuid)
			} else {
				c.g.Debug("characteristic does not support notifications", "uuid", uuid, "error", err)
			}
		}
	}
	if notifications == 0 {
		return nil, errors.New("watch exposes no notifiable characteristic")
	}
	return w, nil
}

func NewWatchConnector(adapter *bluetooth.Adapter, g *slog.Logger, reqTimeout time.Duration) (*WatchConnector, error) {
	if reqTimeout <= 0 {
		return nil, errors.New("request timeout must be positive")
	}
	uuid, err := bluetooth.ParseUUID(casioServiceUUID)
	if err != nil {
		return nil, fmt.Errorf("parse Casio service UUID: %w", err)
	}
	if err := adapter.Enable(); err != nil {
		return nil, err
	}
	return &WatchConnector{
		adapt: adapter,

		requestTimeout: reqTimeout,
		casioUUID:      uuid,

		g: g,
	}, nil
}
