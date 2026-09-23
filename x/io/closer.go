package io

import (
	"io"

	"github.com/upfluence/stats"
)

type closer struct {
	io.Closer

	instrument stats.Instrument
}

// WrapCloser instruments calls to c.
func WrapCloser(c io.Closer, scope stats.Scope, opts ...stats.InstrumentOption) io.Closer {
	return &closer{
		Closer:     c,
		instrument: stats.NewInstrument(scope, "close", opts...),
	}
}

func (c *closer) Close() error {
	// Preserve the close error for callers and custom instrument formatters.
	//nolint:wrapcheck
	return c.instrument.Exec(c.Closer.Close)
}
