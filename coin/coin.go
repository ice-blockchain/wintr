// SPDX-License-Identifier: ice License 1.0

package coin

import (
	"context"
	"strings"

	"cosmossdk.io/math"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/ice-blockchain/wintr/log"
)

func UnsafeParse(amount string) *Coin {
	coin, err := Parse(amount)
	log.Panic(err)

	return coin
}

func New(amount *ICEFlake) *Coin {
	if amount.IsNil() {
		return nil
	}

	return new(Coin).setAmount(amount)
}

func UnsafeParseAmount(amount string) *ICEFlake {
	f, err := ParseAmount(amount)
	log.Panic(err)

	return f
}

func ParseAmount(amount string) (*ICEFlake, error) {
	if amount == "" {
		return nil, nil //nolint:nilnil // Intended.
	}
	if amount == "0" {
		return ZeroICEFlakes(), nil
	}
	u, err := math.ParseUint(amount)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse 256Bit uint %v", amount)
	}

	return &ICEFlake{Uint: u}, nil
}

func NewUint64(amount uint64) *Coin {
	return new(Coin).setAmount(NewAmountUint64(amount))
}

func NewAmountUint64(amount uint64) *ICEFlake {
	return &ICEFlake{Uint: math.NewUint(amount)}
}

func Parse(amount string) (*Coin, error) {
	am, err := ParseAmount(amount)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build NewAmount for %v", amount)
	}
	if am.IsNil() {
		return nil, nil //nolint:nilnil // Intended.
	}

	return new(Coin).setAmount(am), nil
}

func (c *Coin) setAmount(amount *ICEFlake) *Coin {
	if c == nil {
		return new(Coin).setAmount(amount)
	}
	c.Amount = amount
	if c.Amount.IsNil() {
		c.AmountWords = AmountWords{}

		return c
	}
	bits := c.Amount.BigInt().Bits()
	for i, w := range bits {
		c.setWord(i, uint64(w))
	}
	for i := len(bits); i < 1+1+1+1; i++ {
		c.setWord(i, 0)
	}

	return c
}

func ZeroICEFlakes() *ICEFlake {
	return &ICEFlake{Uint: math.ZeroUint()}
}

func ZeroCoins() *Coin {
	return new(Coin).setAmount(ZeroICEFlakes())
}

func (c *Coin) Add(amount *ICEFlake) *Coin {
	if amount.IsZero() {
		return c
	}
	if c.IsZero() {
		return new(Coin).setAmount(amount)
	}

	return new(Coin).setAmount(c.Amount.Add(amount))
}

func (c *Coin) Subtract(amount *ICEFlake) *Coin {
	if amount.IsZero() || c.IsZero() {
		return c
	}

	return new(Coin).setAmount(c.Amount.Subtract(amount))
}

func (c *Coin) IsZero() bool {
	return c.IsNil() || c.Amount.IsZero()
}

