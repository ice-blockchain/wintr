// // SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"math/rand/v2"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	db := New(t.Context())
	connStr := db.ConnectionString(t.Context(), pgDatabase)
	require.NotEmpty(t, connStr)

	t.Logf("Postgres container started at %s", connStr)

	require.NoError(t, db.Close(t.Context()))
}

func TestMustTempDB(t *testing.T) {
	t.Parallel()

	maindb := New(t.Context())
	defer maindb.Close(t.Context())

	var wg sync.WaitGroup
	wg.Add(runtime.NumCPU())

	for range runtime.NumCPU() {
		go func() {
			defer wg.Done()

			n := rand.Uint() % 10
			for range n {
				_, release := maindb.MustTempDB(t.Context())
				release()
			}
		}()
	}

	t.Logf("waiting for all goroutines to finish...")
	wg.Wait()
}
