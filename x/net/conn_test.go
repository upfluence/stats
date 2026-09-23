package net

import (
	"bytes"
	"io"
	stdnet "net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upfluence/stats"
	statsio "github.com/upfluence/stats/x/io"
)

const (
	statusLabel = "status"
	success     = "success"
)

type mockConn struct {
	readData  []byte
	written   bytes.Buffer
	closeErr  error
	readErr   error
	writeErr  error
	closeCall int
}

func (c *mockConn) Close() error {
	c.closeCall++

	return c.closeErr
}

func (c *mockConn) Read(p []byte) (int, error) {
	n := copy(p, c.readData)
	c.readData = c.readData[n:]

	return n, c.readErr
}

func (c *mockConn) Write(p []byte) (int, error) {
	n, _ := c.written.Write(p)

	return n, c.writeErr
}

func (c *mockConn) LocalAddr() stdnet.Addr           { return mockAddr("local") }
func (c *mockConn) RemoteAddr() stdnet.Addr          { return mockAddr("remote") }
func (c *mockConn) SetDeadline(time.Time) error      { return nil }
func (c *mockConn) SetReadDeadline(time.Time) error  { return nil }
func (c *mockConn) SetWriteDeadline(time.Time) error { return nil }

type mockAddr string

func (a mockAddr) Network() string { return string(a) }
func (a mockAddr) String() string  { return string(a) }

type readerFromMockConn struct {
	*mockConn
}

func (c *readerFromMockConn) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(&c.written, r)
}

type writerToMockConn struct {
	*mockConn
}

func (c *writerToMockConn) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(c.readData)
	c.readData = c.readData[n:]

	return int64(n), err
}

type readerFromWriterToMockConn struct {
	*mockConn
}

func (c *readerFromWriterToMockConn) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(&c.written, r)
}

func (c *readerFromWriterToMockConn) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(c.readData)
	c.readData = c.readData[n:]

	return int64(n), err
}

func TestWrapConnOptionalInterfaces(t *testing.T) {
	for _, tt := range []struct {
		name           string
		haveConn       stdnet.Conn
		wantReaderFrom bool
		wantWriterTo   bool
	}{
		{
			name:     "plain connection",
			haveConn: &mockConn{},
		},
		{
			name:           "reader from connection",
			haveConn:       &readerFromMockConn{mockConn: &mockConn{}},
			wantReaderFrom: true,
		},
		{
			name:         "writer to connection",
			haveConn:     &writerToMockConn{mockConn: &mockConn{}},
			wantWriterTo: true,
		},
		{
			name: "reader from and writer to connection",
			haveConn: &readerFromWriterToMockConn{
				mockConn: &mockConn{},
			},
			wantReaderFrom: true,
			wantWriterTo:   true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conn := wrapConn(
				tt.haveConn,
				connectionMetrics{
					scope: stats.NoopScope,
				},
			)
			_, hasReaderFrom := conn.(io.ReaderFrom)
			_, hasWriterTo := conn.(io.WriterTo)

			assert.Equal(t, tt.wantReaderFrom, hasReaderFrom)
			assert.Equal(t, tt.wantWriterTo, hasWriterTo)
		})
	}
}

func TestConnMetrics(t *testing.T) {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)
	rawConn := &readerFromWriterToMockConn{
		mockConn: &mockConn{readData: []byte("read")},
	}
	conn := wrapConn(
		rawConn,
		connectionMetrics{
			scope:        scope,
			closeOptions: []stats.InstrumentOption{stats.DisableDurationTracking()},
			readConfig: statsio.Config{
				InstrumentOptions: []stats.InstrumentOption{stats.DisableDurationTracking()},
				TrackBytes:        true,
			},
			writeConfig: statsio.Config{
				InstrumentOptions: []stats.InstrumentOption{stats.DisableDurationTracking()},
				TrackBytes:        true,
			},
		},
	)

	buf := make([]byte, 2)
	n, err := conn.Read(buf)

	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte("re"), buf)

	n, err = conn.Write([]byte("write"))

	require.NoError(t, err)
	assert.Equal(t, 5, n)

	readerFrom := conn.(io.ReaderFrom)
	n64, err := readerFrom.ReadFrom(bytes.NewBufferString("from"))

	require.NoError(t, err)
	assert.Equal(t, int64(4), n64)

	var dst bytes.Buffer

	writerTo := conn.(io.WriterTo)
	n64, err = writerTo.WriteTo(&dst)

	require.NoError(t, err)
	assert.Equal(t, int64(2), n64)
	assert.Equal(t, "ad", dst.String())

	err = conn.Close()

	require.NoError(t, err)
	assert.Equal(t, 1, rawConn.closeCall)
	assert.Equal(
		t,
		[]stats.Int64Snapshot{
			{Name: "close_started_total", Labels: map[string]string{}, Value: 1},
			{Name: "close_total", Labels: map[string]string{statusLabel: success}, Value: 1},
			{Name: "read_bytes_total", Labels: map[string]string{}, Value: 4},
			{Name: "read_started_total", Labels: map[string]string{}, Value: 2},
			{Name: "read_total", Labels: map[string]string{statusLabel: success}, Value: 2},
			{Name: "write_started_total", Labels: map[string]string{}, Value: 2},
			{Name: "write_total", Labels: map[string]string{statusLabel: success}, Value: 2},
			{Name: "written_bytes_total", Labels: map[string]string{}, Value: 9},
		},
		collector.Get().Counters,
	)
}

func BenchmarkConnRead(b *testing.B) {
	for _, bb := range []struct {
		name string
		conn func() interface{ Read([]byte) (int, error) }
	}{
		{
			name: "direct",
			conn: func() interface{ Read([]byte) (int, error) } {
				return &mockConn{}
			},
		},
		{
			name: "noop scope",
			conn: func() interface{ Read([]byte) (int, error) } {
				return wrapConn(
					&mockConn{},
					connectionMetrics{scope: stats.NoopScope},
				)
			},
		},
		{
			name: "default",
			conn: func() interface{ Read([]byte) (int, error) } {
				scope := stats.RootScope(stats.NewStaticCollector())

				return wrapConn(
					&mockConn{},
					connectionMetrics{scope: scope},
				)
			},
		},
		{
			name: "without duration",
			conn: func() interface{ Read([]byte) (int, error) } {
				scope := stats.RootScope(stats.NewStaticCollector())

				return wrapConn(
					&mockConn{},
					connectionMetrics{
						scope: scope,
						readConfig: statsio.Config{
							InstrumentOptions: []stats.InstrumentOption{stats.DisableDurationTracking()},
						},
					},
				)
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			conn := bb.conn()
			buf := make([]byte, 32*1024)

			b.ReportAllocs()
			b.SetBytes(int64(len(buf)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = conn.Read(buf)
			}
		})
	}
}
