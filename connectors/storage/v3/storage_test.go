// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/time"
)

type (
	xx struct {
		C *time.Time `redis:"cc"`
		A string     `redis:"-"`
		B int        `redis:"bb"`
	}
)

func (x *xx) Key() string {
	return x.A
}

func (x *xx) SetKey(key string) {
	x.A = key
}

func TestStorage(t *testing.T) {
	t.Parallel()
	db := MustConnect(context.Background(), "self")
	res, err := db.Del(context.Background(), "x1", "x2", "x3", "x4", "x5", "x6").Result()
	require.NoError(t, err)
	require.EqualValues(t, 0, res)
	now := time.Now()
	require.NoError(t, Set(context.Background(), db, &xx{A: "x1", B: 111, C: now}, &xx{A: "x2", B: 222}))
	require.NoError(t, Set(context.Background(), db, &xx{A: "x3", B: 333}))
	require.NoError(t, AtomicSet(context.Background(), db, &xx{A: "x4", B: 444}, &xx{A: "x5", B: 555}))
	require.NoError(t, AtomicSet(context.Background(), db, &xx{A: "x6", B: 666}))
	usr, err := Get[xx](context.Background(), db, "x1")
	require.NoError(t, err)
	require.EqualValues(t, []*xx{{A: "x1", B: 111, C: now}}, usr)
	usr, err = Get[xx](context.Background(), db, "x7")
	require.NoError(t, err)
	require.Nil(t, usr)
	usrs, err := Get[xx](context.Background(), db, "x1", "x2", "x3", "x4", "x5", "x7", "x6")
	require.NoError(t, err)
	require.EqualValues(t, []*xx{
		{A: "x1", B: 111, C: now},
		{A: "x2", B: 222, C: new(time.Time)},
		{A: "x3", B: 333, C: new(time.Time)},
		{A: "x4", B: 444, C: new(time.Time)},
		{A: "x5", B: 555, C: new(time.Time)},
		{A: "x6", B: 666, C: new(time.Time)},
	}, usrs)
	res, err = db.Del(context.Background(), "x1", "x2", "x3", "x4", "x5", "x6").Result()
	require.NoError(t, err)
	require.EqualValues(t, 6, res)
}
