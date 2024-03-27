package wsserver_test

import (
	"context"
	"github.com/gobwas/ws"
	"github.com/google/uuid"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/wsserver"
	"github.com/ice-blockchain/wintr/wsserver/fixture"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	stdlibtime "time"
)

const (
	connCountTCP = 100
	connCountUDP = 100
	testDeadline = 30 * stdlibtime.Second
)

func TestSimpleEchoDifferentTransports(t *testing.T) {
	t.Run("webtransport http 3", func(t *testing.T) {
		testEcho(t, connCountUDP, func(ctx context.Context) (fixture.Client, error) {
			return fixture.NewWebTransportClientHttp3(ctx, "https://localhost:9999/")
		})
	})
	t.Run("websocket http 3", func(t *testing.T) {
		testEcho(t, connCountUDP, func(ctx context.Context) (fixture.Client, error) {
			return fixture.NewWebsocketClientHttp3(ctx, "https://localhost:9999/")
		})
	})

	t.Run("websocket http 2", func(t *testing.T) {
		testEcho(t, connCountTCP, func(ctx context.Context) (fixture.Client, error) {
			return fixture.NewWebsocketClientHttp2(ctx, "https://localhost:9999/")
		})
	})

	t.Run("websocket http 1.1", func(t *testing.T) {
		testEcho(t, connCountTCP, func(ctx context.Context) (fixture.Client, error) {
			return fixture.NewWebsocketClient(ctx, "wss://localhost:9999/")
		})
	})
}

func testEcho(t *testing.T, conns int, client func(ctx context.Context) (fixture.Client, error)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testDeadline)
	defer cancel()
	var handlersMx sync.Mutex
	handlers := make(map[wsserver.WSWriter]struct{}, conns)
	echoFunc := func(w wsserver.WSWriter, in string) error {
		handlersMx.Lock()
		handlers[w] = struct{}{}
		handlersMx.Unlock()
		return w.WriteMessage(int(ws.OpText), []byte("server reply:"+in))
	}
	srv := fixture.NewTestServer(ctx, cancel, echoFunc)
	stdlibtime.Sleep(100 * stdlibtime.Millisecond)
	var wg sync.WaitGroup
	var clients []fixture.Client
	for i := 0; i < conns; i++ {
		clientConn, err := client(ctx)
		if err != nil {
			log.Panic(err)
		}
		clients = append(clients, clientConn)
	}
	for i := 0; i < conns; i++ {
		wg.Add(1)
		go func(ii int) {
			defer wg.Done()
			clientConn := clients[ii]
			defer clientConn.Close()
			sendMsgs := make([]string, 0)
			sendMsgsTransformed := make([]string, 0)
			receivedBackOnClient := make([]string, 0)
			go func() {
				receivedCh := clientConn.Received()
				for received := range receivedCh {
					receivedBackOnClient = append(receivedBackOnClient, string(received))
					assert.Equal(t, sendMsgsTransformed[0:len(receivedBackOnClient)], receivedBackOnClient)
				}
			}()
			for ctx.Err() == nil {
				msg := uuid.NewString()
				sendMsgs = append(sendMsgs, msg)
				sendMsgsTransformed = append(sendMsgsTransformed, "server reply:"+msg)
				err := clientConn.WriteMessage(int(ws.OpText), []byte(msg))
				if ctx.Err() == nil {
					assert.NoError(t, err)
				}
			}
			assert.GreaterOrEqual(t, len(receivedBackOnClient), 0)
		}(i)
	}
	wg.Wait()
	shutdownCtx, _ := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	for srv.ReaderExited.Load() != uint64(conns) {
		if shutdownCtx.Err() != nil {
			log.Panic(errors.Errorf("shutdown timeout %v of %v", srv.ReaderExited.Load(), conns))
		}
		stdlibtime.Sleep(100 * stdlibtime.Millisecond)
	}
	require.Equal(t, uint64(conns), srv.ReaderExited.Load())
	require.Len(t, handlers, conns)
	for w := range handlers {
		var closed bool
		switch h := w.(type) {
		case *internal.WebsocketAdapter:
			closed = h.Closed()
		case *internal.WebtransportAdapter:
			closed = h.Closed()
		default:
			panic("unknown protocol implementation")
		}
		require.True(t, closed)
	}
}