func (c *Coin) IsNil() bool {
	return c == nil || c.Amount.IsNil()
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

func (i *ICEFlake) Add(amount *ICEFlake) *ICEFlake {
	if amount.IsZero() {
		return i
	}
	if i.IsZero() {
		return amount
	}

	return new(ICEFlake).set(i.Uint.Add(amount.Uint))
}

func (i *ICEFlake) Subtract(amount *ICEFlake) *ICEFlake {
	if amount.IsZero() || i.IsZero() {
		return i
	}

	if i.BigInt().Cmp(amount.BigInt()) <= 0 {
		return ZeroICEFlakes()
	}

	return new(ICEFlake).set(i.Uint.Sub(amount.Uint))
}

func (i *ICEFlake) Divide(amount *ICEFlake) *ICEFlake {
	if i.IsNil() {
		return nil
	}
	if i.IsZero() || amount.IsZero() {
		return i
	}

	return new(ICEFlake).set(i.Uint.Quo(amount.Uint))
}

func (i *ICEFlake) DivideUint64(amount uint64) *ICEFlake {
	if i.IsNil() {
		return nil
	}
	if amount == 0 || i.IsZero() {
		return i
	}

	return new(ICEFlake).set(i.Uint.QuoUint64(amount))
}

func (i *ICEFlake) MultiplyUint64(amount uint64) *ICEFlake {
	if i.IsNil() {
		return nil
	}
	if i.IsZero() {
		return i
	}
	if amount == 0 {
		return ZeroICEFlakes()
	}

	return new(ICEFlake).set(i.Uint.MulUint64(amount))
}

func (i *ICEFlake) Multiply(amount *ICEFlake) *ICEFlake {
	if i.IsNil() {
		return nil
	}
	if amount.IsNil() || i.IsZero() {
		return i
	}
	if amount.IsZero() {
		return amount
	}

	return new(ICEFlake).set(i.Uint.Mul(amount.Uint))
}

func (i *ICEFlake) IsNil() bool {
	return i == nil || i.Uint == (math.Uint{})
}

func (i *ICEFlake) set(amount math.Uint) *ICEFlake {
	i.Uint = amount

	return i
}

func (i *ICEFlake) IsZero() bool {
	return i.IsNil() || i.Uint.IsZero()
}

func (i *ICEFlake) MarshalJSON(_ context.Context) ([]byte, error) {
	return i.Uint.MarshalJSON() //nolint:wrapcheck // Because we want to proxy the call.
}

//nolint:wrapcheck // Because we want to proxy the call.
func (i *ICEFlake) UnmarshalJSON(_ context.Context, bytes []byte) error {
	if len(bytes) == 1+1 {
		return i.Uint.UnmarshalJSON([]byte(`"0"`))
	}

	return i.Uint.UnmarshalJSON(bytes)
}

func (i *ICEFlake) EncodeMsgpack(enc *msgpack.Encoder) error {
	bytes, err := i.Uint.Marshal()
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
	bytes, err := i.Uint.Marshal()
	if err != nil {
		return nil, errors.Wrapf(err, "cound not EncodeMsgpack->Marshall *ICEFlake %v", i.String())
	}
	if i.GTE(denomination) {
		return i.transformNumbersBiggerThan1ICE(string(bytes))
	}
	r := "0."
	for ix := 0; ix < e9-len(bytes); ix++ {
		r += "0"
	}
	ice := ICE(r + strings.TrimRight(string(bytes), "0"))

	return &ice, nil
}

func (i *ICEFlake) transformNumbersBiggerThan1ICE(val string) (*ICE, error) {
	var ice ICE
	if i.Mod(denomination).IsZero() {
		ice = ICE(val[:len(val)-e9] + ".0")
	} else {
		digits := []rune(val[len(val)-e9:])
		rest := make([]rune, 0, e9)
		var hasNonZeroTrailing bool
		for digitIx := len(digits) - 1; digitIx >= 0; digitIx-- {
			if digits[digitIx] != 48 { //nolint:gomnd // Not magical at all, that's 0 in ASCII.
				hasNonZeroTrailing = true
			}
			if hasNonZeroTrailing {
				rest = append(rest, digits[digitIx])
			}
		}
		for ii, jj := 0, len(rest)-1; ii < jj; ii, jj = ii+1, jj-1 {
			rest[ii], rest[jj] = rest[jj], rest[ii]
		}
		ice = ICE(val[:len(val)-e9] + "." + string(rest))
	}

	return &ice, nil
}

func (i *ICEFlake) UnsafeICE() *ICE {
	ice, err := i.ICE()
	log.Panic(err)

	return ice
}

func (i *ICE) DecodeMsgpack(decoder *msgpack.Decoder) error {
	iceFlakes := new(ICEFlake)
	if err := iceFlakes.DecodeMsgpack(decoder); err != nil {
		return errors.Wrapf(err, "failed to DecodeMsgpack as ICEFlake")
	}
	if ice, err := iceFlakes.ICE(); err != nil {
		return errors.Wrapf(err, "failed to convert iceFlakes to ice")
	} else { //nolint:revive // Nope, it's better this way.
		*i = *ice

		return nil
	}
}

func (i *ICE) IsZero() bool {
	return i == nil || *i == "" || *i == "0" || *i == zero
}

func (i *ICE) ICEFlake() (*ICEFlake, error) {
	val := strings.Trim(string(*i), " ")
	if val == "" {
		return nil, nil //nolint:nilnil // Intended.
	}
	ix := strings.Index(val, ".")
	if ix < 0 {
		return ParseAmount(val + e9Zeros)
	}
	missingZeros := e9 - len(val[ix+1:])
	for j := 0; j < missingZeros; j++ {
		val += "0"
	}
	res := val[:ix] + val[ix+1:]
	res = strings.TrimLeftFunc(res, func(r rune) bool {
		return r == '0'
	})
	if res == "" && (strings.Contains(string(*i), ".") || strings.Contains(string(*i), "0")) {
		return ZeroICEFlakes(), nil
	}

	return ParseAmount(res)
}

func (i *ICE) UnsafeICEFlake() *ICEFlake {
	iceFlake, err := i.ICEFlake()
	log.Panic(err)

	return iceFlake
}

func (i *ICE) String() string {
	return i.Format()
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

func (i *ICE) MarshalJSON(_ context.Context) ([]byte, error) {
	if strings.Contains(string(*i), ",") {
		return []byte(`"` + *i + `"`), nil
	}

	return []byte(`"` + i.Format() + `"`), nil
}

func (i *ICE) UnmarshalJSON(_ context.Context, bytes []byte) error {
	val := strings.ReplaceAll(string(bytes), ",", "")
	val = strings.ReplaceAll(val, `"`, "")
	val = strings.Trim(val, " ")
	_, err := ParseAmount(toICEFlake(val))
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

	return strings.TrimLeftFunc(ice, func(r rune) bool {
		return r == '0'
	})
}
