package io

import (
	"io"

	"github.com/upfluence/stats"
)

type writer struct {
	io.Writer

	instrument stats.Instrument
	bytes      stats.Counter
}

// WrapWriter instruments calls to w. If w implements io.ReaderFrom, the
// returned writer implements it too.
func WrapWriter(w io.Writer, scope stats.Scope, cfg Config) io.Writer {
	var bytes = stats.NoopCounter

	if cfg.TrackBytes {
		bytes = scope.Counter("written_bytes_total")
	}

	wrapped := &writer{
		Writer:     w,
		instrument: stats.NewInstrument(scope, "write", cfg.InstrumentOptions...),
		bytes:      bytes,
	}

	if readerFrom, ok := w.(io.ReaderFrom); ok {
		return &readerFromWriter{writer: wrapped, readerFrom: readerFrom}
	}

	return wrapped
}

func (w *writer) Write(p []byte) (int, error) {
	operation := w.instrument.Begin()
	n, err := w.Writer.Write(p)
	operation.Finish(err)
	w.bytes.Add(int64(n))

	return n, err //nolint:wrapcheck
}

type readerFromWriter struct {
	*writer

	readerFrom io.ReaderFrom
}

func (w *readerFromWriter) ReadFrom(r io.Reader) (int64, error) {
	operation := w.instrument.Begin()
	n, err := w.readerFrom.ReadFrom(r)
	operation.Finish(err)
	w.bytes.Add(n)

	return n, err //nolint:wrapcheck
}
