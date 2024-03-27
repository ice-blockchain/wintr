// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"context"
	h2ec "github.com/ice-blockchain/go/src/net/http"
	"net"
	"net/http"
	"strings"
	"syscall"
	stdlibtime "time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
)

func NewWebSocketAdapter(ctx context.Context, conn net.Conn, readTimeout, writeTimeout stdlibtime.Duration) (WSWithWriter, context.Context) {
	wt := &WebsocketAdapter{
		conn:         conn,
		closeChannel: make(chan struct{}, 1),
		out:          make(chan wsWrite),
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}

	return wt, NewCustomCancelContext(ctx, wt.closeChannel)
}

func (w *WebsocketAdapter) writeMessageToWebsocket(messageType int, data []byte) error {
	select {
	case <-w.closeChannel:
		return nil
	default:
		var err error
		if w.writeTimeout > 0 {
			err = multierror.Append(nil, w.conn.SetWriteDeadline(time.Now().Add(w.writeTimeout)))
		}
		w.closeMx.Lock()
		if w.closed {
			w.closeMx.Unlock()
			return nil
		}
		w.closeMx.Unlock()
		wErr := wsutil.WriteServerMessage(w.conn, ws.OpCode(messageType), data)
		w.wrErrMx.Lock()
		w.wrErr = wErr
		w.wrErrMx.Unlock()
		if isConnClosedErr(wErr) {
			wErr = nil
		}
		err = multierror.Append(err,
			wErr,
		).ErrorOrNil()

		if flusher, ok := w.conn.(http.Flusher); err == nil && ok {
			flusher.Flush()
		}

		return errors.Wrapf(err, "failed to write data to websocket")
	}
}

func (w *WebsocketAdapter) WriteMessage(messageType int, data []byte) error {
	select {
	case <-w.closeChannel:
		return nil
	default:
		w.wrErrMx.Lock()
		if isConnClosedErr(w.wrErr) {
			w.wrErrMx.Unlock()
			return w.Close()
		}
		w.wrErrMx.Unlock()
		w.out <- wsWrite{
			opCode: messageType,
			data:   data,
		}
	}

	return nil
}

func (w *WebsocketAdapter) Write(ctx context.Context) {
	for msg := range w.out {
		if ctx.Err() != nil || isConnClosedErr(w.wrErr) {
			break
		}
		log.Error(w.writeMessageToWebsocket(msg.opCode, msg.data), "failed to send message to websocket")
	}
}

func (w *WebsocketAdapter) ReadMessage() (messageType int, p []byte, err error) {
	if w.readTimeout > 0 {
		_ = w.conn.SetReadDeadline(time.Now().Add(w.readTimeout)) //nolint:errcheck // It is not crucial if we ignore it here.
	}
	msgBytes, typ, err := wsutil.ReadClientData(w.conn)
	if err != nil {
		return int(typ), msgBytes, err
	}
	if typ == ws.OpPing {
		err = wsutil.WriteServerMessage(w.conn, ws.OpPong, nil)
		if err == nil {
			return w.ReadMessage()
		}

		return int(typ), msgBytes, err
	}

	return int(typ), msgBytes, err
}
func (w *WebsocketAdapter) Closed() bool {
	w.closeMx.Lock()
	defer w.closeMx.Unlock()
	return w.closed
}
func (w *WebsocketAdapter) Close() error {
	w.closeMx.Lock()
	if w.closed {
		w.closeMx.Unlock()

		return nil
	}
	w.closed = true
	close(w.closeChannel)
	w.closeMx.Unlock()
	var wErr error
	if w.wrErr == nil || !isConnClosedErr(w.wrErr) {
		wErr = wsutil.WriteServerMessage(w.conn, ws.OpClose, ws.NewCloseFrameBody(ws.StatusNormalClosure, ""))
		if wErr != nil && isConnClosedErr(wErr) {
			wErr = nil
		}
	}
	clErr := w.conn.Close()
	if clErr != nil && isConnClosedErr(clErr) {
		clErr = nil
	}

	return multierror.Append( //nolint:wrapcheck // .
		wErr,
		clErr,
	).ErrorOrNil()
}

func isConnClosedErr(err error) bool {
	return err != nil &&
		(errors.Is(err, syscall.EPIPE) ||
			errors.Is(err, syscall.ECONNRESET) ||
			errors.Is(err, h2ec.Http2errClientDisconnected) ||
			errors.Is(err, h2ec.Http2errStreamClosed) ||
			strings.Contains(err.Error(), "convert stream error 386759528") ||
			strings.Contains(err.Error(), "canceled by remote with error code 256") ||
			strings.Contains(err.Error(), "use of closed network connection"))
}
