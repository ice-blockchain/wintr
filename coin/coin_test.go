// SPDX-License-Identifier: ice License 1.0

package coin

import (
	"context"
	"math/big"
	"testing"

	"cosmossdk.io/math"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	maxUint64Word = big.Word(1<<64 - 1)
)

func TestICEFlakeAdd(t *testing.T) {
	t.Parallel()
	var nilVal *ICEFlake
	newVal := new(ICEFlake)
	zeroVal := UnsafeParseAmount("0")
	emptyVal := UnsafeParseAmount("")

	someValue := new(ICEFlake).set(math.NewUint(123456))
	a1 := UnsafeParseAmount("123")
	assert.EqualValues(t, UnsafeParseAmount("123579"), a1.Add(someValue))
	assert.EqualValues(t, UnsafeParseAmount("123579"), a1.Add(someValue))
	assert.EqualValues(t, UnsafeParseAmount("123579"), someValue.Add(a1))
	assert.EqualValues(t, UnsafeParseAmount("123579"), someValue.Add(a1))

	assert.EqualValues(t, someValue, zeroVal.Add(someValue))
	assert.EqualValues(t, someValue, emptyVal.Add(someValue))
	assert.EqualValues(t, someValue, newVal.Add(someValue))
	assert.EqualValues(t, someValue, nilVal.Add(someValue))

	assert.EqualValues(t, someValue, someValue.Add(zeroVal))
	assert.EqualValues(t, someValue, someValue.Add(emptyVal))
	assert.EqualValues(t, someValue, someValue.Add(newVal))
	assert.EqualValues(t, someValue, someValue.Add(nilVal))

	assert.EqualValues(t, zeroVal, zeroVal.Add(zeroVal))
	assert.EqualValues(t, emptyVal, emptyVal.Add(emptyVal))
	assert.EqualValues(t, newVal, newVal.Add(newVal))
	assert.EqualValues(t, nilVal, nilVal.Add(nilVal))
	assert.EqualValues(t, nilVal, nilVal.Add(zeroVal))
}

func TestICEFlakeSub(t *testing.T) {
	t.Parallel()
	var nilVal *ICEFlake
	newVal := new(ICEFlake)
	zeroVal := UnsafeParseAmount("0")
	emptyVal := UnsafeParseAmount("")

	someValue := new(ICEFlake).set(math.NewUint(123456))
	a1 := UnsafeParseAmount("123")
	assert.EqualValues(t, zeroVal, a1.Subtract(someValue))
	assert.EqualValues(t, zeroVal, a1.Subtract(someValue))
	assert.EqualValues(t, UnsafeParseAmount("123333"), someValue.Subtract(a1))
	assert.EqualValues(t, UnsafeParseAmount("123333"), someValue.Subtract(a1))
	assert.EqualValues(t, zeroVal, someValue.Subtract(someValue))

	assert.EqualValues(t, zeroVal, zeroVal.Subtract(someValue))
	assert.EqualValues(t, emptyVal, emptyVal.Subtract(someValue))
	assert.EqualValues(t, newVal, newVal.Subtract(someValue))
	assert.EqualValues(t, nilVal, nilVal.Subtract(someValue))

	assert.EqualValues(t, someValue, someValue.Subtract(zeroVal))
	assert.EqualValues(t, someValue, someValue.Subtract(emptyVal))
	assert.EqualValues(t, someValue, someValue.Subtract(newVal))
	assert.EqualValues(t, someValue, someValue.Subtract(nilVal))

	assert.EqualValues(t, zeroVal, zeroVal.Subtract(zeroVal))
	assert.EqualValues(t, emptyVal, emptyVal.Subtract(emptyVal))
	assert.EqualValues(t, newVal, newVal.Subtract(newVal))
	assert.EqualValues(t, nilVal, nilVal.Subtract(nilVal))
	assert.EqualValues(t, nilVal, nilVal.Subtract(zeroVal))
}

