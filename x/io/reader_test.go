package io

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upfluence/stats"
)

const (
	readBytesMetric    = "read_bytes_total"
	failed             = "failed"
	statusLabel        = "status"
	success            = "success"
	writtenBytesMetric = "written_bytes_total"
)

type plainReader struct {
	r io.Reader
}

func (r *plainReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

type benchmarkReader struct{}

func (benchmarkReader) Read(p []byte) (int, error) {
	return len(p), nil
}

func TestWrapReader(t *testing.T) {
	collector := stats.NewStaticCollector()
	r := WrapReader(
		&plainReader{r: bytes.NewBufferString("content")},
		stats.RootScope(collector),
		Config{
			InstrumentOptions: []stats.InstrumentOption{stats.DisableDurationTracking()},
			TrackBytes:        true,
		},
	)
	buf := make([]byte, 4)

	n, err := r.Read(buf)

	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, []byte("cont"), buf)
	assert.Equal(
		t,
		[]stats.Int64Snapshot{
			{Name: readBytesMetric, Labels: map[string]string{}, Value: 4},
			{Name: "read_started_total", Labels: map[string]string{}, Value: 1},
			{Name: "read_total", Labels: map[string]string{statusLabel: success}, Value: 1},
		},
		collector.Get().Counters,
	)
}

func TestWrapReaderWriterTo(t *testing.T) {
	for _, tt := range []struct {
		name         string
		haveReader   io.Reader
		wantWriterTo bool
	}{
		{name: "plain reader", haveReader: &plainReader{r: bytes.NewBufferString("content")}},
		{name: "writer to reader", haveReader: bytes.NewBufferString("content"), wantWriterTo: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapReader(tt.haveReader, stats.NoopScope, Config{})
			_, hasWriterTo := wrapped.(io.WriterTo)

			assert.Equal(t, tt.wantWriterTo, hasWriterTo)
		})
	}
}

func TestWrapReaderWriteTo(t *testing.T) {
	collector := stats.NewStaticCollector()
	r := WrapReader(
		bytes.NewBufferString("content"),
		stats.RootScope(collector),
		Config{
			InstrumentOptions: []stats.InstrumentOption{stats.DisableDurationTracking()},
			TrackBytes:        true,
		},
	)

	var dst bytes.Buffer

	n, err := r.(io.WriterTo).WriteTo(&dst)

	require.NoError(t, err)
	assert.Equal(t, int64(7), n)
	assert.Equal(t, "content", dst.String())
	assert.Equal(
		t,
		[]stats.Int64Snapshot{
			{Name: readBytesMetric, Labels: map[string]string{}, Value: 7},
			{Name: "read_started_total", Labels: map[string]string{}, Value: 1},
			{Name: "read_total", Labels: map[string]string{statusLabel: success}, Value: 1},
		},
		collector.Get().Counters,
	)
}

func TestWrapReaderWithoutByteTracking(t *testing.T) {
	collector := stats.NewStaticCollector()
	_, _ = WrapReader(
		bytes.NewBufferString("content"),
		stats.RootScope(collector),
		Config{},
	).Read(make([]byte, 7))

	assert.NotContains(t, collector.Get().Counters, stats.Int64Snapshot{Name: readBytesMetric})
}

func BenchmarkReader(b *testing.B) {
	for _, bb := range []struct {
		name   string
		reader func() io.Reader
	}{
		{name: "direct", reader: func() io.Reader { return benchmarkReader{} }},
		{name: "noop scope", reader: func() io.Reader { return WrapReader(benchmarkReader{}, stats.NoopScope, Config{}) }},
		{
			name: "default",
			reader: func() io.Reader {
				return WrapReader(benchmarkReader{}, stats.RootScope(stats.NewStaticCollector()), Config{})
			},
		},
		{
			name: "without duration",
			reader: func() io.Reader {
				return WrapReader(
					benchmarkReader{},
					stats.RootScope(stats.NewStaticCollector()),
					Config{InstrumentOptions: []stats.InstrumentOption{stats.DisableDurationTracking()}},
				)
			},
		},
		{
			name: "without duration with bytes",
			reader: func() io.Reader {
				return WrapReader(
					benchmarkReader{},
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
			r := bb.reader()
			buf := make([]byte, 32*1024)

			b.ReportAllocs()
			b.SetBytes(int64(len(buf)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = r.Read(buf)
			}
		})
	}
}
