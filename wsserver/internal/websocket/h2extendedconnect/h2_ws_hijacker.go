// SPDX-License-Identifier: ice License 1.0

package http

import (
	"bufio"
	"net"
	stdlibtime "time"

	"github.com/hashicorp/go-multierror"
)

func (rw *http2responseWriter) Read(b []byte) (n int, err error) {
	return rw.rws.stream.body.Read(b)
}

func (rw *http2responseWriter) Close() error {
	if rw.rws.stream.state != http2stateClosed {
		rw.handlerDone()
	}

	return nil
}

func (rw *http2responseWriter) LocalAddr() net.Addr {
	return rw.rws.conn.conn.LocalAddr()
}

func (rw *http2responseWriter) RemoteAddr() net.Addr {
	return rw.rws.conn.conn.RemoteAddr()
}

func (rw *http2responseWriter) SetDeadline(t stdlibtime.Time) error {
	return multierror.Append(
		rw.SetReadDeadline(t),
		rw.SetWriteDeadline(t),
	).ErrorOrNil()
}

func (rw *http2responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	brw := bufio.NewReadWriter(bufio.NewReader(rw.rws.stream.body), rw.rws.bw)
	return rw, brw, nil
}
