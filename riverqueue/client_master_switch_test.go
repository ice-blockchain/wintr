// SPDX-License-Identifier: ice License 1.0

package riverqueue

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/puddle/v2"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/connectors/storage/v2/fixture"
)

var (
	testContainer *fixture.Container
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	testContainer = fixture.New(ctx)

	code := m.Run()
	cancel()
	testContainer.Close(ctx)

	os.Exit(code)
}

// tcpProxy is a simple TCP proxy that can be interrupted to simulate connection failures.
type tcpProxy struct {
	listener   net.Listener
	targetAddr string
	closed     atomic.Bool
	conns      sync.WaitGroup
	activeConn sync.Map // map[net.Conn]struct{}
}

func newTCPProxy(t *testing.T, targetAddr string) *tcpProxy {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to create proxy listener")

	proxy := &tcpProxy{
		listener:   listener,
		targetAddr: targetAddr,
	}
	t.Logf("Started TCP proxy on %s forwarding to %s", proxy.Addr(), targetAddr)

	go proxy.acceptLoop(t)

	return proxy
}

func (p *tcpProxy) acceptLoop(t *testing.T) {
	t.Helper()

	for {
		conn, err := p.listener.Accept()
		if err != nil {
			if p.closed.Load() {
				return
			}
			t.Logf("proxy accept error: %v", err)
			return
		}

		p.conns.Add(1)
		go p.handleConnection(t, conn)
	}
}

