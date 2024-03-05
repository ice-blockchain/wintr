package h2upgrader

import (
	"bufio"
	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/ice-blockchain/wintr/log"
	"github.com/pkg/errors"
	"net"
	"net/http"
)

func (u *H2Upgrader) Upgrade(r *http.Request, w http.ResponseWriter) (conn net.Conn, rw *bufio.ReadWriter, hs ws.Handshake, err error) {
	// todo extra rfc5881 checks like connect call, protocol header, etc
	if hj, ok := w.(http.Hijacker); ok {
		conn, rw, err = hj.Hijack()
		if err != nil {
			return nil, nil, hs, errors.Wrapf(err, "failed to hijack http2")
		}
	} else {
		log.Error(errors.New("http.ResponseWriter does not support hijack"))
		w.WriteHeader(400)
		return
	}
	if check := u.Protocol; err == nil && check != nil {
		ps := r.Header[headerSecProtocolCanonical]
		for i := 0; i < len(ps) && err == nil && hs.Protocol == ""; i++ {
			var ok bool
			hs.Protocol, ok = strSelectProtocol(ps[i], check)
			if !ok {
				err = ws.ErrMalformedRequest
			}
		}
	}

	if f := u.Negotiate; err == nil && f != nil {
		for _, h := range r.Header[headerSecExtensionsCanonical] {
			hs.Extensions, err = negotiateExtensions([]byte(h), hs.Extensions, f)
			if err != nil {
				break
			}
		}
	}
	if check := u.Extension; err == nil && check != nil && u.Negotiate == nil {
		xs := r.Header[headerSecExtensionsCanonical]
		for i := 0; i < len(xs) && err == nil; i++ {
			var ok bool
			hs.Extensions, ok = btsSelectExtensions([]byte(xs[i]), hs.Extensions, check)
			if !ok {
				err = ws.ErrMalformedRequest
			}
		}
	}

	w.Header().Add(headerSecProtocolCanonical, hs.Protocol)
	w.Header().Add(headerSecVersionCanonical, "13")
	w.WriteHeader(200)
	flusher, ok := w.(http.Flusher)
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

func btsSelectExtensions(h []byte, selected []httphead.Option, check func(httphead.Option) bool) ([]httphead.Option, bool) {
	s := httphead.OptionSelector{
		Flags: httphead.SelectCopy,
		Check: check,
	}
	return s.Select(h, selected)
}

func negotiateMaybe(in httphead.Option, dest []httphead.Option, f func(httphead.Option) (httphead.Option, error)) ([]httphead.Option, error) {
	if in.Size() == 0 {
		return dest, nil
	}
	opt, err := f(in)
	if err != nil {
		return nil, err
	}
	if opt.Size() > 0 {
		dest = append(dest, opt)
	}
	return dest, nil
}

func negotiateExtensions(
	h []byte, dest []httphead.Option,
	f func(httphead.Option) (httphead.Option, error),
) (_ []httphead.Option, err error) {
	index := -1
	var current httphead.Option
	ok := httphead.ScanOptions(h, func(i int, name, attr, val []byte) httphead.Control {
		if i != index {
			dest, err = negotiateMaybe(current, dest, f)
			if err != nil {
				return httphead.ControlBreak
			}
			index = i
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
	return negotiateMaybe(current, dest, f)
}
