// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	stdlibtime "time"

	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/connectors/storage/v2/fixture"
	"github.com/ice-blockchain/wintr/terror"
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

func TestMustConnect(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	ddl := `
create table if not exists bogus 
(
    a  text not null unique,
    b  integer not null check (b >= 0),
    c  boolean not null default false,
    primary key(a, b, c)
);
----
create table if not exists bogus2 
(
    a  text not null unique REFERENCES bogus(a) ON DELETE CASCADE,
    b  integer not null primary key check (b >= 0),
    c  boolean not null default false
);
----
CREATE OR REPLACE FUNCTION doSomething(tableName text, count smallint)
  RETURNS VOID AS
$$
BEGIN
    FOR worker_index IN 0 .. count-1 BY 1
    LOOP
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %s_%s PARTITION OF %s FOR VALUES WITH (MODULUS %s,REMAINDER %s);',
           tableName,
           worker_index,
           tableName,
           count,
           worker_index
        );
    END LOOP;
END
$$ LANGUAGE plpgsql;`
	type (
		Bogus struct {
			A string
			B int
			C bool
		}
	)

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, &stringDDL{Data: ddl})
	defer func() {
		_, err := Exec(t.Context(), db, `DROP TABLE bogus2`)
		require.NoError(t, err)
		_, err = Exec(t.Context(), db, `DROP TABLE bogus`)
		require.NoError(t, err)
		_, err = Exec(t.Context(), db, `DROP function doSomething`)
		require.NoError(t, err)
		require.NoError(t, db.Close())
	}()
	require.NoError(t, db.Ping(t.Context()))
	rowsAffected, err := Exec(t.Context(), db, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3)`, "a1", 1, true)
	require.NoError(t, err)
	require.EqualValues(t, 1, rowsAffected)
	rowsAffected, err = Exec(t.Context(), db, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3)`, "a1", 1, true)
	require.ErrorIs(t, err, ErrDuplicate)
	require.EqualValues(t, terror.New(ErrDuplicate, map[string]any{"column": "pk"}), err)
	require.True(t, IsErr(err, ErrDuplicate, "pk"))
	require.EqualValues(t, 0, rowsAffected)
	rowsAffected, err = Exec(t.Context(), db, `INSERT INTO bogus2(a,b,c) VALUES ($1,$2,$3)`, "a1", 1, true)
	require.NoError(t, err)
	require.EqualValues(t, 1, rowsAffected)
	rowsAffected, err = Exec(t.Context(), db, `INSERT INTO bogus2(a,b,c) VALUES ($1,$2,$3)`, "a1", 2, true)
	require.ErrorIs(t, err, ErrDuplicate)
	require.EqualValues(t, terror.New(ErrDuplicate, map[string]any{"column": "a"}), err)
	require.True(t, IsErr(err, ErrDuplicate, "a"))
	require.EqualValues(t, 0, rowsAffected)
	rowsAffected, err = Exec(t.Context(), db, `INSERT INTO bogus2(a,b,c) VALUES ($1,$2,$3)`, "a2", 1, true)
	require.ErrorIs(t, err, ErrDuplicate)
	require.EqualValues(t, terror.New(ErrDuplicate, map[string]any{"column": "pk"}), err)
	require.EqualValues(t, 0, rowsAffected)
	rowsAffected, err = Exec(t.Context(), db, `INSERT INTO bogus2(a,b,c) VALUES ($1,$2,$3)`, "axx", 33, true)
	require.ErrorIs(t, err, ErrRelationNotFound)
	require.EqualValues(t, terror.New(ErrRelationNotFound, map[string]any{"column": "a"}), err)
	require.True(t, IsErr(err, ErrRelationNotFound, "a"))
	require.EqualValues(t, 0, rowsAffected)
	res1, err := ExecOne[Bogus](t.Context(), db, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3) RETURNING *`, "a2", 2, true)
	require.NoError(t, err)
	require.EqualValues(t, &Bogus{A: "a2", B: 2, C: true}, res1)
	res2, err := ExecMany[Bogus](t.Context(), db, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3),($4,$5,$6) RETURNING *`, "a3", 3, true, "a4", 4, false)
	require.NoError(t, err)
	require.EqualValues(t, []*Bogus{{A: "a3", B: 3, C: true}, {A: "a4", B: 4, C: false}}, res2)
	res3, err := Get[Bogus](t.Context(), db, `SELECT * FROM bogus WHERE a = $1`, "a1") //nolint:unqueryvet // .
	require.NoError(t, err)
	require.EqualValues(t, &Bogus{A: "a1", B: 1, C: true}, res3)
	resX, err := Get[Bogus](t.Context(), db, `SELECT * FROM bogus WHERE a = $1`, "axxx") //nolint:unqueryvet // .
	require.ErrorIs(t, err, ErrNotFound)
	require.True(t, IsErr(err, ErrNotFound))
	require.Nil(t, resX)
	res4, err := Select[Bogus](t.Context(), db, `SELECT * FROM bogus WHERE a != $1  ORDER BY b`, "b") //nolint:unqueryvet // .
	require.NoError(t, err)
	require.EqualValues(t, []*Bogus{{A: "a1", B: 1, C: true}, {A: "a2", B: 2, C: true}, {A: "a3", B: 3, C: true}, {A: "a4", B: 4, C: false}}, res4)
	require.NoError(t, DoInTransaction(t.Context(), db, func(conn QueryExecer) error {
		rowsAffected, err = Exec(t.Context(), conn, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3)`, "a5", 5, true)
		require.NoError(t, err)
		if err != nil {
			return err
		}
		require.EqualValues(t, 1, rowsAffected)
		res1, err = ExecOne[Bogus](t.Context(), conn, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3) RETURNING *`, "a6", 6, true)
		require.NoError(t, err)
		if err != nil {
			return err
		}
		require.EqualValues(t, &Bogus{A: "a6", B: 6, C: true}, res1)
		res2, err = ExecMany[Bogus](t.Context(), conn, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3),($4,$5,$6) RETURNING *`, "a7", 7, true, "a8", 8, false)
		require.NoError(t, err)
		if err != nil {
			return err
		}
		require.EqualValues(t, []*Bogus{{A: "a7", B: 7, C: true}, {A: "a8", B: 8, C: false}}, res2)
		res3, err = Get[Bogus](t.Context(), conn, `SELECT * FROM bogus WHERE a = $1`, "a5") //nolint:unqueryvet // .
		require.NoError(t, err)
		if err != nil {
			return err
		}
		require.EqualValues(t, &Bogus{A: "a5", B: 5, C: true}, res3)
		res4, err = Select[Bogus](t.Context(), conn, `SELECT * FROM bogus WHERE a != $1  ORDER BY b`, "bb") //nolint:unqueryvet // .
		require.NoError(t, err)
		if err != nil {
			return err
		}
		require.EqualValues(t, []*Bogus{{A: "a1", B: 1, C: true}, {A: "a2", B: 2, C: true}, {A: "a3", B: 3, C: true}, {A: "a4", B: 4, C: false}, {A: "a5", B: 5, C: true}, {A: "a6", B: 6, C: true}, {A: "a7", B: 7, C: true}, {A: "a8", B: 8, C: false}}, res4) //nolint:lll // .

		return nil
	}))
}

func TestStopWhenTxAborted(t *testing.T) {
	t.Parallel()

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	ddl := `CREATE TABLE test_abort (id INT PRIMARY KEY);`

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, &stringDDL{Data: ddl})
	require.NotNil(t, db)
	defer func() {
		require.NoError(t, db.Close())
	}()

	_, err := Exec(t.Context(), db, `INSERT INTO test_abort (id) VALUES (1)`)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 2*stdlibtime.Second)
	defer cancel()

	err = DoInTransaction(ctx, db, func(tx QueryExecer) error {
		_, insertErr := Exec(ctx, tx, `INSERT INTO test_abort (id) VALUES (1)`)
		if insertErr == nil {
			t.Fatal("expected duplicate key error")
		}

		_, queryErr := Get[int](ctx, tx, `SELECT id FROM test_abort WHERE id = 1`)

		return queryErr
	})

	require.True(t, errors.Is(err, ErrTxAborted) || errors.Is(err, context.DeadlineExceeded),
		"expected ErrTxAborted or DeadlineExceeded, got: %v", err)
}

func TestListenNotify_SuccessfulFlow(t *testing.T) {
	t.Parallel()

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, nil)
	require.NotNil(t, db)
	defer func() {
		require.NoError(t, db.Close())
	}()

	ctx := t.Context()
	channel := "test_channel_success"

	listener, err := db.Listen(ctx, channel)
	require.NoError(t, err)
	require.NotNil(t, listener)
	defer func() { _ = listener.Close() }()

	notifications := []string{"message1", "message2", "message3"}
	for _, msg := range notifications {
		_, err := Exec(ctx, db, `SELECT pg_notify($1, $2)`, channel, msg)
		require.NoError(t, err)
	}

	received := make([]string, 0, len(notifications))
	require.Eventually(t, func() bool {
		select {
		case notif := <-listener.Channel():
			require.NotNil(t, notif)
			require.Equal(t, channel, notif.Channel)
			received = append(received, notif.Payload)
			return len(received) == len(notifications)
		default:
			return false
		}
	}, 5*stdlibtime.Second, 10*stdlibtime.Millisecond, "timeout waiting for all notifications")

	for _, msg := range notifications {
		require.Contains(t, received, msg)
	}
	require.Nil(t, listener.Err())
}

func TestListenNotify_ListenCommandFails(t *testing.T) {
	t.Parallel()

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, nil)
	require.NotNil(t, db)

	require.NoError(t, db.Close())

	ctx := t.Context()
	channel := "test_channel"

	listener, err := db.Listen(ctx, channel)
	require.Error(t, err)
	require.Nil(t, listener)
	require.Contains(t, err.Error(), "failed to acquire connection")
}

func TestListenNotify_CloseCleanup(t *testing.T) {
	t.Parallel()

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, nil)
	require.NotNil(t, db)
	defer func() {
		require.NoError(t, db.Close())
	}()

	ctx := t.Context()
	channel := "test_channel_cleanup"

	listener, err := db.Listen(ctx, channel)
	require.NoError(t, err)
	require.NotNil(t, listener)

	err = listener.Close()
	require.True(t, err == nil || errors.Is(err, context.Canceled), "unexpected error: %v", err)

	_, ok := <-listener.Channel()
	require.False(t, ok, "channel should be closed after Close()")

	err = listener.Close()
	require.NoError(t, err, "double close should not return error")
}

func TestListenNotify_ContextCancellation(t *testing.T) {
	t.Parallel()

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, nil)
	require.NotNil(t, db)
	defer func() {
		require.NoError(t, db.Close())
	}()

	ctx, cancel := context.WithCancel(t.Context())
	channel := "test_channel_ctx_cancel"

	listener, err := db.Listen(ctx, channel)
	require.NoError(t, err)
	require.NotNil(t, listener)
	defer func() { _ = listener.Close() }()

	cancel()

	require.Eventually(t, func() bool {
		select {
		case _, ok := <-listener.Channel():
			require.False(t, ok, "channel should be closed after context cancellation")
			return true
		default:
			return false
		}
	}, 1*stdlibtime.Second, 50*stdlibtime.Millisecond, "timeout waiting for channel to close")
}

func TestListenNotify_NotificationBackpressure(t *testing.T) {
	t.Parallel()

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, nil)
	require.NotNil(t, db)
	defer func() {
		require.NoError(t, db.Close())
	}()

	ctx := t.Context()
	channel := "test_channel_backpressure"

	listener, err := db.Listen(ctx, channel)
	require.NoError(t, err)
	require.NotNil(t, listener)
	defer func() { _ = listener.Close() }()

	numNotifications := 2000
	for i := 0; i < numNotifications; i++ {
		_, err := Exec(ctx, db, `SELECT pg_notify($1, $2)`, channel, "message")
		require.NoError(t, err)
	}

	receivedCount := 0
	timeout := stdlibtime.After(5 * stdlibtime.Second)
	for {
		select {
		case notif := <-listener.Channel():
			if notif == nil {
				require.Greater(t, receivedCount, 0, "should have received at least some notifications")

				return
			}
			receivedCount++
		case <-timeout:
			require.Greater(t, receivedCount, 0, "should have received at least some notifications before timeout")

			return
		}
	}
}

func TestListenNotify_MultipleListeners(t *testing.T) {
	t.Parallel()

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, nil)
	require.NotNil(t, db)
	defer func() {
		require.NoError(t, db.Close())
	}()

	ctx := t.Context()
	channel := "test_channel_multiple"
	numListeners := 3

	var listeners []*Listener
	for i := 0; i < numListeners; i++ {
		listener, err := db.Listen(ctx, channel)
		require.NoError(t, err)
		require.NotNil(t, listener)
		listeners = append(listeners, listener)
		defer func(l *Listener) { _ = l.Close() }(listener)
	}

	testMessage := "broadcast_message"
	_, err := Exec(ctx, db, `SELECT pg_notify($1, $2)`, channel, testMessage)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(numListeners)

	for i, listener := range listeners {
		go func(idx int, l *Listener) {
			defer wg.Done()
			require.Eventually(t, func() bool {
				select {
				case notif := <-l.Channel():
					require.NotNil(t, notif)
					require.Equal(t, channel, notif.Channel)
					require.Equal(t, testMessage, notif.Payload)
					return true
				default:
					return false
				}
			}, 5*stdlibtime.Second, 10*stdlibtime.Millisecond, "listener %d: timeout waiting for notification", idx)
		}(i, listener)
	}

	wg.Wait()
}

func TestListenNotify_ErrorReporting(t *testing.T) {
	t.Parallel()

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, nil)
	require.NotNil(t, db)
	defer func() {
		require.NoError(t, db.Close())
	}()

	ctx := t.Context()
	channel := "test_channel_error_reporting"

	listener, err := db.Listen(ctx, channel)
	require.NoError(t, err)
	require.NotNil(t, listener)
	defer func() { _ = listener.Close() }()

	require.Nil(t, listener.Err())

	_, err = Exec(ctx, db, `SELECT pg_notify($1, $2)`, channel, "test")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		select {
		case notif := <-listener.Channel():
			require.NotNil(t, notif)
			return true
		default:
			return false
		}
	}, 2*stdlibtime.Second, 10*stdlibtime.Millisecond, "timeout waiting for notification")

	require.Nil(t, listener.Err())
}

func TestListenNotify_ConnectionRecovery(t *testing.T) {
	t.Parallel()

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, nil)
	require.NotNil(t, db)
	defer func() {
		require.NoError(t, db.Close())
	}()

	ctx := t.Context()
	channel := "test_channel_reconnect"

	listener, err := db.Listen(ctx, channel)
	require.NoError(t, err)
	require.NotNil(t, listener)
	defer func() { _ = listener.Close() }()

	var initialPID uint32
	require.Eventually(t, func() bool {
		initialPID = listener.BackendPID()
		return initialPID != 0
	}, 2*stdlibtime.Second, 50*stdlibtime.Millisecond, "listener should have non-zero backend PID")

	_, err = Exec(ctx, db, `SELECT pg_notify($1, $2)`, channel, "before_kill")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		select {
		case notif := <-listener.Channel():
			require.NotNil(t, notif)
			require.Equal(t, "before_kill", notif.Payload)
			return true
		default:
			return false
		}
	}, 2*stdlibtime.Second, 10*stdlibtime.Millisecond, "timeout waiting for first notification")

	_, err = Exec(ctx, db, `SELECT pg_terminate_backend($1)`, initialPID)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		newPID := listener.BackendPID()
		return newPID != 0 && newPID != initialPID
	}, 5*stdlibtime.Second, 100*stdlibtime.Millisecond, "should reconnect with new PID")

	require.Eventually(t, func() bool {
		_, err = Exec(ctx, db, `SELECT pg_notify($1, $2)`, channel, "after_reconnect")
		if err != nil {
			return false
		}

		select {
		case notif := <-listener.Channel():
			if notif != nil {
				require.Equal(t, "after_reconnect", notif.Payload)
				return true
			}
		default:
		}
		return false
	}, 5*stdlibtime.Second, 100*stdlibtime.Millisecond, "should receive notification after reconnection")
}

func TestListenNotify_ReconnectionWithMultipleFailures(t *testing.T) {
	t.Parallel()

	connString, release := testContainer.MustTempDB(t.Context())
	defer release()

	cfg := &Cfg{
		PrimaryURL:  connString,
		ReplicaURLs: []string{connString},
		RunDDL:      true,
	}

	db := MustConnectWithCfg(t.Context(), cfg, nil)
	require.NotNil(t, db)
	defer func() {
		require.NoError(t, db.Close())
	}()

	ctx := t.Context()
	channel := "test_channel_multi_reconnect"

	listener, err := db.Listen(ctx, channel)
	require.NoError(t, err)
	require.NotNil(t, listener)
	defer func() { _ = listener.Close() }()

	_, err = Exec(ctx, db, `SELECT pg_notify($1, $2)`, channel, "message_0")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		select {
		case notif := <-listener.Channel():
			require.NotNil(t, notif)
			require.Equal(t, "message_0", notif.Payload)
			return true
		default:
			return false
		}
	}, 2*stdlibtime.Second, 10*stdlibtime.Millisecond, "timeout waiting for initial notification")

	for i := 1; i <= 3; i++ {
		var currentPID uint32
		require.Eventually(t, func() bool {
			currentPID = listener.BackendPID()
			return currentPID != 0
		}, 5*stdlibtime.Second, 100*stdlibtime.Millisecond, "iteration %d: should have valid PID before termination", i)

		_, err = Exec(ctx, db, `SELECT pg_terminate_backend($1)`, currentPID)
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			newPID := listener.BackendPID()
			return newPID != 0 && newPID != currentPID
		}, 5*stdlibtime.Second, 100*stdlibtime.Millisecond, "iteration %d: should reconnect with new PID", i)

		messagePayload := fmt.Sprintf("message_%d", i)
		require.Eventually(t, func() bool {
			_, err = Exec(ctx, db, `SELECT pg_notify($1, $2)`, channel, messagePayload)
			if err != nil {
				return false
			}

			select {
			case notif := <-listener.Channel():
				if notif != nil {
					return true
				}
			default:
			}
			return false
		}, 5*stdlibtime.Second, 100*stdlibtime.Millisecond, "iteration %d: should receive notification after reconnection", i)
	}
}
