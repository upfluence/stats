package net

import (
	"io"
	stdnet "net"

	statsio "github.com/upfluence/stats/x/io"
)

type conn struct {
	stdnet.Conn

	closer io.Closer
	reader io.Reader
	writer io.Writer
}

func (c *conn) Close() error {
	return c.closer.Close() //nolint:wrapcheck
}

func (c *conn) Read(p []byte) (int, error) {
	return c.reader.Read(p) //nolint:wrapcheck
}

func (c *conn) Write(p []byte) (int, error) {
	return c.writer.Write(p) //nolint:wrapcheck
}

type readerFromConn struct {
	*conn

	readerFrom io.ReaderFrom
}

func (c *readerFromConn) ReadFrom(r io.Reader) (int64, error) {
	return c.readerFrom.ReadFrom(r) //nolint:wrapcheck
}

type writerToConn struct {
	*conn

	writerTo io.WriterTo
}

func (c *writerToConn) WriteTo(w io.Writer) (int64, error) {
	return c.writerTo.WriteTo(w) //nolint:wrapcheck
}

type readerFromWriterToConn struct {
	*conn

	readerFrom io.ReaderFrom
	writerTo   io.WriterTo
}

func (c *readerFromWriterToConn) ReadFrom(r io.Reader) (int64, error) {
	return c.readerFrom.ReadFrom(r) //nolint:wrapcheck
}

func (c *readerFromWriterToConn) WriteTo(w io.Writer) (int64, error) {
	return c.writerTo.WriteTo(w) //nolint:wrapcheck
}

func wrapConn(rawConn stdnet.Conn, metrics connectionMetrics) stdnet.Conn {
	var c = &conn{
		Conn:   rawConn,
		closer: statsio.WrapCloser(rawConn, metrics.scope, metrics.closeOptions...),
		reader: statsio.WrapReader(rawConn, metrics.scope, metrics.readConfig),
		writer: statsio.WrapWriter(rawConn, metrics.scope, metrics.writeConfig),
	}

	readerFrom, hasReaderFrom := c.writer.(io.ReaderFrom)
	writerTo, hasWriterTo := c.reader.(io.WriterTo)

	if hasReaderFrom && hasWriterTo {
		return &readerFromWriterToConn{
			conn:       c,
			readerFrom: readerFrom,
			writerTo:   writerTo,
		}
	}

	if hasReaderFrom {
		return &readerFromConn{conn: c, readerFrom: readerFrom}
	}

	if hasWriterTo {
		return &writerToConn{conn: c, writerTo: writerTo}
	}

	return c
}
