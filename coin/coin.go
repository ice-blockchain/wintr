// SPDX-License-Identifier: BUSL-1.1

package coin

import (
	"strings"

	"cosmossdk.io/math"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/ice-blockchain/wintr/log"
)

func UnsafeNew(amount string) *Coin {
	coin, err := New(amount)
	log.Panic(err)

	return coin
}

func UnsafeNewAmount(amount string) *ICEFlake {
	f, err := NewAmount(amount)
	log.Panic(err)

	return f
}

func NewAmount(amount string) (*ICEFlake, error) {
	if amount == "" {
		return &ICEFlake{Uint: math.ZeroUint()}, nil
	}
	u, err := math.ParseUint(amount)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse 256Bit uint %v", amount)
	}

	return &ICEFlake{Uint: u}, nil
}

func NewUint64(amount uint64) *Coin {
	return new(Coin).SetAmount(NewAmountUint64(amount))
}

func NewAmountUint64(amount uint64) *ICEFlake {
	return &ICEFlake{Uint: math.NewUint(amount)}
}

func New(amount string) (*Coin, error) {
	a, err := NewAmount(amount)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build NewAmount for %v", amount)
	}

	return new(Coin).SetAmount(a), nil
}

func (c *Coin) SetAmount(amount *ICEFlake) *Coin {
	c.Amount = amount
	bits := c.Amount.BigInt().Bits()
	for i, w := range bits {
		c.setWord(i, uint64(w))
	}
	for i := len(bits); i < 1+1+1+1; i++ {
		c.setWord(i, 0)
	}

	return c
}

//nolint:gomnd // Those are fixed numbers, nothing magical about em.
func (c *Coin) setWord(wordPosition int, wordValue uint64) {
	switch wordPosition {
	case 0:
		c.AmountWord0 = wordValue
	case 1:
		c.AmountWord1 = wordValue
	case 2:
		c.AmountWord2 = wordValue
	case 3:
		c.AmountWord3 = wordValue
	}
}

//nolint:wrapcheck // Because we want to proxy the call.
func (i *ICEFlake) UnmarshalJSON(bytes []byte) error {
	if len(bytes) == 1+1 {
		return i.Uint.UnmarshalJSON([]byte(`"0"`))
	}

	return i.Uint.UnmarshalJSON(bytes)
}

func (i *ICEFlake) EncodeMsgpack(enc *msgpack.Encoder) error {
	bytes, err := i.Marshal()
	if err != nil {
		return errors.Wrapf(err, "cound not EncodeMsgpack->Marshall *ICEFlake %v", i.String())
	}
	v := string(bytes)

	return errors.Wrapf(enc.EncodeString(v), "coud not EncodeMsgpack->EncodeString *ICEFlake %v", v)
}

func (i *ICEFlake) DecodeMsgpack(dec *msgpack.Decoder) error {
	val, err := dec.DecodeString()
	if err != nil {
		return errors.Wrap(err, "could not DecodeMsgpack->DecodeString *ICEFlake")
	}
	if val == "" {
		val = "0"
	}

	return errors.Wrapf(i.Unmarshal([]byte(val)), "coud not DecodeMsgpack->Unmarshal *ICEFlake %v", val)
}

func (i *ICEFlake) ICE() (*ICE, error) {
	if i.IsZero() {
		ice := ICE(zero)

		return &ice, nil
	}
	bytes, err := i.Marshal()
	if err != nil {
		return nil, errors.Wrapf(err, "cound not EncodeMsgpack->Marshall *ICEFlake %v", i.String())
	}
	if i.GTE(denomination) {
		return i.transformNumbersBiggerThan1ICE(string(bytes))
	}
	r := "0."
	for i := 0; i < e9-len(bytes); i++ {
		r += "0"
	}
	ice := ICE(r + string(bytes))

	return &ice, nil
}

func (i *ICEFlake) transformNumbersBiggerThan1ICE(v string) (*ICE, error) {
	var ice ICE
	if i.Mod(denomination).IsZero() {
		ice = ICE(v[:len(v)-e9] + ".0")
	} else {
		ice = ICE(v[:len(v)-e9] + "." + v[len(v)-e9:])
	}

	return &ice, nil
}

func (i *ICEFlake) UnsafeICE() *ICE {
	ice, err := i.ICE()
	log.Panic(err)

	return ice
}

func (i *ICE) ICEFlake() (*ICEFlake, error) {
	val := strings.Trim(string(*i), " ")
	ix := strings.Index(val, ".")
	if ix < 0 {
		return NewAmount(val + e9Zeros)
	}
	missingZeros := e9 - len(val[ix+1:])
	for i := 0; i < missingZeros; i++ {
		val += "0"
	}
	r := val[:ix] + val[ix+1:]
	r = strings.TrimLeftFunc(r, func(r rune) bool { return r == '0' })

	return NewAmount(r)
}

func (i *ICE) UnsafeICEFlake() *ICEFlake {
	iceFlake, err := i.ICEFlake()
	log.Panic(err)

	return iceFlake
}

func (i *ICE) Format() string {
	val := strings.Trim(string(*i), " ")
	if val == "" {
		return zero
	}
	dotIx := strings.Index(val, ".")
	if dotIx == 0 {
		val = "0" + val
		dotIx = 1
	}
	if dotIx < 0 {
		dotIx = len(val)
	}
	s := formatGroups(dotIx, val)
	if s[0] == ',' {
		s = s[1:]
	}
	val = s + val[dotIx:]
	if !strings.Contains(val, ".") {
		val += ".0"
	}

	return val
}

func formatGroups(dotIx int, r string) string {
	var val, group string
	for j := dotIx - 1; j >= 0; j-- {
		group = string(r[j]) + group
		if len(group) == 3 { //nolint:gomnd // Its not a magic number, it's the number of elements in a group.
			val = "," + group + val
			group = ""
		}
	}
	if group != "" {
		val = group + val
	}

	return val
}

func (i *ICE) MarshalJSON() ([]byte, error) {
	if strings.Contains(string(*i), ",") {
		return []byte(`"` + *i + `"`), nil
	}

	return []byte(`"` + i.Format() + `"`), nil
}

func (i *ICE) UnmarshalJSON(bytes []byte) error {
	val := strings.ReplaceAll(string(bytes), ",", "")
	val = strings.ReplaceAll(val, `"`, "")
	val = strings.Trim(val, " ")
	_, err := NewAmount(toICEFlake(val))
	if err != nil {
		return errors.Wrapf(err, "invalid number: ice amount %v", string(bytes))
	}
	if val == "" {
		val = zero
	}
	if !strings.Contains(val, ".") {
		val += ".0"
	}
	if strings.Index(val, ".") == 0 {
		val = "0" + val
	}
	*i = ICE(val)

	return nil
}

func toICEFlake(iceValue string) string {
	ice := iceValue
	dotIdx := strings.Index(ice, ".")
	if dotIdx >= 0 {
		if dotIdx == 0 {
			ice = "0" + ice
			dotIdx++
		}
		zeros := ""
		for i := 0; i <= e9-(len(ice)-dotIdx); i++ {
			zeros += "0"
		}

		ice = ice[dotIdx+1:] + zeros
	} else {
		ice += e9Zeros
	}

	return strings.TrimLeftFunc(ice, func(r rune) bool { return r == '0' })
}
