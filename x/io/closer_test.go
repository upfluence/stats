package io

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/upfluence/stats"
)

var errClose = errors.New("close")

type mockCloser struct {
	err   error
	calls int
}

func (c *mockCloser) Close() error {
	c.calls++

	return c.err
}

func TestWrapCloser(t *testing.T) {
	for _, tt := range []struct {
		name       string
		haveErr    error
		wantStatus string
	}{
		{
			name:       "success",
			wantStatus: success,
		},
		{
			name:       "error",
			haveErr:    errClose,
			wantStatus: failed,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			collector := stats.NewStaticCollector()
			rawCloser := &mockCloser{err: tt.haveErr}
			closer := WrapCloser(
				rawCloser,
				stats.RootScope(collector),
				stats.DisableDurationTracking(),
			)

			err := closer.Close()

			assert.Equal(t, tt.haveErr, err)
			assert.Equal(t, 1, rawCloser.calls)
			assert.Equal(
				t,
				[]stats.Int64Snapshot{
					{Name: "close_started_total", Labels: map[string]string{}, Value: 1},
					{Name: "close_total", Labels: map[string]string{statusLabel: tt.wantStatus}, Value: 1},
				},
				collector.Get().Counters,
			)
		})
	}
}
