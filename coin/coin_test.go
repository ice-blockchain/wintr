// SPDX-License-Identifier: BUSL-1.1

package coin

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

func TestICEConversion(t *testing.T) {
	t.Parallel()

	a1 := UnsafeNewAmount("115792089237316195423570985008687907853269984665640564039457584007913129639935")
	a2 := UnsafeNewAmount("12000000000")
	a3 := UnsafeNewAmount("12000000001")
	a4 := UnsafeNewAmount("1000000000")
	a5 := UnsafeNewAmount("1000000001")
	a6 := UnsafeNewAmount("999999999")
	a7 := UnsafeNewAmount("55")
	a8 := UnsafeNewAmount("5")
	a9 := UnsafeNewAmount("0")
	a10 := UnsafeNewAmount("100000005")

	assert.EqualValues(t, "115792089237316195423570985008687907853269984665640564039457584007913.129639935", *a1.UnsafeICE())
	assert.EqualValues(t, "12.0", *a2.UnsafeICE())
	assert.EqualValues(t, "12.000000001", *a3.UnsafeICE())
	assert.EqualValues(t, "1.0", *a4.UnsafeICE())
	assert.EqualValues(t, "1.000000001", *a5.UnsafeICE())
	assert.EqualValues(t, "0.999999999", *a6.UnsafeICE())
	assert.EqualValues(t, "0.000000055", *a7.UnsafeICE())
	assert.EqualValues(t, "0.000000005", *a8.UnsafeICE())
	assert.EqualValues(t, "0.0", *a9.UnsafeICE())
	assert.EqualValues(t, "0.100000005", *a10.UnsafeICE())
}

func TestICEFlakeConversion(t *testing.T) {
	t.Parallel()

	a1 := ICE("115792089237316195423570985008687907853269984665640564039457584007913.129639935")
	a2 := ICE("12.0")
	a3 := ICE("12.000000001")
	a4 := ICE("1.0")
	a5 := ICE("1.000000001")
	a6 := ICE("0.999999999")
	a7 := ICE("0.000000055")
	a8 := ICE("0.000000005")
	a9 := ICE("0.0")
	a10 := ICE(".0")
	a11 := ICE("123")
	a12 := ICE("0.100000005")

	assert.True(t, a1.UnsafeICEFlake().Equal(UnsafeNewAmount("115792089237316195423570985008687907853269984665640564039457584007913129639935").Uint))
	assert.True(t, a2.UnsafeICEFlake().Equal(UnsafeNewAmount("12000000000").Uint))
	assert.True(t, a3.UnsafeICEFlake().Equal(UnsafeNewAmount("12000000001").Uint))
	assert.True(t, a4.UnsafeICEFlake().Equal(UnsafeNewAmount("1000000000").Uint))
	assert.True(t, a5.UnsafeICEFlake().Equal(UnsafeNewAmount("1000000001").Uint))
	assert.True(t, a6.UnsafeICEFlake().Equal(UnsafeNewAmount("999999999").Uint))
	assert.True(t, a7.UnsafeICEFlake().Equal(UnsafeNewAmount("55").Uint))
	assert.True(t, a8.UnsafeICEFlake().Equal(UnsafeNewAmount("5").Uint))
	assert.True(t, a9.UnsafeICEFlake().Equal(UnsafeNewAmount("0").Uint))
	assert.True(t, a10.UnsafeICEFlake().Equal(UnsafeNewAmount("0").Uint))
	assert.True(t, a11.UnsafeICEFlake().Equal(UnsafeNewAmount("123000000000").Uint))
	assert.True(t, a12.UnsafeICEFlake().Equal(UnsafeNewAmount("100000005").Uint))
}

func TestICEJSONSerialization(t *testing.T) {
	t.Parallel()
	type whatever struct {
		ICE ICE `json:"ice"`
	}
	w := whatever{
		ICE: "1,123,123,123.01",
	}
	b, err := json.Marshal(w)
	require.NoError(t, err)
	assert.Equal(t, `{"ice":"1,123,123,123.01"}`, string(b))
	var w2 whatever
	require.NoError(t, json.Unmarshal(b, &w2))
	require.Equal(t, whatever{ICE: "1123123123.01"}, w2)
	require.True(t, w2.ICE.UnsafeICEFlake().Equal(math.NewUint(1123123123010000000)))
}

