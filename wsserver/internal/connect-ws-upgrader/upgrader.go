// SPDX-License-Identifier: ice License 1.0

package connectwsupgrader

import (
	"bufio"
	"net"
	"net/http"

	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go/http3"

	"github.com/ice-blockchain/wintr/log"
)

//nolint:funlen,gocritic,revive // Nope, we're keeping it compatible with 3rd party
func (u *ConnectUpgrader) Upgrade(req *http.Request, writer http.ResponseWriter) (conn net.Conn, rw *bufio.ReadWriter, hs ws.Handshake, err error) {
	if req.Proto != "websocket" {
		writer.WriteHeader(http.StatusBadRequest)

		return nil, nil, hs, ErrBadProtocol
	}
	switch w := writer.(type) {
	case http.Hijacker:
		conn, rw, err = w.Hijack()
	case http3.Hijacker:
		httpStreamer := req.Body.(http3.HTTPStreamer) //nolint:errcheck,forcetypeassert // Should be fine unless quick change API.
		conn = &http3StreamProxy{stream: httpStreamer.HTTPStream(), streamCreator: w.StreamCreator()}
	default:
		err = errors.New("http.ResponseWriter does not support hijack")
		log.Error(err)
		writer.WriteHeader(http.StatusInternalServerError)
	}

	if err != nil {
		return nil, nil, hs, errors.Wrapf(err, "failed to hijack http2")
	}
	hs, err = u.syncWSProtocols(req)
	if err != nil {
		return nil, nil, hs, errors.Wrapf(err, "failed to sync ws protocol and extensions")
	}

	writer.Header().Add(headerSecProtocolCanonical, hs.Protocol)
	writer.Header().Add(headerSecVersionCanonical, "13")
	writer.WriteHeader(http.StatusOK)
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return conn, rw, hs, errors.New("websocket: response writer must implement flusher")
	}
	flusher.Flush()

	return conn, rw, hs, err
}

func strSelectProtocol(h string, check func(string) bool) (ret string, ok bool) {
	ok = httphead.ScanTokens([]byte(h), func(v []byte) bool {
		if check(string(v)) {
			ret = string(v)

			return false
		}

		return true
	})

	return ret, ok
}

func btsSelectExtensions(header []byte, selected []httphead.Option, check func(httphead.Option) bool) ([]httphead.Option, bool) {
	s := httphead.OptionSelector{
		Flags: httphead.SelectCopy,
		Check: check,
	}

	return s.Select(header, selected)
}

//nolint:gocritic // Nope, we need to keep it compatible with 3rd party.
func negotiateMaybe(in httphead.Option, dest []httphead.Option, f func(httphead.Option) (httphead.Option, error)) ([]httphead.Option, error) {
	if in.Size() == 0 {
		return dest, nil
	}
	opt, err := f(in)
	if err != nil {
		return nil, err
	}
	if opt.Size() > 0 {
		dest = append(dest, opt) //nolint:revive // .
	}

	return dest, nil
}

func negotiateExtensions(
	h []byte, dest []httphead.Option,
	extensionsFunc func(httphead.Option) (httphead.Option, error),
) (_ []httphead.Option, err error) {
	index := -1
	var current httphead.Option
	ok := httphead.ScanOptions(h, func(idx int, name, attr, val []byte) httphead.Control {
		if idx != index {
			dest, err = negotiateMaybe(current, dest, extensionsFunc) //nolint:revive // .
			if err != nil {
				return httphead.ControlBreak
			}
			index = idx
			current = httphead.Option{Name: name}
		}
		if attr != nil {
			current.Parameters.Set(attr, val)
		}

		return httphead.ControlContinue
	})
	if !ok {
		return nil, ws.ErrMalformedRequest
	}

	return negotiateMaybe(current, dest, extensionsFunc)
}

//nolint:gocognit,gocyclo,revive,cyclop // .
func (u *ConnectUpgrader) syncWSProtocols(req *http.Request) (hs ws.Handshake, err error) {
	if check := u.Protocol; check != nil {
		ps := req.Header[headerSecProtocolCanonical]
		for i := 0; hs.Protocol == "" && err == nil && i < len(ps); i++ {
			var ok bool
			hs.Protocol, ok = strSelectProtocol(ps[i], check)
			if !ok {
				err = ws.ErrMalformedRequest
			}
		}
	}
	if f := u.Negotiate; err == nil && f != nil {
		for _, h := range req.Header[headerSecExtensionsCanonical] {
			hs.Extensions, err = negotiateExtensions([]byte(h), hs.Extensions, f)
			if err != nil {
				break
			}
		}
	}
	if check := u.Extension; err == nil && check != nil && u.Negotiate == nil {
		xs := req.Header[headerSecExtensionsCanonical]
		for i := 0; err == nil && i < len(xs); i++ {
			var ok bool
			hs.Extensions, ok = btsSelectExtensions([]byte(xs[i]), hs.Extensions, check)
			if !ok {
				err = ws.ErrMalformedRequest
			}
		}
	}

	return hs, err
}