func TestCoinAdd(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	var nilVal *Coin
	newVal := new(Coin)
	zeroVal := UnsafeParse("0")
	emptyVal := UnsafeParse("")

	var nilValAmount *ICEFlake
	newValAmount := new(ICEFlake)
	zeroValAmount := UnsafeParseAmount("0")
	emptyValAmount := UnsafeParseAmount("")

	someValue := UnsafeParse("123456")
	someICEFlakeValue := UnsafeParseAmount("123456")
	a1 := UnsafeParse("123")
	a1ICEFlake := UnsafeParseAmount("123")
	assert.EqualValues(t, UnsafeParse("123579"), a1.Add(someICEFlakeValue))
	assert.EqualValues(t, UnsafeParse("123579"), a1.Add(someICEFlakeValue))
	assert.EqualValues(t, UnsafeParse("123579"), someValue.Add(a1ICEFlake))
	assert.EqualValues(t, UnsafeParse("123579"), someValue.Add(a1ICEFlake))

	assert.EqualValues(t, someValue, zeroVal.Add(someICEFlakeValue))
	assert.EqualValues(t, someValue, emptyVal.Add(someICEFlakeValue))
	assert.EqualValues(t, someValue, newVal.Add(someICEFlakeValue))
	assert.EqualValues(t, someValue, nilVal.Add(someICEFlakeValue))
	assert.True(t, newVal.IsNil())
	assert.True(t, nilVal.IsNil())

	assert.EqualValues(t, someValue, someValue.Add(zeroValAmount))
	assert.EqualValues(t, someValue, someValue.Add(emptyValAmount))
	assert.EqualValues(t, someValue, someValue.Add(newValAmount))
	assert.EqualValues(t, someValue, someValue.Add(nilValAmount))

	assert.EqualValues(t, zeroVal, zeroVal.Add(zeroValAmount))
	assert.EqualValues(t, emptyVal, emptyVal.Add(emptyValAmount))
	assert.EqualValues(t, newVal, newVal.Add(newValAmount))
	assert.EqualValues(t, nilVal, nilVal.Add(nilValAmount))
	assert.EqualValues(t, nilVal, nilVal.Add(zeroValAmount))
	assert.EqualValues(t, UnsafeParse("123579"), nilVal.setAmount(new(ICEFlake).set(math.NewUint(123579))))
	assert.EqualValues(t, UnsafeParse("0"), nilVal.setAmount(new(ICEFlake).set(math.NewUint(0))))
	assert.EqualValues(t, new(Coin), UnsafeParse("123579").setAmount(nil))
	assert.EqualValues(t, &Coin{Amount: &ICEFlake{}}, nilVal.setAmount(new(ICEFlake)))
	assert.EqualValues(t, new(Coin), nilVal.setAmount(nil))
}

func TestCoinSub(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	var nilVal *Coin
	newVal := new(Coin)
	zeroVal := UnsafeParse("0")
	emptyVal := UnsafeParse("")
	var nilValAmount *ICEFlake
	newValAmount := new(ICEFlake)
	zeroValAmount := UnsafeParseAmount("0")
	emptyValAmount := UnsafeParseAmount("")

	someICEFlakeValue := UnsafeParseAmount("123456")
	someValue := UnsafeParse("123456")
	a1 := UnsafeParse("123")
	a1ICEFlake := UnsafeParseAmount("123")
	assert.EqualValues(t, zeroVal, a1.Subtract(someICEFlakeValue))
	assert.EqualValues(t, zeroVal, a1.Subtract(someICEFlakeValue))
	assert.EqualValues(t, UnsafeParse("123333"), someValue.Subtract(a1ICEFlake))
	assert.EqualValues(t, UnsafeParse("123333"), someValue.Subtract(a1ICEFlake))
	assert.EqualValues(t, zeroVal, someValue.Subtract(someICEFlakeValue))

	assert.EqualValues(t, zeroVal, zeroVal.Subtract(someICEFlakeValue))
	assert.EqualValues(t, emptyVal, emptyVal.Subtract(someICEFlakeValue))
	assert.EqualValues(t, newVal, newVal.Subtract(someICEFlakeValue))
	assert.EqualValues(t, nilVal, nilVal.Subtract(someICEFlakeValue))

	assert.EqualValues(t, someValue, someValue.Subtract(zeroValAmount))
	assert.EqualValues(t, someValue, someValue.Subtract(emptyValAmount))
	assert.EqualValues(t, someValue, someValue.Subtract(newValAmount))
	assert.EqualValues(t, someValue, someValue.Subtract(nilValAmount))

	assert.EqualValues(t, zeroVal, zeroVal.Subtract(zeroValAmount))
	assert.EqualValues(t, emptyVal, emptyVal.Subtract(emptyValAmount))
	assert.EqualValues(t, newVal, newVal.Subtract(newValAmount))
	assert.EqualValues(t, nilVal, nilVal.Subtract(nilValAmount))
	assert.EqualValues(t, nilVal, nilVal.Subtract(zeroValAmount))
}