func TestICEFormat(t *testing.T) {
	t.Parallel()

	a1 := ICE("115792089237316195423570985008687907853269984665640564039457584007913.129639935")
	a2 := ICE("12.0")
	a3 := ICE("12.000000001")
	a4 := ICE("1.0")
	a5 := ICE("1.000000001")
	a6 := ICE("0.999999999")
	a7 := ICE("0.0")
	a8 := ICE(".1")
	a9 := ICE("7913.129639935")
	a10 := ICE("913.129639935")
	a11 := ICE("13.129639935")
	a12 := ICE("7913")
	a13 := ICE("913")
	a14 := ICE("13")
	assert.Equal(t, "115,792,089,237,316,195,423,570,985,008,687,907,853,269,984,665,640,564,039,457,584,007,913.129639935", a1.Format())
	assert.Equal(t, "12.0", a2.Format())
	assert.Equal(t, "12.000000001", a3.Format())
	assert.Equal(t, "1.0", a4.Format())
	assert.Equal(t, "1.000000001", a5.Format())
	assert.Equal(t, "0.999999999", a6.Format())
	assert.Equal(t, "0.0", a7.Format())
	assert.Equal(t, "0.1", a8.Format())
	assert.Equal(t, "7,913.129639935", a9.Format())
	assert.Equal(t, "913.129639935", a10.Format())
	assert.Equal(t, "13.129639935", a11.Format())
	assert.Equal(t, "7,913", a12.Format())
	assert.Equal(t, "913", a13.Format())
	assert.Equal(t, "13", a14.Format())
}

func TestICEFlakeJSONSerialization(t *testing.T) {
	t.Parallel()

	c1 := UnsafeNew("115792089237316195423570985008687907853269984665640564039457584007913129639935")
	b, err := json.Marshal(c1)
	require.NoError(t, err)
	assert.Equal(t, `{"amount":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}`, string(b))
	var c2 Coin
	require.NoError(t, json.Unmarshal(b, &c2))
	assert.Equal(t, UnsafeNewAmount("115792089237316195423570985008687907853269984665640564039457584007913129639935"), c2.Amount)
}

func TestICEFlakeMsgPackSerialization(t *testing.T) {
	t.Parallel()

	type tmpStruct struct {
		//nolint:unused // It is used by db to marshall/unmarshall.
		_msgpack struct{} `msgpack:",asArray"`
		*Coin
	}
	c1 := tmpStruct{
		Coin: UnsafeNew("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
	}
	b, err := msgpack.Marshal(c1)
	//nolint:lll // .
	assert.Equal(t, []byte{0x91, 0xd9, 0x4e, 0x31, 0x31, 0x35, 0x37, 0x39, 0x32, 0x30, 0x38, 0x39, 0x32, 0x33, 0x37, 0x33, 0x31, 0x36, 0x31, 0x39, 0x35, 0x34, 0x32, 0x33, 0x35, 0x37, 0x30, 0x39, 0x38, 0x35, 0x30, 0x30, 0x38, 0x36, 0x38, 0x37, 0x39, 0x30, 0x37, 0x38, 0x35, 0x33, 0x32, 0x36, 0x39, 0x39, 0x38, 0x34, 0x36, 0x36, 0x35, 0x36, 0x34, 0x30, 0x35, 0x36, 0x34, 0x30, 0x33, 0x39, 0x34, 0x35, 0x37, 0x35, 0x38, 0x34, 0x30, 0x30, 0x37, 0x39, 0x31, 0x33, 0x31, 0x32, 0x39, 0x36, 0x33, 0x39, 0x39, 0x33, 0x35}, b)
	require.NoError(t, err)
	var c2 tmpStruct
	require.NoError(t, msgpack.Unmarshal(b, &c2))
	assert.Equal(t, UnsafeNewAmount("115792089237316195423570985008687907853269984665640564039457584007913129639935"), c2.Amount)
}
