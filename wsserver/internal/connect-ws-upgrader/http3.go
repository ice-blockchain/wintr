// SPDX-License-Identifier: ice License 1.0

package connectwsupgrader

import (
	"net"
	stdlibtime "time"
)

func (h *http3StreamProxy) Read(b []byte) (n int, err error) {
	return h.stream.Read(b) //nolint:wrapcheck // Proxy.
}

func (h *http3StreamProxy) Write(b []byte) (n int, err error) {
	return h.stream.Write(b) //nolint:wrapcheck // Proxy.
}

func (h *http3StreamProxy) Close() error {
	return h.stream.Close() //nolint:wrapcheck // Proxy.
}

func (h *http3StreamProxy) LocalAddr() net.Addr {
	return h.streamCreator.LocalAddr()
}

func (h *http3StreamProxy) RemoteAddr() net.Addr {
	return h.streamCreator.RemoteAddr()
}

func (h *http3StreamProxy) SetDeadline(t stdlibtime.Time) error {
	return h.stream.SetDeadline(t) //nolint:wrapcheck // Proxy.
}

func (h *http3StreamProxy) SetReadDeadline(t stdlibtime.Time) error {
	return h.stream.SetReadDeadline(t) //nolint:wrapcheck // Proxy.
}

func (h *http3StreamProxy) SetWriteDeadline(t stdlibtime.Time) error {
	return h.stream.SetWriteDeadline(t) //nolint:wrapcheck // Proxy.
}
