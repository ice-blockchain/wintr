// SPDX-License-Identifier: BUSL-1.1

package time

import (
	"encoding/json"
	"testing"
	stdlibtime "time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

//nolint:funlen // It's better to keep it together.
func TestTime(t *testing.T) {
	t.Parallel()
	type tmpStruct struct {
		CreatedAt *Time `json:"createdAt"`
	}
	time1, err := stdlibtime.Parse(stdlibtime.RFC3339Nano, "2006-01-02T15:04:05.999999999Z")
	require.NoError(t, err)
	t1 := tmpStruct{CreatedAt: New(time1)}
	bytes, err := json.Marshal(t1)
	require.NoError(t, err)
	assert.Equal(t, `{"createdAt":"2006-01-02T15:04:05.999999999Z"}`, string(bytes))
	bytes, err = msgpack.Marshal(t1)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x81, 0xa9, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0xcf, 0xf, 0xc4, 0xa4, 0xd6, 0x39, 0x91, 0x7b, 0xff}, bytes)
	var t11 tmpStruct
	require.NoError(t, msgpack.Unmarshal(bytes, &t11))
	assert.Equal(t, t1, t11)
	var t2 tmpStruct
	require.NoError(t, json.Unmarshal([]byte(`{"createdAt":1}`), &t2))
	assert.Equal(t, tmpStruct{CreatedAt: New(stdlibtime.Unix(0, 1).UTC())}, t2)
	bytes, err = json.Marshal(&tmpStruct{CreatedAt: New(stdlibtime.Unix(0, 0).UTC())})
	require.NoError(t, err)
	assert.Equal(t, `{"createdAt":null}`, string(bytes))
	var t21 tmpStruct
	require.NoError(t, json.Unmarshal([]byte(`{"createdAt":1655303440552373000}`), &t21))
	assert.Equal(t, tmpStruct{CreatedAt: New(stdlibtime.Unix(0, 1655303440552373000).UTC())}, t21)
	var t22 tmpStruct
	require.NoError(t, json.Unmarshal([]byte(`{"createdAt":1655303440552}`), &t22))
	assert.Equal(t, tmpStruct{CreatedAt: New(stdlibtime.Unix(0, 1655303440552000000).UTC())}, t22)
	var t3 tmpStruct
	require.NoError(t, json.Unmarshal([]byte(`{"createdAt":"2006-01-02T15:04:05.999999999Z"}`), &t3))
	assert.Equal(t, t1, t3)
	bytes, err = json.Marshal(tmpStruct{CreatedAt: Now()})
	require.NoError(t, err)
	assert.Regexp(t, `{"createdAt":".+"}`, string(bytes))
}
