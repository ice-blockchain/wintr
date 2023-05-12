// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/time"
)

type (
	xKey struct {
		A  string `redis:"-"`
		ZZ int    `redis:"zz"`
	}
	xx struct {
		C *time.Time `redis:"cc"`
		xKey
		B int `redis:"bb"`
	}
)

func (x *xKey) Key() string {
	return x.A
}

func (x *xKey) SetKey(key string) {
	x.A = key
}

func TestStorage(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	db := MustConnect(context.Background(), "self")
	result, eerr := db.FlushAll(context.Background()).Result()
	assert.NoError(t, eerr)
	assert.Equal(t, "OK", result)
	res, err := db.Del(context.Background(), "x1", "x2", "x3", "x4", "x5", "x6").Result()
	require.NoError(t, err)
	require.EqualValues(t, 0, res)
	now := time.Now()
	require.NoError(t, Set(context.Background(), db, &xx{xKey: xKey{A: "x1"}, B: 111, C: now}, &xx{xKey: xKey{A: "x2"}, B: 222}))
	require.NoError(t, Set(context.Background(), db, &xx{xKey: xKey{A: "x3"}, B: 333}))
	require.NoError(t, AtomicSet(context.Background(), db, &xx{xKey: xKey{A: "x4"}, B: 444}, &xx{xKey: xKey{A: "x5"}, B: 555}))
	require.NoError(t, AtomicSet(context.Background(), db, &xx{xKey: xKey{A: "x6"}, B: 666}))
	usr, err := Get[xx](context.Background(), db, "x1")
	require.NoError(t, err)
	require.EqualValues(t, []*xx{{xKey: xKey{A: "x1"}, B: 111, C: now}}, usr)
	usr, err = Get[xx](context.Background(), db, "x7")
	require.NoError(t, err)
	require.Nil(t, usr)
	usrs, err := Get[xx](context.Background(), db, "x1", "x2", "x3", "x4", "x5", "x7", "x6")
	require.NoError(t, err)
	require.EqualValues(t, []*xx{
		{xKey: xKey{A: "x1"}, B: 111, C: now},
		{xKey: xKey{A: "x2"}, B: 222, C: new(time.Time)},
		{xKey: xKey{A: "x3"}, B: 333, C: new(time.Time)},
		{xKey: xKey{A: "x4"}, B: 444, C: new(time.Time)},
		{xKey: xKey{A: "x5"}, B: 555, C: new(time.Time)},
		{xKey: xKey{A: "x6"}, B: 666, C: new(time.Time)},
	}, usrs)
	usrs = usrs[:0]
	require.NoError(t, Bind[xx](context.Background(), db, []string{"x1", "x2", "x3", "x4", "x5", "x7", "x6"}, ProcessRedisFieldTags[xx](), &usrs))
	require.EqualValues(t, []*xx{
		{xKey: xKey{A: "x1"}, B: 111, C: now},
		{xKey: xKey{A: "x2"}, B: 222, C: new(time.Time)},
		{xKey: xKey{A: "x3"}, B: 333, C: new(time.Time)},
		{xKey: xKey{A: "x4"}, B: 444, C: new(time.Time)},
		{xKey: xKey{A: "x5"}, B: 555, C: new(time.Time)},
		{xKey: xKey{A: "x6"}, B: 666, C: new(time.Time)},
	}, usrs)
	require.NoError(t, Set(context.Background(), db, &xx{xKey: xKey{A: "x1"}, B: 111111}))
	usr, err = Get[xx](context.Background(), db, "x1")
	require.NoError(t, err)
	require.True(t, usr[0].C.IsNil())
	require.True(t, usr[0].B == 111111)
	res, err = db.Del(context.Background(), "x1", "x2", "x3", "x4", "x5", "x6").Result()
	require.NoError(t, err)
	require.EqualValues(t, 6, res)
}
