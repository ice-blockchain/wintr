// SPDX-License-Identifier: ice License 1.0

package privacy

import (
	"context"
	"encoding/hex"

	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/ice-blockchain/wintr/log"
)

func (s *Sensitive) Bind(val string) *Sensitive {
	*s = Sensitive(val)

	return s
}

func (s *DBSensitive) Bind(val string) *DBSensitive {
	*s = DBSensitive(val)

	return s
}

func (s *Sensitive) MarshalJSON(_ context.Context) ([]byte, error) {
	if s == nil {
		return []byte(`null`), nil
	}
	if *s == "" {
		return []byte(`""`), nil
	}
	if _, err := hex.DecodeString(string(*s)); err == nil {
		return []byte(`"` + string(*s) + `"`), nil
	}

	return []byte(`"` + Encrypt(string(*s)) + `"`), nil
}

func (s *Sensitive) UnmarshalJSON(_ context.Context, bytes []byte) error {
	val := string(bytes)
	if val == "null" || val == `""` || val == "" {
		return nil
	}

	return s.decrypt(string(bytes[1 : len(bytes)-1]))
}

func (s *DBSensitive) DecodeMsgpack(decoder *msgpack.Decoder) error {
	val, err := decoder.DecodeString()
	if err != nil {
		return errors.Wrap(err, "failed to decode value as string")
	}
	if val == "" {
		return nil
	}

	return s.decrypt(val)
}

func (s *DBSensitive) decrypt(val string) error {
	decrypted, err := Decrypt(val)
	if err != nil {
		if errors.Is(err, errHexDecodingFailed) || errors.Is(err, errDecryptionFailed) {
			*s = DBSensitive(val)
			if errors.Is(err, errDecryptionFailed) {
				log.Error(err)
			}

			return nil
		}

		return errors.Wrap(err, "failed to decrypt value")
	}
	*s = DBSensitive(decrypted)

	return nil
}

func (s *Sensitive) DecodeMsgpack(decoder *msgpack.Decoder) error {
	val, err := decoder.DecodeString()
	if err != nil {
		return errors.Wrap(err, "failed to decode value as string")
	}
	if val == "" {
		return nil
	}

	return s.decrypt(val)
}

func (s *Sensitive) decrypt(val string) error {
	decrypted, err := Decrypt(val)
	if err != nil {
		if errors.Is(err, errHexDecodingFailed) || errors.Is(err, errDecryptionFailed) {
			*s = Sensitive(val)
			if errors.Is(err, errDecryptionFailed) {
				log.Error(err)
			}

			return nil
		}

		return errors.Wrap(err, "failed to decrypt value")
	}
	*s = Sensitive(decrypted)

	return nil
}

func (s *DBSensitive) EncodeMsgpack(encoder *msgpack.Encoder) error {
	if s == nil || *s == "" {
		return errors.Wrap(encoder.EncodeNil(), "failed to encode to nil")
	}
	if _, err := hex.DecodeString(string(*s)); err == nil {
		return errors.Wrap(encoder.EncodeString(string(*s)), "failed to encode as plain string")
	}

	return errors.Wrap(encoder.EncodeString(Encrypt(string(*s))), "failed to encode as encrypted string")
}

func (s *Sensitive) EncodeMsgpack(encoder *msgpack.Encoder) error {
	if s == nil || *s == "" {
		return errors.Wrap(encoder.EncodeNil(), "failed to encode to nil")
	}
	if _, err := hex.DecodeString(string(*s)); err == nil {
		return errors.Wrap(encoder.EncodeString(string(*s)), "failed to encode as plain string")
	}

	return errors.Wrap(encoder.EncodeString(Encrypt(string(*s))), "failed to encode as encrypted string")
}

func (s *Sensitive) String() string {
	if s == nil {
		return ""
	}

	return string(*s)
}

func (s *DBSensitive) String() string {
	if s == nil {
		return ""
	}

	return string(*s)
}
