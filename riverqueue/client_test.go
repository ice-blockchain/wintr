// SPDX-License-Identifier: ice License 1.0

package riverqueue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type (
	testAddWorkerArgs struct {
		A int
		B int
	}
	testAddWorker struct {
		T      *testing.T
		Result chan int
		WorkerDefaults[testAddWorkerArgs]
	}
	testSubtractWorkerArgs struct {
		A int
		B int
	}
	testSubtractWorker struct {
		T      *testing.T
		Result chan int
		WorkerDefaults[testSubtractWorkerArgs]
	}
)

func (testSubtractWorkerArgs) Kind() string {
	return "test_math_sub_args"
}

func (testAddWorkerArgs) Kind() string {
	return "test_math_add_args"
}

func (w *testAddWorker) Work(ctx context.Context, job *Job[testAddWorkerArgs]) error {
	r := job.Args.A + job.Args.B
	w.T.Logf("Addition result: %d + %d = %d", job.Args.A, job.Args.B, r)
	select {
	case w.Result <- r:
	case <-ctx.Done():
		w.T.Error("failed to send result, context done")
	}
	return nil
}

func (w *testSubtractWorker) Work(ctx context.Context, job *Job[testSubtractWorkerArgs]) error {
	r := job.Args.A - job.Args.B
	w.T.Logf("Subtraction result: %d - %d = %d", job.Args.A, job.Args.B, r)
	select {
	case w.Result <- r:
	case <-ctx.Done():
		w.T.Error("failed to send result, context done")
	}
	return nil
}

func TestProcessDifferentJobs(t *testing.T) {
	t.Parallel()

	const testClientID = "test_client_master_switch"
	addr, _ := testContainer.MustTempDB(t.Context())

	client, err := newClient(t.Context(),
		"",
		WithConfig(&Config{
			PrimaryURLs: []string{addr},
			ID:          testClientID,
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, client)

	addResults := make(chan int, 1)
	substractResults := make(chan int, 1)

	RegisterWorker(client.Register(), &testAddWorker{T: t, Result: addResults})
	RegisterWorker(client.Register(), &testSubtractWorker{T: t, Result: substractResults})

	require.NoError(t, client.Start(t.Context()))

	t.Run("Addition job", func(t *testing.T) {
		args := &testAddWorkerArgs{A: 10, B: 15}
		require.NoError(t, client.Push(t.Context(), args))
		select {
		case res := <-addResults:
			require.Equal(t, 25, res)
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for addition result")
		}
	})
	t.Run("Subtraction job", func(t *testing.T) {
		args := &testSubtractWorkerArgs{A: 20, B: 8}
		require.NoError(t, client.Push(t.Context(), args))
		select {
		case res := <-substractResults:
			require.Equal(t, 12, res)
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for subtraction result")
		}
	})
	require.NoError(t, client.Close(t.Context()))
}
