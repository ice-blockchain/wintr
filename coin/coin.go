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
	u, err := math.ParseUint(amount)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse 256Bit uint %v", amount)
	}

	return &ICEFlake{Uint: u}, nil
}

func NewUint64(amount uint64) *Coin {
	return &Coin{Amount: NewAmountUint64(amount)}
}

func NewAmountUint64(amount uint64) *ICEFlake {
	return &ICEFlake{Uint: math.NewUint(amount)}
}

func New(amount string) (*Coin, error) {
	a, err := NewAmount(amount)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build NewAmount for %v", amount)
	}

	return &Coin{Amount: a}, nil
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
	v, err := dec.DecodeString()
	if err != nil {
		return errors.Wrap(err, "could not DecodeMsgpack->DecodeString *ICEFlake")
	}

	return errors.Wrapf(i.Unmarshal([]byte(v)), "coud not DecodeMsgpack->Unmarshal *ICEFlake %v", v)
}

func (i *ICEFlake) ICE() (*ICE, error) {
	if i.IsZero() {
		ice := ICE("0.0")

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
	v := strings.Trim(string(*i), " ")
	ix := strings.Index(v, ".")
	if ix < 0 {
		zeros := ""
		for i := 0; i < e9; i++ {
			zeros += "0"
		}

		return NewAmount(v + zeros)
	}
	missingZeros := e9 - len(v[ix+1:])
	for i := 0; i < missingZeros; i++ {
		v += "0"
	}
	r := v[:ix] + v[ix+1:]
	r = strings.TrimLeftFunc(r, func(r rune) bool { return r == '0' })
	if r == "" {
		r = "0"
	}

	return NewAmount(r)
}

func (i *ICE) UnsafeICEFlake() *ICEFlake {
	iceFlake, err := i.ICEFlake()
	log.Panic(err)

	return iceFlake
}

func (i *ICE) Format() string {
	r := string(*i)
	dotIx := strings.Index(r, ".")
	if dotIx == 0 {
		r = "0" + r
		dotIx = 1
	}
	if dotIx < 0 {
		dotIx = len(r)
	}
	var s, g string
	for j := dotIx - 1; j >= 0; j-- {
		g = string(r[j]) + g
		//nolint:gomnd // Its not a magic number, it's the number of elements in a group.
		if len(g) == 3 {
			s = "," + g + s
			g = ""
		}
	}
	if g != "" {
		s = g + s
	}
	if s[0] == ',' {
		s = s[1:]
	}
	r = s + r[dotIx:]

	return r
}

func (i *ICE) MarshalJSON() ([]byte, error) {
	return []byte(i.Format()), nil
}

func (i *ICE) UnmarshalJSON(bytes []byte) error {
	r := strings.ReplaceAll(string(bytes), ",", "")
	r = strings.ReplaceAll(r, `"`, "")
	_, err := NewAmount(strings.Replace(r, ".", "", 1))
	if err != nil {
		return errors.Wrapf(err, "invalid number: ice amount %v", string(bytes))
	}
	*i = ICE(r)

	return nil
}
