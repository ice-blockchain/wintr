// SPDX-License-Identifier: BUSL-1.1

package coin

import (
	"math/big"
	"testing"

	"cosmossdk.io/math"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

const maxUint64Word = big.Word(1<<64 - 1)

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
	a13 := ICE("")

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
	assert.True(t, a13.UnsafeICEFlake().Equal(UnsafeNewAmount("").Uint))
	assert.True(t, a13.UnsafeICEFlake().Equal(math.ZeroUint()))
}

func TestICEJSONSerialization(t *testing.T) {
	t.Parallel()
	type whatever struct {
		ICE *ICE `json:"ice"`
	}
	s := ICE("1,123,123,123.01")
	w := whatever{ICE: &s}
	b, err := json.Marshal(w)
	require.NoError(t, err)
	assert.Equal(t, `{"ice":"1,123,123,123.01"}`, string(b))
	var w2 whatever
	require.NoError(t, json.Unmarshal(b, &w2))
	ice := ICE("1123123123.01")
	require.Equal(t, whatever{ICE: &ice}, w2)
	require.True(t, w2.ICE.UnsafeICEFlake().Equal(math.NewUint(1123123123010000000)))
	ice2 := ICE("1123123123.01")
	b, err = json.Marshal(whatever{ICE: &ice2})
	require.NoError(t, err)
	assert.Equal(t, `{"ice":"1,123,123,123.01"}`, string(b))
	w = whatever{ICE: new(ICE)}
	b, err = json.Marshal(w)
	require.NoError(t, err)
	assert.Equal(t, `{"ice":"0.0"}`, string(b))
	var w3 whatever
	require.NoError(t, json.Unmarshal([]byte(`{"ice":""}`), &w3))
	ice = "0.0"
	require.Equal(t, whatever{ICE: &ice}, w3)
	require.True(t, w3.ICE.UnsafeICEFlake().Equal(math.ZeroUint()))
}

func TestICEUnmarshalJSON(t *testing.T) {
	t.Parallel()
	i := new(ICE)
	assert.NoError(t, i.UnmarshalJSON([]byte("")))
	i2 := new(ICE)
	assert.NoError(t, i2.UnmarshalJSON([]byte(" .0")))
	i3 := new(ICE)
	assert.NoError(t, i3.UnmarshalJSON([]byte("0")))
	assert.Equal(t, ICE("0.0"), *i)
	assert.Equal(t, ICE("0.0"), *i2)
	assert.Equal(t, ICE("0.0"), *i3)
}

func TestICEFormat(t *testing.T) {
	t.Parallel()

	a1 := ICE("115792089237316195423570985008687907853269984665640564039457584007913.129639935")
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
	a15 := ICE("")
	assert.Equal(t, "115,792,089,237,316,195,423,570,985,008,687,907,853,269,984,665,640,564,039,457,584,007,913.129639935", a1.Format())
	assert.Equal(t, "12.000000001", a3.Format())
	assert.Equal(t, "1.0", a4.Format())
	assert.Equal(t, "1.000000001", a5.Format())
	assert.Equal(t, "0.999999999", a6.Format())
	assert.Equal(t, "0.0", a7.Format())
	assert.Equal(t, "0.1", a8.Format())
	assert.Equal(t, "7,913.129639935", a9.Format())
	assert.Equal(t, "913.129639935", a10.Format())
	assert.Equal(t, "13.129639935", a11.Format())
	assert.Equal(t, "7,913.0", a12.Format())
	assert.Equal(t, "913.0", a13.Format())
	assert.Equal(t, "13.0", a14.Format())
	assert.Equal(t, "0.0", a15.Format())
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
	b, err = json.Marshal(Coin{Amount: &ICEFlake{}})
	require.NoError(t, err)
	assert.Equal(t, `{"amount":"0"}`, string(b))
	var c3 Coin
	require.NoError(t, json.Unmarshal([]byte(`{"amount":""}`), &c3))
	assert.Equal(t, NewAmountUint64(0), c3.Amount)
}

type tmpStruct struct {
	//nolint:unused // It is used by db to marshall/unmarshall.
	_msgpack struct{} `msgpack:",asArray"`
	*Coin
}

