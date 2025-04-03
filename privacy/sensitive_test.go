// SPDX-License-Identifier: ice License 1.0

package privacy

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

type (
	dummy[T ~string | *Sensitive | *DBSensitive] struct {
		A T `json:"a,omitempty"`
	}
)

func TestSensitiveJSONMarshalUnmarshal(t *testing.T) { //nolint:dupl // .
	t.Parallel()
	val := dummy[*Sensitive]{A: new(Sensitive).Bind("bogus@foo.com")}
	bytes, err := json.MarshalContext(t.Context(), val)
	require.NoError(t, err)
	assert.Equal(t, `{"a":"e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc"}`, string(bytes))
	bytes, err = json.MarshalContext(t.Context(), dummy[*Sensitive]{A: new(Sensitive).Bind("e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc")})
	assert.Equal(t, `{"a":"e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc"}`, string(bytes))
	require.NoError(t, err)
	var resp dummy[*Sensitive]
	require.NoError(t, json.UnmarshalContext(t.Context(), bytes, &resp))
	assert.EqualValues(t, val, resp)
	var resp2 dummy[*Sensitive]
	require.NoError(t, json.UnmarshalContext(t.Context(), []byte(`{"a":"bogus@foo.com"}`), &resp2))
	assert.EqualValues(t, val, resp2)
	val = dummy[*Sensitive]{}
	bytes, err = json.MarshalContext(t.Context(), val)
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(bytes))
	val = dummy[*Sensitive]{A: new(Sensitive)}
	bytes, err = json.MarshalContext(t.Context(), val)
	require.NoError(t, err)
	assert.Equal(t, `{"a":""}`, string(bytes))
	var resp3 dummy[*Sensitive]
	require.NoError(t, json.UnmarshalContext(t.Context(), []byte(`{}`), &resp3))
	assert.EqualValues(t, dummy[*Sensitive]{}, resp3)
	var resp4 dummy[*Sensitive]
	require.NoError(t, json.UnmarshalContext(t.Context(), []byte(`{"a":null}`), &resp4))
	assert.EqualValues(t, dummy[*Sensitive]{}, resp4)
}

func TestDBSensitiveJSONMarshalUnmarshal(t *testing.T) { //nolint:dupl // .
	t.Parallel()
	val := dummy[*DBSensitive]{A: new(DBSensitive).Bind("bogus@foo.com")}
	bytes, err := json.MarshalContext(t.Context(), val)
	require.NoError(t, err)
	assert.Equal(t, `{"a":"bogus@foo.com"}`, string(bytes))
	bytes, err = json.MarshalContext(t.Context(), dummy[*DBSensitive]{A: new(DBSensitive).Bind("bogus@foo.com")})
	assert.Equal(t, `{"a":"bogus@foo.com"}`, string(bytes))
	require.NoError(t, err)
	var resp dummy[*DBSensitive]
	require.NoError(t, json.UnmarshalContext(t.Context(), bytes, &resp))
	assert.EqualValues(t, val, resp)
	var resp2 dummy[*DBSensitive]
	require.NoError(t, json.UnmarshalContext(t.Context(), []byte(`{"a":"bogus@foo.com"}`), &resp2))
	assert.EqualValues(t, val, resp2)
	val = dummy[*DBSensitive]{}
	bytes, err = json.MarshalContext(t.Context(), val)
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(bytes))
	val = dummy[*DBSensitive]{A: new(DBSensitive)}
	bytes, err = json.MarshalContext(t.Context(), val)
	require.NoError(t, err)
	assert.Equal(t, `{"a":""}`, string(bytes))
	var resp3 dummy[*DBSensitive]
	require.NoError(t, json.UnmarshalContext(t.Context(), []byte(`{}`), &resp3))
	assert.EqualValues(t, dummy[*DBSensitive]{}, resp3)
	var resp4 dummy[*DBSensitive]
	require.NoError(t, json.UnmarshalContext(t.Context(), []byte(`{"a":null}`), &resp4))
	assert.EqualValues(t, dummy[*DBSensitive]{}, resp4)
}

