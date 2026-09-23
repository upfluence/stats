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
	operation := r.instrument.Begin()
	n, err := r.Reader.Read(p)
	operation.Finish(err)
	r.bytes.Add(int64(n))

	return n, err //nolint:wrapcheck
}

type writerToReader struct {
	*reader

	writerTo io.WriterTo
}

func (r *writerToReader) WriteTo(w io.Writer) (int64, error) {
	operation := r.instrument.Begin()
	n, err := r.writerTo.WriteTo(w)
	operation.Finish(err)
	r.bytes.Add(n)

	return n, err //nolint:wrapcheck
}
