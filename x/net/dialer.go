// Package net provides instrumentation helpers for network connections.
package net

import (
	"context"
	stdnet "net"

	"github.com/upfluence/stats"
	statsio "github.com/upfluence/stats/x/io"
)

// Dialer establishes network connections.
type Dialer interface {
	Dial(network, address string) (stdnet.Conn, error)
	DialContext(ctx context.Context, network, address string) (stdnet.Conn, error)
}

// Config configures the metrics emitted by WrapDialer.
type Config struct {
	OpenOptions  []stats.InstrumentOption
	CloseOptions []stats.InstrumentOption
	ReadOptions  []stats.InstrumentOption
	WriteOptions []stats.InstrumentOption

	TrackReadBytes    bool
	TrackWrittenBytes bool
}

type dialer struct {
	d Dialer

	open stats.Instrument
	conn connectionMetrics
}

type connectionMetrics struct {
	scope        stats.Scope
	closeOptions []stats.InstrumentOption
	readConfig   statsio.Config
	writeConfig  statsio.Config
}

// WrapDialer instruments connections established by d.
func WrapDialer(d Dialer, scope stats.Scope, cfg Config) Dialer {
	return &dialer{
		d:    d,
		open: stats.NewInstrument(scope, "open", cfg.OpenOptions...),
		conn: connectionMetrics{
			scope:        scope,
			closeOptions: cfg.CloseOptions,
			readConfig: statsio.Config{
				InstrumentOptions: cfg.ReadOptions,
				TrackBytes:        cfg.TrackReadBytes,
			},
			writeConfig: statsio.Config{
				InstrumentOptions: cfg.WriteOptions,
				TrackBytes:        cfg.TrackWrittenBytes,
			},
		},
	}
}

func (d *dialer) Dial(network, address string) (stdnet.Conn, error) {
	return dial(d.open, d.conn, func() (stdnet.Conn, error) {
		return d.d.Dial(network, address)
	})
}

func (d *dialer) DialContext(ctx context.Context, network, address string) (stdnet.Conn, error) {
	return dial(d.open, d.conn, func() (stdnet.Conn, error) {
		return d.d.DialContext(ctx, network, address)
	})
}

func dial(instrument stats.Instrument, metrics connectionMetrics, fn func() (stdnet.Conn, error)) (stdnet.Conn, error) {
	// Preserve the dial error for callers and custom instrument formatters.
	//nolint:wrapcheck
	conn, err := stats.ExecInstrument2(instrument, fn)

	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return wrapConn(conn, metrics), nil
}
