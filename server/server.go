package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gonejack/gshocksrv/gshock"
)

type Config struct {
	FineAdjustment int
	ScanTimeout    time.Duration
	RequestTimeout time.Duration
	StorePath      string
}

type Connector interface {
	ScanAndConnect(context.Context, func(string) bool) (*gshock.Watch, error)
}

type Server struct {
	cfg Config

	c Connector
	t *connectionLimiter

	g *slog.Logger
}

func (s *Server) Run(ctx context.Context) error {
	stateStore, err := openStore(s.cfg.StorePath)
	if err != nil {
		s.g.Warn("state file could not be loaded; starting empty", "error", err)
		stateStore = &store{path: s.cfg.StorePath}
	}

	s.g.Info("Long-press LOWER-LEFT or short-press LOWER-RIGHT on the watch to set time")
	s.g.Info("With automatic time adjustment enabled, the watch may connect up to four times per day")
	for {
		select {
		case <-ctx.Done():
			s.g.Info("Server stopped")
			return nil
		case <-time.After(time.Second):
			s.g.Info("Waiting for connection...")
			scanCtx, cancel := context.WithTimeout(ctx, s.cfg.ScanTimeout)
			watch, err := s.c.ScanAndConnect(scanCtx, s.acceptWatch)
			cancel()
			if err != nil {
				switch {
				case ctx.Err() != nil:
					s.g.Info("Server stopped")
					return nil
				case errors.Is(err, context.DeadlineExceeded), errors.Is(err, gshock.ErrNotFound):
					s.g.Debug("no matching watch found")
				default:
					s.g.Error("connection failed", "error", err)
				}
				continue
			}
			s.handleWatch(ctx, stateStore, watch)
		}
	}
}
func (s *Server) acceptWatch(name string) bool {
	return name != "CASIO OCW-T200" && s.t.allow(name)
}
func (s *Server) handleWatch(ctx context.Context, store *store, watch *gshock.Watch) {
	if !watch.AlwaysConnected {
		defer func() {
			if err := watch.Disconnect(); err != nil {
				s.g.Warn("disconnect failed", "watch", watch.Name, "error", err)
			}
		}()
	}
	s.g.Info("Connected", "watch", watch.Name, "address", watch.Address)
	err := store.update(state{
		LastConnected: time.Now().Format(time.DateTime),
		WatchName:     watch.Name,
	})
	if err != nil {
		s.g.Warn("state file could not be saved", "error", err)
	}
	btn, err := watch.PressedButton(ctx)
	switch {
	case err != nil:
		s.g.Error("button query failed", "watch", watch.Name, "error", err)
		return
	case btn == gshock.ButtonInvalid:
		s.g.Info("connection ignored: unsupported button", "watch", watch.Name)
		return
	}
	adjust := time.Duration(s.cfg.FineAdjustment) * time.Second
	s.g.Info("Set time start", "watch", watch.Name)
	t, err := watch.SetTime(ctx, adjust)
	if err != nil {
		s.g.Error("Set time failed", "watch", watch.Name, "error", err)
		return
	}
	s.g.Info("Set time done", "watch", watch.Name, "time", t.Format(time.RFC3339), "adjust", adjust)
}

func New(config Config, client Connector, logger *slog.Logger) *Server {
	return &Server{cfg: config, c: client, g: logger, t: newConnectionLimiter()}
}