func TestIsZero(t *testing.T) {
	t.Parallel()
	assert.False(t, UnsafeParse("123").IsZero())
	assert.False(t, UnsafeParseAmount("123").IsZero())
	a1 := ICE("123")
	assert.False(t, (&a1).IsZero())

	var nilCoin *Coin
	var nilICEFlake *ICEFlake
	assert.True(t, UnsafeParse("0").IsZero())
	assert.True(t, UnsafeParse("").IsZero())
	assert.True(t, new(Coin).IsZero())
	assert.True(t, nilCoin.IsZero())
	assert.True(t, UnsafeParseAmount("0").IsZero())
	assert.True(t, UnsafeParseAmount("").IsZero())
	assert.True(t, new(ICEFlake).IsZero())
	assert.True(t, nilICEFlake.IsZero())
	a2 := ICE("")
	assert.True(t, (&a2).IsZero())
	a3 := ICE("0")
	assert.True(t, (&a3).IsZero())
	a4 := ICE("0.0")
	assert.True(t, (&a4).IsZero())
	var a5 *ICE
	assert.True(t, a5.IsZero())
}

func TestICEConversion(t *testing.T) {
	t.Parallel()

	a1 := UnsafeParseAmount("115792089237316195423570985008687907853269984665640564039457584007913129639935")
	a2 := UnsafeParseAmount("12000000000")
	a3 := UnsafeParseAmount("12000000001")
	a4 := UnsafeParseAmount("1000000000")
	a5 := UnsafeParseAmount("1000000001")
	a6 := UnsafeParseAmount("999999999")
	a7 := UnsafeParseAmount("55")
	a8 := UnsafeParseAmount("5")
	a9 := UnsafeParseAmount("0")
	a10 := UnsafeParseAmount("100000005")
	a11 := UnsafeParseAmount("1123123123010000000")

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
	assert.EqualValues(t, "1123123123.01", *a11.UnsafeICE())
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

	assert.EqualValues(t, UnsafeParseAmount("115792089237316195423570985008687907853269984665640564039457584007913129639935"), a1.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("12000000000"), a2.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("12000000001"), a3.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("1000000000"), a4.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("1000000001"), a5.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("999999999"), a6.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("55"), a7.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("5"), a8.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("0"), a9.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("0"), a10.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("123000000000"), a11.UnsafeICEFlake())
	assert.EqualValues(t, UnsafeParseAmount("100000005"), a12.UnsafeICEFlake())
	assert.Nil(t, a13.UnsafeICEFlake())
}

func TestICEJSONSerialization(t *testing.T) {
	t.Parallel()
	type whatever struct {
		ICE *ICE `json:"ice"`
	}
	s := ICE("1,123,123,123.01")
	we := whatever{ICE: &s}
	bytes, err := json.MarshalContext(context.Background(), we)
	require.NoError(t, err)
	assert.Equal(t, `{"ice":"1,123,123,123.01"}`, string(bytes))
	var w2 whatever
	require.NoError(t, json.UnmarshalContext(context.Background(), bytes, &w2))
	ice := ICE("1123123123.01")
	require.Equal(t, whatever{ICE: &ice}, w2)
	require.True(t, w2.ICE.UnsafeICEFlake().Equal(math.NewUint(1123123123010000000)))
	ice2 := ICE("1123123123.01")
	bytes, err = json.MarshalContext(context.Background(), whatever{ICE: &ice2})
	require.NoError(t, err)
	assert.Equal(t, `{"ice":"1,123,123,123.01"}`, string(bytes))
	bytes, err = json.MarshalContext(context.Background(), whatever{ICE: UnsafeParseAmount("1123123123010000000").UnsafeICE()})
	require.NoError(t, err)
	assert.Equal(t, `{"ice":"1,123,123,123.01"}`, string(bytes))
	bytes, err = json.MarshalContext(context.Background(), whatever{ICE: new(ICE)})
	require.NoError(t, err)
	assert.Equal(t, `{"ice":"0.0"}`, string(bytes))
	var w3 whatever
	require.NoError(t, json.UnmarshalContext(context.Background(), []byte(`{"ice":""}`), &w3))
	ice = "0.0"
	require.Equal(t, whatever{ICE: &ice}, w3)
	require.True(t, w3.ICE.UnsafeICEFlake().Equal(math.ZeroUint()))
}