func TestICEFlakeMsgPackSerialization(t *testing.T) {
	t.Parallel()
	c1 := tmpStruct{Coin: UnsafeNew("115792089237316195423570985008687907853269984665640564039457584007913129639935")}
	b, err := msgpack.Marshal(c1)
	require.NoError(t, err)
	assert.Equal(t, []byte{
		0x95, 0xd9, 0x4e, 0x31, 0x31, 0x35, 0x37, 0x39, 0x32, 0x30, 0x38, 0x39, 0x32, 0x33, 0x37, 0x33, 0x31, 0x36, 0x31, 0x39, 0x35, 0x34,
		0x32, 0x33, 0x35, 0x37, 0x30, 0x39, 0x38, 0x35, 0x30, 0x30, 0x38, 0x36, 0x38, 0x37, 0x39, 0x30, 0x37, 0x38, 0x35, 0x33, 0x32, 0x36, 0x39, 0x39, 0x38,
		0x34, 0x36, 0x36, 0x35, 0x36, 0x34, 0x30, 0x35, 0x36, 0x34, 0x30, 0x33, 0x39, 0x34, 0x35, 0x37, 0x35, 0x38, 0x34, 0x30, 0x30, 0x37, 0x39, 0x31, 0x33,
		0x31, 0x32, 0x39, 0x36, 0x33, 0x39, 0x39, 0x33, 0x35, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}, b)
	var c2 tmpStruct
	require.NoError(t, msgpack.Unmarshal(b, &c2))
	assert.Equal(t, UnsafeNewAmount("115792089237316195423570985008687907853269984665640564039457584007913129639935"), c2.Amount)
	assert.Equal(t, AmountWords{uint64(maxUint64Word), uint64(maxUint64Word), uint64(maxUint64Word), uint64(maxUint64Word)}, c2.AmountWords)
	c3 := tmpStruct{Coin: &Coin{Amount: &ICEFlake{}}}
	b, err = msgpack.Marshal(c3)
	require.NoError(t, err)
	assert.Equal(t, []byte{
		0x95, 0xa1, 0x30, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	}, b)
	var c4 tmpStruct
	require.NoError(t, msgpack.Unmarshal(b, &c4))
	assert.True(t, c4.Amount.IsZero())
	assert.Equal(t, AmountWords{}, c4.AmountWords)
}

func TestToICEFlake(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", toICEFlake(""))
	assert.Equal(t, "", toICEFlake("0"))
	assert.Equal(t, "", toICEFlake("0.0"))
	assert.Equal(t, "10000000", toICEFlake("0.010000"))
	assert.Equal(t, "10000000", toICEFlake(".01"))
	assert.Equal(t, "123000000000", toICEFlake("123"))
	assert.Equal(t, "231230000", toICEFlake("0.23123"))
	assert.Equal(t, "999999999", toICEFlake("0.999999999"))
}

func TestCoinSetAmount(t *testing.T) {
	t.Parallel()
	c := new(Coin)
	c.verifySetAmount(t,
		"115792089237316195423570985008687907853269984665640564039457584007913129639935",
		maxUint64Word, maxUint64Word, maxUint64Word, maxUint64Word)
	c.verifySetAmount(t,
		"115792089237316195423570985008687907853269984665640564039457584007913129639934",
		maxUint64Word-1, maxUint64Word, maxUint64Word, maxUint64Word)
	c.verifySetAmount(t, "1", 1, 0, 0, 0)
	c.verifySetAmount(t, "6277101735386680763835789423207666416102355444464034512896", 0, 0, 0, 1)
	c.verifySetAmount(t, "18446744073709551616", 0, 1, 0, 0)
	c.verifySetAmount(t, "0", 0, 0, 0, 0)
	c.verifySetAmount(t, "340282366920938463463374607431768211456", 0, 0, 1, 0)
	c.verifySetAmount(t, "340282366920938463463374607431768211455", maxUint64Word, maxUint64Word, 0, 0)
	c.verifySetAmount(t, "340282366920938463463374607431768211454", maxUint64Word-1, maxUint64Word, 0, 0)
	c.verifySetAmount(t, "6277101735386680763835789423207666410000000000000000000000", 3516843933827072000, 18446744073709551285, maxUint64Word, 0)
}

func (c *Coin) verifySetAmount(t *testing.T, amount string, expectedWords ...big.Word) {
	t.Helper()

	c.SetAmount(UnsafeNewAmount(amount))
	assert.Equal(t, UnsafeNewAmount(amount), c.Amount)
	dummy := new(Coin)
	for i, w := range expectedWords {
		dummy.setWord(i, uint64(w))
	}
	assert.Equal(t, dummy.AmountWords, c.AmountWords)
}
