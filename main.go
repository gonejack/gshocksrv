package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/gonejack/gshocksrv/internal/gshock"
	"github.com/gonejack/gshocksrv/internal/server"
	"github.com/lmittmann/tint"
	"tinygo.org/x/bluetooth"
)

func main() {
	if err := new(application).run(); err != nil {
		fmt.Fprintln(os.Stderr, "gshocksrv:", err)
		os.Exit(1)
	}
}

type options struct {
	FineAdjustment int           `name:"fine-adjustment-secs" default:"0" help:"Seconds added to watch time (-10..10)."`
	ScanTimeout    time.Duration `name:"scan-timeout" default:"1m" help:"Maximum time for each BLE scan."`
	RequestTimeout time.Duration `name:"request-timeout" default:"5s" help:"Maximum time to wait for a watch response."`
	StorePath      string        `name:"store-path" default:"gshock_server_data.json" help:"Path to the state file."`
	LogLevel       string        `name:"log-level" default:"INFO" help:"Log level: DEBUG, INFO, WARN, or ERROR."`
	NoColor        bool          `name:"no-color" help:"Disable colored log output."`
}

type application struct {
	options
}

func (a *application) run() error {
	kong.Parse(&a.options,
		kong.Name("gshocksrv"),
		kong.Description("Synchronize Casio G-Shock watches over Bluetooth."),
		kong.UsageOnError(),
	)

	if a.FineAdjustment < -10 || a.FineAdjustment > 10 {
		return fmt.Errorf("--fine-adjustment-secs must be between -10 and 10")
	}
	level, err := a.parseLogLevel()
	if err != nil {
		return err
	}
	logger := slog.New(tint.NewTextHandler(os.Stdout, &tint.Options{
		Level:      level,
		TimeFormat: time.DateTime,
		NoColor:    a.NoColor,
	}))

	client, err := gshock.NewClient(bluetooth.DefaultAdapter, logger, a.RequestTimeout)
	if err != nil {
		return fmt.Errorf("initialize Bluetooth: %w", err)
	}

	cfg := server.Config{
		FineAdjustment: a.FineAdjustment,
		ScanTimeout:    a.ScanTimeout,
		RequestTimeout: a.RequestTimeout,
		StorePath:      a.StorePath,
	}
	srv := server.New(cfg, client, logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := srv.Run(ctx); err != nil {
		return fmt.Errorf("server stopped: %w", err)
	}
	return nil
}

func (a *application) parseLogLevel() (slog.Level, error) {
	switch strings.ToUpper(a.LogLevel) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN", "WARNING":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid --log-level %q", a.LogLevel)
	}
}
