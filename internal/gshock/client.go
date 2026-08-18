package gshock

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"tinygo.org/x/bluetooth"
)

var ErrNotFound = errors.New("no matching G-Shock found")

const (
	casioServiceUUID  = "00001804-0000-1000-8000-00805f9b34fb"
	readRequestUUID   = "26eb002c-b012-49a8-b1f8-394fb2032b0f"
	allFeaturesUUID   = "26eb002d-b012-49a8-b1f8-394fb2032b0f"
	notificationUUID  = "26eb0030-b012-49a8-b1f8-394fb2032b0f"
	spRequestUUID     = "26eb002e-b012-49a8-b1f8-394fb2032b0f"
	spDataUUID        = "26eb002f-b012-49a8-b1f8-394fb2032b0f"
	notificationQueue = 64
)

type Client struct {
	adapter        *bluetooth.Adapter
	logger         *slog.Logger
	requestTimeout time.Duration
	serviceUUID    bluetooth.UUID
}

func (c *Client) ScanAndConnect(ctx context.Context, allow func(string) bool) (*Watch, error) {
	type candidate struct {
		address bluetooth.Address
		name    string
	}
	var found *candidate
	scanDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = c.adapter.StopScan()
		case <-scanDone:
		}
	}()

	err := c.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if found != nil || !result.HasServiceUUID(c.serviceUUID) {
			return
		}
		name := result.LocalName()
		if allow != nil && !allow(name) {
			return
		}
		found = &candidate{address: result.Address, name: name}
		_ = adapter.StopScan()
	})
	close(scanDone)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	if found == nil {
		return nil, ErrNotFound
	}

	device, err := c.adapter.Connect(found.address, bluetooth.ConnectionParams{})
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
func (c *Client) prepareWatch(device bluetooth.Device, name, address string) (*Watch, error) {
	services, err := device.DiscoverServices(nil)
	if err != nil {
		return nil, fmt.Errorf("discover services: %w", err)
	}

	chars := make(map[string]bluetooth.DeviceCharacteristic)
	for _, service := range services {
		discovered, discoverErr := service.DiscoverCharacteristics(nil)
		if discoverErr != nil {
			c.logger.Debug("characteristic discovery failed", "service", service.UUID().String(), "error", discoverErr)
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
	notify, ok := chars[notificationUUID]
	if !ok {
		return nil, fmt.Errorf("watch lacks notification characteristic %s", notificationUUID)
	}

	p := profileFor(name)
	w := &Watch{
		Name:            name,
		Address:         address,
		AlwaysConnected: p.alwaysConnected,
		device:          device,
		profile:         p,
		requestTimeout:  c.requestTimeout,
		logger:          c.logger,
		readRequest:     readRequest,
		allFeatures:     allFeatures,
		notifications:   make(chan []byte, notificationQueue),
		spNotifications: make(chan []byte, notificationQueue),
	}
	if err := notify.EnableNotifications(func(data []byte) {
		w.enqueue(w.notifications, data)
	}); err != nil {
		return nil, fmt.Errorf("enable watch notifications: %w", err)
	}

	if characteristic, exists := chars[spRequestUUID]; exists {
		w.spRequest = &characteristic
	}
	if characteristic, exists := chars[spDataUUID]; exists {
		w.spData = &characteristic
		if err := characteristic.EnableNotifications(func(data []byte) {
			w.enqueue(w.spNotifications, data)
		}); err != nil && p.protocol == mipProtocol {
			return nil, fmt.Errorf("enable SP notifications: %w", err)
		}
	}
	return w, nil
}

func NewClient(adapter *bluetooth.Adapter, logger *slog.Logger, requestTimeout time.Duration) (*Client, error) {
	if requestTimeout <= 0 {
		return nil, errors.New("request timeout must be positive")
	}
	serviceUUID, err := bluetooth.ParseUUID(casioServiceUUID)
	if err != nil {
		return nil, fmt.Errorf("parse Casio service UUID: %w", err)
	}
	if err := adapter.Enable(); err != nil {
		return nil, err
	}
	return &Client{adapter: adapter, logger: logger, requestTimeout: requestTimeout, serviceUUID: serviceUUID}, nil
}