func TestSensitiveMsgpackMarshalUnmarshal(t *testing.T) { //nolint:dupl // .
	t.Parallel()
	val := dummy[*Sensitive]{A: new(Sensitive).Bind("bogus@foo.com")}
	bytes, err := msgpack.Marshal(val)
	require.NoError(t, err)
	assert.Equal(t, "\x81\xa1A\xd9:e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc", string(bytes))
	bytes, err = msgpack.Marshal(dummy[*Sensitive]{A: new(Sensitive).Bind("e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc")})
	require.NoError(t, err)
	assert.Equal(t, "\x81\xa1A\xd9:e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc", string(bytes))
	var resp dummy[*Sensitive]
	require.NoError(t, msgpack.Unmarshal(bytes, &resp))
	assert.EqualValues(t, val, resp)
	bytes, err = msgpack.Marshal(dummy[string]{A: "bogus@foo.com"})
	require.NoError(t, err)
	var resp2 dummy[*Sensitive]
	require.NoError(t, msgpack.Unmarshal(bytes, &resp2))
	assert.EqualValues(t, val, resp2)
	val = dummy[*Sensitive]{}
	bytes, err = msgpack.Marshal(val)
	require.NoError(t, err)
	assert.Equal(t, "\x81\xa1A\xc0", string(bytes))
	val = dummy[*Sensitive]{A: new(Sensitive)}
	bytes, err = msgpack.Marshal(val)
	require.NoError(t, err)
	assert.Equal(t, "\x81\xa1A\xc0", string(bytes))
	var resp3 dummy[*Sensitive]
	require.NoError(t, msgpack.Unmarshal([]byte("\x81\xa1A\xc0"), &resp3))
	assert.EqualValues(t, dummy[*Sensitive]{}, resp3)
	var resp4 dummy[*Sensitive]
	require.NoError(t, msgpack.Unmarshal([]byte("\x81\xa1A\xa0"), &resp4))
	assert.EqualValues(t, dummy[*Sensitive]{A: new(Sensitive)}, resp4)
}

func TestDBSensitiveMsgpackMarshalUnmarshal(t *testing.T) { //nolint:dupl // .
	t.Parallel()
	val := dummy[*DBSensitive]{A: new(DBSensitive).Bind("bogus@foo.com")}
	bytes, err := msgpack.Marshal(val)
	require.NoError(t, err)
	assert.Equal(t, "\x81\xa1A\xd9:e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc", string(bytes))
	bytes, err = msgpack.Marshal(dummy[*DBSensitive]{A: new(DBSensitive).Bind("e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc")})
	require.NoError(t, err)
	assert.Equal(t, "\x81\xa1A\xd9:e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc", string(bytes))
	var resp dummy[*DBSensitive]
	require.NoError(t, msgpack.Unmarshal(bytes, &resp))
	assert.EqualValues(t, val, resp)
	bytes, err = msgpack.Marshal(dummy[string]{A: "bogus@foo.com"})
	require.NoError(t, err)
	var resp2 dummy[*DBSensitive]
	require.NoError(t, msgpack.Unmarshal(bytes, &resp2))
	assert.EqualValues(t, val, resp2)
	val = dummy[*DBSensitive]{}
	bytes, err = msgpack.Marshal(val)
	require.NoError(t, err)
	assert.Equal(t, "\x81\xa1A\xc0", string(bytes))
	val = dummy[*DBSensitive]{A: new(DBSensitive)}
	bytes, err = msgpack.Marshal(val)
	require.NoError(t, err)
	assert.Equal(t, "\x81\xa1A\xc0", string(bytes))
	var resp3 dummy[*DBSensitive]
	require.NoError(t, msgpack.Unmarshal([]byte("\x81\xa1A\xc0"), &resp3))
	assert.EqualValues(t, dummy[*DBSensitive]{}, resp3)
	var resp4 dummy[*DBSensitive]
	require.NoError(t, msgpack.Unmarshal([]byte("\x81\xa1A\xa0"), &resp4))
	assert.EqualValues(t, dummy[*DBSensitive]{A: new(DBSensitive)}, resp4)
}