func (p *tcpProxy) handleConnection(t *testing.T, clientConn net.Conn) {
	t.Helper()
	defer p.conns.Done()

	p.activeConn.Store(clientConn, struct{}{})
	defer p.activeConn.Delete(clientConn)

	if p.closed.Load() {
		clientConn.Close()
		return
	}

	targetConn, err := net.DialTimeout("tcp", p.targetAddr, 5*time.Second)
	if err != nil {
		t.Logf("proxy dial error: %v", err)
		clientConn.Close()
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Target.
	go func() {
		defer wg.Done()
		_, _ = io.Copy(targetConn, clientConn)
		targetConn.Close()
	}()

	// Target -> Client.
	go func() {
		defer wg.Done()
		_, _ = io.Copy(clientConn, targetConn)
		clientConn.Close()
	}()

	wg.Wait()
}

func (p *tcpProxy) Addr() string {
	return p.listener.Addr().String()
}

func (p *tcpProxy) Interrupt() {
	// Close all active connections to simulate a connection failure.
	p.activeConn.Range(func(key, _ any) bool {
		if conn, ok := key.(net.Conn); ok {
			conn.Close()
		}
		return true
	})
}

func (p *tcpProxy) Close(t *testing.T) {
	t.Helper()

	t.Logf("Closing TCP proxy on %s", p.Addr())
	p.closed.Store(true)
	p.listener.Close()
	p.Interrupt()
	p.conns.Wait()
}

// extractHostPort extracts host:port from a postgres connection string.
func extractHostPort(t *testing.T, connStr string) string {
	t.Helper()

	u, err := url.Parse(connStr)
	require.NoError(t, err, "failed to parse connection string")

	return u.Host
}

// replaceHostPort replaces the host:port in a postgres connection string.
func replaceHostPort(t *testing.T, connStr, newHostPort string) string {
	t.Helper()

	u, err := url.Parse(connStr)
	require.NoError(t, err, "failed to parse connection string")

	u.Host = newHostPort

	return u.String()
}

func TestDatabaseClientMasterSwitch(t *testing.T) {
	t.Parallel()

	const testClientID = "test-client-master-switch"

	addr, _ := testContainer.MustTempDB(t.Context())
	dbHostPort := extractHostPort(t, addr)

	proxy1 := newTCPProxy(t, dbHostPort)
	defer proxy1.Close(t)

	proxy2 := newTCPProxy(t, dbHostPort)
	defer proxy2.Close(t)

	proxyAddr1 := replaceHostPort(t, addr, proxy1.Addr())
	proxyAddr2 := replaceHostPort(t, addr, proxy2.Addr())

	t.Logf("Proxy 1 address: %s", proxyAddr1)
	t.Logf("Proxy 2 address: %s", proxyAddr2)

	t.Run("SwitchMasterOnConnectionFailure", func(t *testing.T) {
		client, err := newClient(t.Context(),
			"",
			WithConfig(&Config{
				PrimaryURLs: []string{proxyAddr1, proxyAddr2},
				ID:          testClientID,
			}),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
		defer client.Close(t.Context())

		require.NoError(t, client.HealthCheck(t.Context()), "initial health check should pass")
		require.Equal(t, uint64(0), client.DB.CurrentIndex, "should start with first master")

		t.Log("Interrupting proxy 1...")
		proxy1.Close(t)

		// Give some time for connections to be interrupted.
		require.Error(t, client.DB.Ping(t.Context()))

		// The next operation should trigger a master switch.
		// We use switchMaster directly to test the switching logic.
		err = client.DB.switchMaster(t.Context(), io.EOF)
		require.NoError(t, err, "should switch to another master")
		require.Equal(t, uint64(1), client.DB.CurrentIndex, "should switch to second master")

		// Reinitialize River client after master switch.
		require.NoError(t, client.initRiverClient(t.Context(), client.DB.Get()), "should reinitialize river client")

		// Verify new connection works.
		require.NoError(t, client.HealthCheck(t.Context()), "health check should pass after switch")
	})
}

func TestDatabaseClientAllMastersFail(t *testing.T) {
	t.Parallel()

	const testClientID = "test-client-all-masters-fail"

	addr, _ := testContainer.MustTempDB(t.Context())

	dbHostPort := extractHostPort(t, addr)

	proxy1 := newTCPProxy(t, dbHostPort)
	proxy2 := newTCPProxy(t, dbHostPort)

	proxyAddr1 := replaceHostPort(t, addr, proxy1.Addr())
	proxyAddr2 := replaceHostPort(t, addr, proxy2.Addr())

	client, err := newClient(t.Context(),
		"",
		WithConfig(&Config{
			PrimaryURLs: []string{proxyAddr1, proxyAddr2},
			ID:          testClientID,
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close(t.Context())

	require.NoError(t, client.HealthCheck(t.Context()), "initial health check should pass")

	t.Log("Closing all proxies...")
	proxy1.Close(t)
	proxy2.Close(t)

	// Make sure current connection is dead.
	require.Error(t, client.DB.Ping(t.Context()))

	err = client.DB.switchMaster(t.Context(), io.EOF)
	require.ErrorIs(t, err, errNoActiveMaster)
}

func TestShouldSwitchMaster(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		doSwitch bool
	}{
		{
			name:     "nil error",
			err:      nil,
			doSwitch: false,
		},
		{
			name:     "net.OpError",
			err:      &net.OpError{Op: "dial", Err: io.EOF},
			doSwitch: true,
		},
		{
			name:     "generic error",
			err:      io.ErrShortBuffer,
			doSwitch: false,
		},
		{
			name:     "closed pool error",
			err:      puddle.ErrClosedPool,
			doSwitch: true,
		},
		{
			name:     "broken pipe error",
			err:      syscall.EPIPE,
			doSwitch: true,
		},
		{
			name:     "wrapped connection error",
			err:      fmt.Errorf("failed to insert: %w", io.EOF),
			doSwitch: true,
		},
		{
			name:     "unrelated error",
			err:      errors.New("duplicate key value violates unique constraint"),
			doSwitch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSwitchMaster(tt.err)
			require.Equal(t, tt.doSwitch, result)
		})
	}
}

func TestDatabaseClientSwitchBackOnRecovery(t *testing.T) {
	t.Parallel()

	const testClientID = "test-client-switch-back-on-recovery"

	addr, _ := testContainer.MustTempDB(t.Context())

	dbHostPort := extractHostPort(t, addr)
	proxy1 := newTCPProxy(t, dbHostPort)
	defer proxy1.Close(t)
	proxy2 := newTCPProxy(t, dbHostPort)
	defer proxy2.Close(t)

	proxyAddr1 := replaceHostPort(t, addr, proxy1.Addr())
	proxyAddr2 := replaceHostPort(t, addr, proxy2.Addr())

	client, err := newClient(t.Context(),
		"",
		WithConfig(&Config{
			PrimaryURLs: []string{proxyAddr1, proxyAddr2},
			ID:          testClientID,
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close(t.Context())

	require.NoError(t, client.HealthCheck(t.Context()), "initial health check should pass")
	require.Equal(t, uint64(0), client.DB.CurrentIndex, "should start with first master")

	// Close proxy1 to force switch to proxy2.
	proxy1.Close(t)
	require.Error(t, client.DB.Ping(t.Context()))

	err = client.DB.switchMaster(t.Context(), io.EOF)
	require.NoError(t, err)
	require.Equal(t, uint64(1), client.DB.CurrentIndex, "should switch to second master")

	// Reinitialize the client with the new connection.
	require.NoError(t, client.initRiverClient(t.Context(), client.DB.Get()))
	require.NoError(t, client.HealthCheck(t.Context()), "health check should pass after first switch")

	// Create a new proxy1 (simulating recovery).
	proxy1New := newTCPProxy(t, dbHostPort)
	defer proxy1New.Close(t)

	// Update the masters list with new proxy address.
	client.DB.Masters[0], err = createPgURL("", "", replaceHostPort(t, addr, proxy1New.Addr()))
	require.NoError(t, err)

	// Close proxy2 to force switch back.
	proxy2.Close(t)
	require.Error(t, client.DB.Ping(t.Context()))

	err = client.DB.switchMaster(t.Context(), io.EOF)
	require.NoError(t, err)
	require.Equal(t, uint64(0), client.DB.CurrentIndex, "should switch back to first master")

	// Reinitialize and verify.
	require.NoError(t, client.initRiverClient(t.Context(), client.DB.Get()))
	require.NoError(t, client.HealthCheck(t.Context()), "health check should pass after switching back")
}

func TestPushWithMasterSwitch(t *testing.T) {
	t.Parallel()

	const testClientID = "test-client-push-with-master-switch"

	addr, _ := testContainer.MustTempDB(t.Context())

	dbHostPort := extractHostPort(t, addr)

	proxy1 := newTCPProxy(t, dbHostPort)
	defer proxy1.Close(t)
	proxy2 := newTCPProxy(t, dbHostPort)
	defer proxy2.Close(t)

	proxyAddr1 := replaceHostPort(t, addr, proxy1.Addr())
	proxyAddr2 := replaceHostPort(t, addr, proxy2.Addr())

	client, err := newClient(t.Context(),
		"",
		WithConfig(&Config{
			PrimaryURLs: []string{proxyAddr1, proxyAddr2},
			ID:          testClientID,
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close(t.Context())

	results := make(chan int, 10)
	RegisterWorker(client.Register(), &testAddWorker{T: t, Result: results})

	require.NoError(t, client.Start(t.Context()))

	// First job: Push successfully through proxy1.
	t.Log("Pushing first job...")
	args1 := &testAddWorkerArgs{A: 5, B: 3}
	require.NoError(t, client.Push(t.Context(), args1))

	select {
	case res := <-results:
		require.Equal(t, 8, res)
		t.Log("First job completed successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first job result")
	}

	// Verify we're on the first master.
	require.Equal(t, uint64(0), client.DB.CurrentIndex, "should be on first master")

	// Close proxy1 to simulate master failure.
	t.Log("Closing proxy1 to simulate master failure...")
	proxy1.Close(t)

	// Wait for connections to be interrupted.
	require.Error(t, client.DB.Ping(t.Context()))

	t.Log("Pushing second job (should trigger master switch)...")
	args2 := &testAddWorkerArgs{A: 10, B: 7}
	err = client.Push(t.Context(), args2)
	require.NoError(t, err, "second push should succeed after master switch")

	// Verify master was switched.
	require.Equal(t, uint64(1), client.DB.CurrentIndex, "should have switched to second master")

	select {
	case res := <-results:
		require.Equal(t, 17, res)
		t.Log("Second job completed successfully after master switch")
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for second job result")
	}

	// Verify health check still works.
	require.NoError(t, client.HealthCheck(t.Context()), "health check should pass after switch")
}
