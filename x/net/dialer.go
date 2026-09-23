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
	operation := d.open.Begin()
	conn, err := d.d.Dial(network, address)
	operation.Finish(err)

	return wrapDialedConn(conn, err, d.conn)
}

func (d *dialer) DialContext(ctx context.Context, network, address string) (stdnet.Conn, error) {
	operation := d.open.Begin()
	conn, err := d.d.DialContext(ctx, network, address)
	operation.Finish(err)

	return wrapDialedConn(conn, err, d.conn)
}

func wrapDialedConn(conn stdnet.Conn, err error, metrics connectionMetrics) (stdnet.Conn, error) {
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return wrapConn(conn, metrics), nil
}
