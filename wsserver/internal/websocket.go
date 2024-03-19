// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"context"
	"net"
	"net/http"
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
	var err error
	if w.writeTimeout > 0 {
		err = multierror.Append(nil, w.conn.SetWriteDeadline(time.Now().Add(w.writeTimeout)))
	}
	err = multierror.Append(err,
		wsutil.WriteServerMessage(w.conn, ws.OpCode(messageType), data),
	).ErrorOrNil()

	if flusher, ok := w.conn.(http.Flusher); err == nil && ok {
		flusher.Flush()
	}

	return errors.Wrapf(err, "failed to write data to websocket")
}

func (w *WebsocketAdapter) WriteMessage(messageType int, data []byte) error {
	w.out <- wsWrite{
		opCode: messageType,
		data:   data,
	}

	return nil
}

func (w *WebsocketAdapter) Write(ctx context.Context) {
	for msg := range w.out {
		if ctx.Err() != nil {
			break
		}
		log.Error(w.writeMessageToWebsocket(msg.opCode, msg.data), "failed to send message to webtransport")
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

func (w *WebsocketAdapter) Close() error {
	close(w.closeChannel)
	close(w.out)

	return multierror.Append( //nolint:wrapcheck // .
		wsutil.WriteServerMessage(w.conn, ws.OpClose, ws.NewCloseFrameBody(ws.StatusNormalClosure, "")),
		w.conn.Close(),
	).ErrorOrNil()
}