func TestICEUnmarshalJSON(t *testing.T) {
	t.Parallel()
	ice1 := new(ICE)
	assert.NoError(t, ice1.UnmarshalJSON(context.Background(), []byte("")))
	ice2 := new(ICE)
	assert.NoError(t, ice2.UnmarshalJSON(context.Background(), []byte(" .0")))
	ice3 := new(ICE)
	assert.NoError(t, ice3.UnmarshalJSON(context.Background(), []byte("0")))
	ice4 := new(ICE)
	assert.NoError(t, ice4.UnmarshalJSON(context.Background(), []byte("1123123123.01")))
	assert.Equal(t, ICE("0.0"), *ice1)
	assert.Equal(t, ICE("0.0"), *ice2)
	assert.Equal(t, ICE("1123123123.01"), *ice4)
}

//nolint:funlen // Alot of usecases.
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
	a16 := ICE("1123123123.01")
	assert.Equal(t, "115,792,089,237,316,195,423,570,985,008,687,907,853,269,984,665,640,564,039,457,584,007,913.129639935", a1.Format())
	assert.Equal(t, "12.000000001", a3.Format())
	assert.Equal(t, "1.0", a4.Format())
	assert.Equal(t, "1.000000001", a5.Format())
	assert.Equal(t, "0.5", NewAmountUint64(500_000_000).UnsafeICE().Format())
	assert.Equal(t, "1,111.0", NewAmountUint64(1_111_000_000_000).UnsafeICE().Format())
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
	assert.Equal(t, "1,123,123,123.01", a16.Format())
}

func TestICEFlakeJSONSerialization(t *testing.T) {
	t.Parallel()

	c1 := UnsafeParse("115792089237316195423570985008687907853269984665640564039457584007913129639935")
	bytes, err := json.MarshalContext(context.Background(), c1)
	require.NoError(t, err)
	assert.Equal(t, `{"amount":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}`, string(bytes))
	var c2 Coin
	require.NoError(t, json.UnmarshalContext(context.Background(), bytes, &c2))
	assert.Equal(t, UnsafeParseAmount("115792089237316195423570985008687907853269984665640564039457584007913129639935"), c2.Amount)
	bytes, err = json.MarshalContext(context.Background(), Coin{Amount: &ICEFlake{}})
	require.NoError(t, err)
	assert.Equal(t, `{"amount":"0"}`, string(bytes))
	var c3 Coin
	require.NoError(t, json.UnmarshalContext(context.Background(), []byte(`{"amount":""}`), &c3))
	assert.Equal(t, NewAmountUint64(0), c3.Amount)
}

type (
	tmpStruct struct {
		//nolint:unused,revive,tagliatelle,nosnakecase // It is used by db to marshall/unmarshall.
		_msgpack struct{} `msgpack:",asArray"`
		*Coin
	}
)

func TestICEFlakeMsgPackSerialization(t *testing.T) {
	t.Parallel()
	c1 := tmpStruct{Coin: UnsafeParse("115792089237316195423570985008687907853269984665640564039457584007913129639935")}
	bytes, err := msgpack.Marshal(c1)
	require.NoError(t, err)
	assert.Equal(t, []byte{
		0x95, 0xd9, 0x4e, 0x31, 0x31, 0x35, 0x37, 0x39, 0x32, 0x30, 0x38, 0x39, 0x32, 0x33, 0x37, 0x33, 0x31, 0x36, 0x31, 0x39, 0x35, 0x34,
		0x32, 0x33, 0x35, 0x37, 0x30, 0x39, 0x38, 0x35, 0x30, 0x30, 0x38, 0x36, 0x38, 0x37, 0x39, 0x30, 0x37, 0x38, 0x35, 0x33, 0x32, 0x36, 0x39, 0x39, 0x38,
		0x34, 0x36, 0x36, 0x35, 0x36, 0x34, 0x30, 0x35, 0x36, 0x34, 0x30, 0x33, 0x39, 0x34, 0x35, 0x37, 0x35, 0x38, 0x34, 0x30, 0x30, 0x37, 0x39, 0x31, 0x33,
		0x31, 0x32, 0x39, 0x36, 0x33, 0x39, 0x39, 0x33, 0x35, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}, bytes)
	var c2 tmpStruct
	require.NoError(t, msgpack.Unmarshal(bytes, &c2))
	assert.Equal(t, UnsafeParseAmount("115792089237316195423570985008687907853269984665640564039457584007913129639935"), c2.Amount)
	assert.Equal(t, AmountWords{uint64(maxUint64Word), uint64(maxUint64Word), uint64(maxUint64Word), uint64(maxUint64Word)}, c2.AmountWords)
	c3 := tmpStruct{Coin: &Coin{Amount: &ICEFlake{}}}
	bytes, err = msgpack.Marshal(c3)
	require.NoError(t, err)
	assert.Equal(t, []byte{
		0x95, 0xa1, 0x30, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	}, bytes)
	var c4 tmpStruct
	require.NoError(t, msgpack.Unmarshal(bytes, &c4))
	assert.True(t, c4.Amount.IsZero())
	assert.Equal(t, AmountWords{}, c4.AmountWords)
}

