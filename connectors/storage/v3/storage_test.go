// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/time"
)

type (
	XKey struct {
		A  string  `redis:"-"`
		ZZ float64 `redis:"zz"`
	}
	EmbeddedC struct {
		C *time.Time `redis:"cc,omitempty"`
	}
	xx struct {
		*EmbeddedC
		XKey
		B int `redis:"bb"`
	}
)

func (x *XKey) Key() string {
	return x.A
}

func (x *XKey) SetKey(key string) {
	x.A = key
}

//nolint:funlen // .
func TestStorage(t *testing.T) {
	t.Parallel()
	db := MustConnect(t.Context(), "self")
	result, err := db.Ping(t.Context()).Result()
	require.NoError(t, err)
	assert.Equal(t, "PONG", result)
	assert.True(t, db.IsRW(t.Context()))
	result, eerr := db.FlushAll(t.Context()).Result()
	require.NoError(t, eerr)
	assert.Equal(t, "OK", result)
	res, err := db.Del(t.Context(), "x1", "x2", "x3", "x4", "x5", "x6").Result()
	require.NoError(t, err)
	require.EqualValues(t, 0, res)
	now := time.Now()
	require.NoError(t, Set(t.Context(), db, &xx{XKey: XKey{A: "x1", ZZ: 999.234}, B: 111, EmbeddedC: &EmbeddedC{C: now}}, &xx{XKey: XKey{A: "x2"}, B: 222}))
	require.NoError(t, Set(t.Context(), db, &xx{XKey: XKey{A: "x3"}, B: 333}))
	require.NoError(t, Set(t.Context(), db, &xx{XKey: XKey{A: "x4"}, B: 444}, &xx{XKey: XKey{A: "x5"}, B: 555}))
	require.NoError(t, Set(t.Context(), db, &xx{XKey: XKey{A: "x6"}, B: 666}))
	usr, err := Get[xx](t.Context(), db, "x1")
	require.NoError(t, err)
	require.EqualValues(t, []*xx{{XKey: XKey{A: "x1", ZZ: 999.234}, B: 111, EmbeddedC: &EmbeddedC{C: now}}}, usr)
	usr, err = Get[xx](t.Context(), db, "x7")
	require.NoError(t, err)
	require.Nil(t, usr)
	usrs, err := Get[xx](t.Context(), db, "x1", "x2", "x3", "x4", "x5", "x7", "x6")
	require.NoError(t, err)
	require.EqualValues(t, []*xx{
		{XKey: XKey{A: "x1", ZZ: 999.234}, B: 111, EmbeddedC: &EmbeddedC{C: now}},
		{XKey: XKey{A: "x2"}, B: 222, EmbeddedC: &EmbeddedC{}},
		{XKey: XKey{A: "x3"}, B: 333, EmbeddedC: &EmbeddedC{}},
		{XKey: XKey{A: "x4"}, B: 444, EmbeddedC: &EmbeddedC{}},
		{XKey: XKey{A: "x5"}, B: 555, EmbeddedC: &EmbeddedC{}},
		{XKey: XKey{A: "x6"}, B: 666, EmbeddedC: &EmbeddedC{}},
	}, usrs)
	usrs = usrs[:0]
	require.NoError(t, Bind[xx](t.Context(), db, []string{"x1", "x2", "x3", "x4", "x5", "x7", "x6"}, &usrs))
	require.EqualValues(t, []*xx{
		{XKey: XKey{A: "x1", ZZ: 999.234}, B: 111, EmbeddedC: &EmbeddedC{C: now}},
		{XKey: XKey{A: "x2"}, B: 222, EmbeddedC: &EmbeddedC{}},
		{XKey: XKey{A: "x3"}, B: 333, EmbeddedC: &EmbeddedC{}},
		{XKey: XKey{A: "x4"}, B: 444, EmbeddedC: &EmbeddedC{}},
		{XKey: XKey{A: "x5"}, B: 555, EmbeddedC: &EmbeddedC{}},
		{XKey: XKey{A: "x6"}, B: 666, EmbeddedC: &EmbeddedC{}},
	}, usrs)
	require.NoError(t, Set(t.Context(), db, &xx{XKey: XKey{A: "x1"}, B: 111111}))
	usr, err = Get[xx](t.Context(), db, "x1")
	require.NoError(t, err)
	require.EqualValues(t, []*xx{{XKey: XKey{A: "x1"}, B: 111111, EmbeddedC: &EmbeddedC{C: now}}}, usr)
	res, err = db.Del(t.Context(), "x1", "x2", "x3", "x4", "x5", "x6").Result()
	require.NoError(t, err)
	require.EqualValues(t, 6, res)
	require.NoError(t, db.Close())
}

func BenchmarkSerializeValue(b *testing.B) {
	value := &xx{XKey: XKey{A: "x1", ZZ: 999.234}, B: 111, EmbeddedC: &EmbeddedC{C: time.Now()}}

	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if resp := SerializeValue(value); len(resp) != 6 {
				panic("it should return 6 elements")
			}
		}
	})
}

func BenchmarkDeserializeValue(b *testing.B) {
	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			type (
				XXX struct {
					*EmbeddedC
					XKey
					B int `redis:"bb"`
				}
			)
			var xxx struct {
				XXX
			}
			scans := 0
			if err := DeserializeValue(&xxx, func(_ any) error {
				scans++

				return nil
			}); err != nil || scans != 4 {
				panic(err)
			}
		}
	})
}
