// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustConnect(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	ddl := `
create table if not exists bogus 
(
    a  text not null,
    b  integer not null check (b >= 0),
    c  boolean not null default false,
    primary key(a, b, c)
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
	db := MustConnect(context.Background(), ddl, "self")
	aa, bb, cc := "a3", 1, true
	a1, b1, c1 := "", 0, false
	err := db.Primary().QueryRow(context.Background(), `INSERT INTO bogus(a,b,c) VALUES ($1,$2,$3) RETURNING *`, aa, bb, cc).Scan(&a1, &b1, &c1)
	assert.Nil(t, err)
	assert.Equal(t, aa, a1)
	assert.Equal(t, bb, b1)
	assert.Equal(t, cc, c1)
	a1, b1, c1 = "", 0, false
	err = db.Replica().QueryRow(context.Background(), `SELECT * FROM bogus WHERE a = $1`, aa).Scan(&a1, &b1, &c1)
	assert.Nil(t, err)
	assert.Equal(t, aa, a1)
	assert.Equal(t, bb, b1)
	assert.Equal(t, cc, c1)
}