type (
	tmpStruct2 struct {
		//nolint:unused,revive,tagliatelle,nosnakecase // It is used by db to marshall/unmarshall.
		_msgpack struct{} `msgpack:",asArray"`
		Amount   *ICE
		AmountWords
	}
)

func TestICEMsgPackSerialization(t *testing.T) {
	t.Parallel()
	var c2 tmpStruct2
	require.NoError(t, msgpack.Unmarshal([]byte{
		0x95, 0xd9, 0x4e, 0x31, 0x31, 0x35, 0x37, 0x39, 0x32, 0x30, 0x38, 0x39, 0x32, 0x33, 0x37, 0x33, 0x31, 0x36, 0x31, 0x39, 0x35, 0x34,
		0x32, 0x33, 0x35, 0x37, 0x30, 0x39, 0x38, 0x35, 0x30, 0x30, 0x38, 0x36, 0x38, 0x37, 0x39, 0x30, 0x37, 0x38, 0x35, 0x33, 0x32, 0x36, 0x39, 0x39, 0x38,
		0x34, 0x36, 0x36, 0x35, 0x36, 0x34, 0x30, 0x35, 0x36, 0x34, 0x30, 0x33, 0x39, 0x34, 0x35, 0x37, 0x35, 0x38, 0x34, 0x30, 0x30, 0x37, 0x39, 0x31, 0x33,
		0x31, 0x32, 0x39, 0x36, 0x33, 0x39, 0x39, 0x33, 0x35, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}, &c2))
	ice1 := ICE("115792089237316195423570985008687907853269984665640564039457584007913.129639935")
	assert.Equal(t, &ice1, c2.Amount)
	var c4 tmpStruct2
	require.NoError(t, msgpack.Unmarshal([]byte{
		0x95, 0xa1, 0x30, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0xcf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	}, &c4))
	assert.True(t, c4.Amount.IsZero())
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
	coin := new(Coin)
	coin.verifySetAmount(t,
		"115792089237316195423570985008687907853269984665640564039457584007913129639935",
		maxUint64Word, maxUint64Word, maxUint64Word, maxUint64Word)
	coin.verifySetAmount(t,
		"115792089237316195423570985008687907853269984665640564039457584007913129639934",
		maxUint64Word-1, maxUint64Word, maxUint64Word, maxUint64Word)
	coin.verifySetAmount(t, "1", 1, 0, 0, 0)
	coin.verifySetAmount(t, "6277101735386680763835789423207666416102355444464034512896", 0, 0, 0, 1)
	coin.verifySetAmount(t, "18446744073709551616", 0, 1, 0, 0)
	coin.verifySetAmount(t, "36893488147419103232", 0, 2, 0, 0)
	coin.verifySetAmount(t, "0", 0, 0, 0, 0)
	coin.verifySetAmount(t, "340282366920938463463374607431768211456", 0, 0, 1, 0)
	coin.verifySetAmount(t, "340282366920938463463374607431768211455", maxUint64Word, maxUint64Word, 0, 0)
	coin.verifySetAmount(t, "340282366920938463463374607431768211454", maxUint64Word-1, maxUint64Word, 0, 0)
	coin.verifySetAmount(t, "6277101735386680763835789423207666410000000000000000000000", 3516843933827072000, 18446744073709551285, maxUint64Word, 0)
}

func (c *Coin) verifySetAmount(t *testing.T, amount string, expectedWords ...big.Word) {
	t.Helper()

	c.setAmount(UnsafeParseAmount(amount))
	assert.Equal(t, UnsafeParseAmount(amount), c.Amount)
	dummy := new(Coin)
	for i, w := range expectedWords {
		dummy.setWord(i, uint64(w))
	}
	assert.Equal(t, dummy.AmountWords, c.AmountWords)
}
