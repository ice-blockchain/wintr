// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/terror"
)

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
	db := MustConnect(context.Background(), ddl, "self")
	defer func() {
		_, err := Exec(context.Background(), db, `DROP TABLE bogus2`)
		require.NoError(t, err)
		_, err = Exec(context.Background(), db, `DROP TABLE bogus`)
		require.NoError(t, err)
		_, err = Exec(context.Background(), db, `DROP function doSomething`)
		require.NoError(t, err)
		require.NoError(t, db.Close())
	}()
	require.NoError(t, db.Ping(context.Background()))
	rowsAffected, err := Exec(context.Background(), db, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3)`, "a1", 1, true)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	rowsAffected, err = Exec(context.Background(), db, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3)`, "a1", 1, true)
	require.ErrorIs(t, err, ErrDuplicate)
	assert.EqualValues(t, terror.New(ErrDuplicate, map[string]any{"column": "pk"}), err)
	assert.True(t, IsErr(err, ErrDuplicate, "pk"))
	assert.EqualValues(t, 0, rowsAffected)
	rowsAffected, err = Exec(context.Background(), db, `INSERT INTO bogus2(a,b,c) VALUES ($1,$2,$3)`, "a1", 1, true)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	rowsAffected, err = Exec(context.Background(), db, `INSERT INTO bogus2(a,b,c) VALUES ($1,$2,$3)`, "a1", 2, true)
	require.ErrorIs(t, err, ErrDuplicate)
	assert.EqualValues(t, terror.New(ErrDuplicate, map[string]any{"column": "a"}), err)
	assert.True(t, IsErr(err, ErrDuplicate, "a"))
	assert.EqualValues(t, 0, rowsAffected)
	rowsAffected, err = Exec(context.Background(), db, `INSERT INTO bogus2(a,b,c) VALUES ($1,$2,$3)`, "a2", 1, true)
	require.ErrorIs(t, err, ErrDuplicate)
	assert.EqualValues(t, terror.New(ErrDuplicate, map[string]any{"column": "pk"}), err)
	assert.EqualValues(t, 0, rowsAffected)
	rowsAffected, err = Exec(context.Background(), db, `INSERT INTO bogus2(a,b,c) VALUES ($1,$2,$3)`, "axx", 33, true)
	require.ErrorIs(t, err, ErrRelationNotFound)
	assert.EqualValues(t, terror.New(ErrRelationNotFound, map[string]any{"column": "a"}), err)
	assert.True(t, IsErr(err, ErrRelationNotFound, "a"))
	assert.EqualValues(t, 0, rowsAffected)
	res1, err := ExecOne[Bogus](context.Background(), db, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3) RETURNING *`, "a2", 2, true)
	require.NoError(t, err)
	assert.EqualValues(t, &Bogus{A: "a2", B: 2, C: true}, res1)
	res2, err := ExecMany[Bogus](context.Background(), db, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3),($4,$5,$6) RETURNING *`, "a3", 3, true, "a4", 4, false)
	require.NoError(t, err)
	assert.EqualValues(t, []*Bogus{{A: "a3", B: 3, C: true}, {A: "a4", B: 4, C: false}}, res2)
	res3, err := Get[Bogus](context.Background(), db, `SELECT * FROM bogus WHERE a = $1`, "a1")
	require.NoError(t, err)
	assert.EqualValues(t, &Bogus{A: "a1", B: 1, C: true}, res3)
	resX, err := Get[Bogus](context.Background(), db, `SELECT * FROM bogus WHERE a = $1`, "axxx")
	require.ErrorIs(t, err, ErrNotFound)
	assert.True(t, IsErr(err, ErrNotFound))
	assert.Nil(t, resX)
	res4, err := Select[Bogus](context.Background(), db, `SELECT * FROM bogus WHERE a != $1  ORDER BY b`, "b")
	require.NoError(t, err)
	assert.EqualValues(t, []*Bogus{{A: "a1", B: 1, C: true}, {A: "a2", B: 2, C: true}, {A: "a3", B: 3, C: true}, {A: "a4", B: 4, C: false}}, res4)
	require.NoError(t, DoInTransaction(context.Background(), db, func(conn QueryExecer) error {
		rowsAffected, err = Exec(context.Background(), conn, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3)`, "a5", 5, true)
		require.NoError(t, err)
		if err != nil {
			return err
		}
		assert.EqualValues(t, 1, rowsAffected)
		res1, err = ExecOne[Bogus](context.Background(), conn, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3) RETURNING *`, "a6", 6, true)
		require.NoError(t, err)
		if err != nil {
			return err
		}
		assert.EqualValues(t, &Bogus{A: "a6", B: 6, C: true}, res1)
		res2, err = ExecMany[Bogus](context.Background(), conn, `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3),($4,$5,$6) RETURNING *`, "a7", 7, true, "a8", 8, false)
		require.NoError(t, err)
		if err != nil {
			return err
		}
		assert.EqualValues(t, []*Bogus{{A: "a7", B: 7, C: true}, {A: "a8", B: 8, C: false}}, res2)
		res3, err = Get[Bogus](context.Background(), conn, `SELECT * FROM bogus WHERE a = $1`, "a5")
		require.NoError(t, err)
		if err != nil {
			return err
		}
		assert.EqualValues(t, &Bogus{A: "a5", B: 5, C: true}, res3)
		res4, err = Select[Bogus](context.Background(), conn, `SELECT * FROM bogus WHERE a != $1  ORDER BY b`, "bb")
		require.NoError(t, err)
		if err != nil {
			return err
		}
		assert.EqualValues(t, []*Bogus{{A: "a1", B: 1, C: true}, {A: "a2", B: 2, C: true}, {A: "a3", B: 3, C: true}, {A: "a4", B: 4, C: false}, {A: "a5", B: 5, C: true}, {A: "a6", B: 6, C: true}, {A: "a7", B: 7, C: true}, {A: "a8", B: 8, C: false}}, res4) //nolint:lll // .

		return nil
	}))
}

func TestStopWhenTxAborted(t *testing.T) {
	t.Parallel()
	db := MustConnect(context.Background(), "", "self")
	require.NotNil(t, db)

	err := DoInTransaction(context.Background(), db, func(tx QueryExecer) error {
		_, gErr := Get[bool](context.Background(), tx, `SELECT $1 + $2`, 1, "2")

		return gErr
	})
	require.ErrorIs(t, err, ErrTxAborted)
	require.NoError(t, db.Close())
}
