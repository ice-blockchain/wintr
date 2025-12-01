// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"database/sql"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/connectors/storage/v2/fixture"
)

func TestMigrationFromFilesystemDDL(t *testing.T) {
	t.Parallel()

	container := fixture.New(t.Context())
	connString, release := container.MustTempDB(t.Context())
	defer release()

	t.Logf("Running migrations on %s", connString)

	fsys := fstest.MapFS{
		"001_init.sql": &fstest.MapFile{
			Data: []byte(`CREATE TABLE foo (data text);`),
		},
		"002_add_column.sql": &fstest.MapFile{
			Data: []byte(`ALTER TABLE foo ADD COLUMN num int;`),
		},
	}

	conn := mustConnectPool(t.Context(), "", "", "", connString, false)
	require.NotNil(t, conn)
	defer conn.Close()

	m := NewFilesystemDDL(fsys, "")
	err := m.run(t.Context(), conn)
	require.NoError(t, err, "migration failed")

	var (
		data string
		num  int
	)
	err = conn.QueryRow(t.Context(), `SELECT data, num FROM foo`).Scan(&data, &num)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
