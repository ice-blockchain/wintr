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

func New(amount string) (*Coin, error) {
	a, err := NewAmount(amount)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build NewAmount for %v", amount)
	}

	return &Coin{Amount: a}, nil
}

func (s *ICEFlake) EncodeMsgpack(enc *msgpack.Encoder) error {
	bytes, err := s.Marshal()
	if err != nil {
		return errors.Wrapf(err, "cound not EncodeMsgpack->Marshall *ICEFlake %v", s.String())
	}
	v := string(bytes)

	return errors.Wrapf(enc.EncodeString(v), "coud not EncodeMsgpack->EncodeString *ICEFlake %v", v)
}

func (s *ICEFlake) DecodeMsgpack(dec *msgpack.Decoder) error {
	v, err := dec.DecodeString()
	if err != nil {
		return errors.Wrap(err, "could not DecodeMsgpack->DecodeString *ICEFlake")
	}

	return errors.Wrapf(s.Unmarshal([]byte(v)), "coud not DecodeMsgpack->Unmarshal *ICEFlake %v", v)
}

func (s *ICEFlake) ICE() (ICE, error) {
	if s.IsZero() {
		return "0.0", nil
	}
	bytes, err := s.Marshal()
	if err != nil {
		return "", errors.Wrapf(err, "cound not EncodeMsgpack->Marshall *ICEFlake %v", s.String())
	}
	v := string(bytes)
	if s.GTE(denomination) {
		if s.Mod(denomination).IsZero() {
			return ICE(v[:len(v)-e9] + ".0"), nil
		}

		return ICE(v[:len(v)-e9] + "." + v[len(v)-e9:]), nil
	}
	r := "0."
	for i := 0; i < e9-len(v); i++ {
		r += "0"
	}
	r += v

	return ICE(r), nil
}

func (s *ICEFlake) UnsafeICE() ICE {
	ice, err := s.ICE()
	log.Panic(err)

	return ice
}

func (s ICE) ICEFlake() (*ICEFlake, error) {
	v := strings.Trim(string(s), " ")
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

func (s ICE) UnsafeICEFlake() *ICEFlake {
	iceFlake, err := s.ICEFlake()
	log.Panic(err)

	return iceFlake
}
