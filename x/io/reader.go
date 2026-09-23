// Package io provides instrumentation helpers for I/O operations.
package io

import (
	"io"

	"github.com/upfluence/stats"
)

// Config configures the metrics emitted by WrapReader and WrapWriter.
type Config struct {
	InstrumentOptions []stats.InstrumentOption
	TrackBytes        bool
}

type reader struct {
	io.Reader

	instrument stats.Instrument
	bytes      stats.Counter
}

// WrapReader instruments calls to r. If r implements io.WriterTo, the returned
// reader implements it too.
func WrapReader(r io.Reader, scope stats.Scope, cfg Config) io.Reader {
	var bytes = stats.NoopCounter

	if cfg.TrackBytes {
		bytes = scope.Counter("read_bytes_total")
	}

	wrapped := &reader{
		Reader:     r,
		instrument: stats.NewInstrument(scope, "read", cfg.InstrumentOptions...),
		bytes:      bytes,
	}

	if writerTo, ok := r.(io.WriterTo); ok {
		return &writerToReader{reader: wrapped, writerTo: writerTo}
	}

	return wrapped
}

func (r *reader) Read(p []byte) (int, error) {
	n, err := exec(r.instrument, func() (int, error) {
		return r.Reader.Read(p)
	})
	r.bytes.Add(int64(n))

	return n, err
}

type writerToReader struct {
	*reader

	writerTo io.WriterTo
}

func (r *writerToReader) WriteTo(w io.Writer) (int64, error) {
	n, err := exec(r.instrument, func() (int64, error) {
		return r.writerTo.WriteTo(w)
	})
	r.bytes.Add(n)

	return n, err
}

func exec[T int | int64](instrument stats.Instrument, fn func() (T, error)) (T, error) {
	// Preserve I/O errors for callers and custom instrument formatters.
	//nolint:wrapcheck
	return stats.ExecInstrument2(instrument, fn)
}
