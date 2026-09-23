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
	n, err := exec(w.instrument, func() (int, error) {
		return w.Writer.Write(p)
	})
	w.bytes.Add(int64(n))

	return n, err
}

type readerFromWriter struct {
	*writer

	readerFrom io.ReaderFrom
}

func (w *readerFromWriter) ReadFrom(r io.Reader) (int64, error) {
	n, err := exec(w.instrument, func() (int64, error) {
		return w.readerFrom.ReadFrom(r)
	})
	w.bytes.Add(n)

	return n, err
}
