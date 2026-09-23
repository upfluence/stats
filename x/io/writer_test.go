package io

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upfluence/stats"
)

type plainWriter struct {
	w io.Writer
}

func (w *plainWriter) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

type benchmarkWriter struct{}

func (benchmarkWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func TestWrapWriter(t *testing.T) {
	collector := stats.NewStaticCollector()

	var dst bytes.Buffer

	w := WrapWriter(
		&plainWriter{w: &dst},
		stats.RootScope(collector),
		Config{
			InstrumentOptions: []stats.InstrumentOption{stats.DisableDurationTracking()},
			TrackBytes:        true,
		},
	)

	n, err := w.Write([]byte("content"))

	require.NoError(t, err)
	assert.Equal(t, 7, n)
	assert.Equal(t, "content", dst.String())
	assert.Equal(
		t,
		[]stats.Int64Snapshot{
			{Name: "write_started_total", Labels: map[string]string{}, Value: 1},
			{Name: "write_total", Labels: map[string]string{statusLabel: success}, Value: 1},
			{Name: writtenBytesMetric, Labels: map[string]string{}, Value: 7},
		},
		collector.Get().Counters,
	)
}

func TestWrapWriterReaderFrom(t *testing.T) {
	for _, tt := range []struct {
		name           string
		haveWriter     io.Writer
		wantReaderFrom bool
	}{
		{name: "plain writer", haveWriter: &plainWriter{w: &bytes.Buffer{}}},
		{name: "reader from writer", haveWriter: &bytes.Buffer{}, wantReaderFrom: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapWriter(tt.haveWriter, stats.NoopScope, Config{})
			_, hasReaderFrom := wrapped.(io.ReaderFrom)

			assert.Equal(t, tt.wantReaderFrom, hasReaderFrom)
		})
	}
}

func TestWrapWriterReadFrom(t *testing.T) {
	collector := stats.NewStaticCollector()

	var dst bytes.Buffer

	w := WrapWriter(
		&dst,
		stats.RootScope(collector),
		Config{
			InstrumentOptions: []stats.InstrumentOption{stats.DisableDurationTracking()},
			TrackBytes:        true,
		},
	)

	n, err := w.(io.ReaderFrom).ReadFrom(&plainReader{r: bytes.NewBufferString("content")})

	require.NoError(t, err)
	assert.Equal(t, int64(7), n)
	assert.Equal(t, "content", dst.String())
	assert.Equal(
		t,
		[]stats.Int64Snapshot{
			{Name: "write_started_total", Labels: map[string]string{}, Value: 1},
			{Name: "write_total", Labels: map[string]string{statusLabel: success}, Value: 1},
			{Name: writtenBytesMetric, Labels: map[string]string{}, Value: 7},
		},
		collector.Get().Counters,
	)
}

func TestWrapWriterWithoutByteTracking(t *testing.T) {
	collector := stats.NewStaticCollector()
	_, _ = WrapWriter(
		&bytes.Buffer{},
		stats.RootScope(collector),
		Config{},
	).Write([]byte("content"))

	assert.NotContains(t, collector.Get().Counters, stats.Int64Snapshot{Name: writtenBytesMetric})
}

func BenchmarkWriter(b *testing.B) {
	for _, bb := range []struct {
		name   string
		writer func() io.Writer
	}{
		{name: "direct", writer: func() io.Writer { return benchmarkWriter{} }},
		{name: "noop scope", writer: func() io.Writer { return WrapWriter(benchmarkWriter{}, stats.NoopScope, Config{}) }},
		{
			name: "default",
			writer: func() io.Writer {
				return WrapWriter(benchmarkWriter{}, stats.RootScope(stats.NewStaticCollector()), Config{})
			},
		},
		{
			name: "without duration",
			writer: func() io.Writer {
				return WrapWriter(
					benchmarkWriter{},
					stats.RootScope(stats.NewStaticCollector()),
					Config{InstrumentOptions: []stats.InstrumentOption{stats.DisableDurationTracking()}},
				)
			},
		},
		{
			name: "without duration with bytes",
			writer: func() io.Writer {
				return WrapWriter(
					benchmarkWriter{},
					stats.RootScope(stats.NewStaticCollector()),
					Config{
						InstrumentOptions: []stats.InstrumentOption{stats.DisableDurationTracking()},
						TrackBytes:        true,
					},
				)
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			w := bb.writer()
			buf := make([]byte, 32*1024)

			b.ReportAllocs()
			b.SetBytes(int64(len(buf)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = w.Write(buf)
			}
		})
	}
}
