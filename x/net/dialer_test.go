package net

import (
	"context"
	stdnet "net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upfluence/stats"
)

type mockDialer struct {
	conn stdnet.Conn

	dialCalls        int
	dialContextCalls int
}

func (d *mockDialer) Dial(string, string) (stdnet.Conn, error) {
	d.dialCalls++

	return d.conn, nil
}

func (d *mockDialer) DialContext(context.Context, string, string) (stdnet.Conn, error) {
	d.dialContextCalls++

	return d.conn, nil
}

func TestWrapDialer(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		haveDial             func(Dialer) (stdnet.Conn, error)
		wantDialCalls        int
		wantDialContextCalls int
	}{
		{
			name: "dial",
			haveDial: func(d Dialer) (stdnet.Conn, error) {
				return d.Dial("tcp", "localhost:80")
			},
			wantDialCalls: 1,
		},
		{
			name: "dial context",
			haveDial: func(d Dialer) (stdnet.Conn, error) {
				return d.DialContext(context.Background(), "tcp", "localhost:80")
			},
			wantDialContextCalls: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			collector := stats.NewStaticCollector()
			rawConn := &mockConn{}
			rawDialer := &mockDialer{conn: rawConn}
			dialer := WrapDialer(
				rawDialer,
				stats.RootScope(collector),
				Config{OpenOptions: []stats.InstrumentOption{stats.DisableDurationTracking()}},
			)

			conn, err := tt.haveDial(dialer)

			require.NoError(t, err)
			assert.NotEqual(t, rawConn, conn)
			assert.Equal(t, tt.wantDialCalls, rawDialer.dialCalls)
			assert.Equal(t, tt.wantDialContextCalls, rawDialer.dialContextCalls)
			assert.Equal(
				t,
				[]stats.Int64Snapshot{
					{Name: "close_started_total", Labels: map[string]string{}},
					{Name: "open_started_total", Labels: map[string]string{}, Value: 1},
					{Name: "open_total", Labels: map[string]string{statusLabel: success}, Value: 1},
					{Name: "read_started_total", Labels: map[string]string{}},
					{Name: "write_started_total", Labels: map[string]string{}},
				},
				collector.Get().Counters,
			)
		})
	}
}
