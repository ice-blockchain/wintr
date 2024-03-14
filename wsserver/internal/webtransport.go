// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"context"
	"io"
	stdlibtime "time"

	"github.com/pkg/errors"
	"github.com/quic-go/webtransport-go"

	"github.com/ice-blockchain/wintr/time"
)

func NewWebTransportAdapter(ctx context.Context, stream webtransport.Stream, readTimeout, writeTimeout stdlibtime.Duration) (WS, context.Context) {
	wt := &WebtransportAdapter{
		stream:       stream,
		closeChannel: make(chan struct{}, 1),
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}

	return wt, NewCustomCancelContext(ctx, wt.closeChannel)
}

func (w *WebtransportAdapter) WriteMessage(_ int, data []byte) (err error) {
	if w.writeTimeout > 0 {
		_ = w.stream.SetWriteDeadline(time.Now().Add(w.writeTimeout)) //nolint:errcheck // .
	}
	_, err = w.stream.Write(data)

	return errors.Wrapf(err, "failed to write data to webtransport stream")
}

func (w *WebtransportAdapter) Close() error {
	close(w.closeChannel)

	return errors.Wrap(w.stream.Close(), "failed to close http3/webtransport stream")
}

func (w *WebtransportAdapter) ReadMessage() (messageType int, readValue []byte, err error) {
	if w.readTimeout > 0 {
		_ = w.stream.SetReadDeadline(time.Now().Add(w.readTimeout)) //nolint:errcheck // .
	}
	readValue, err = io.ReadAll(w.stream)

	return 1, readValue, errors.Wrapf(err, "failed to read data from webtransport stream")
}