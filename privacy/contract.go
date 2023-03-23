// SPDX-License-Identifier: ice License 1.0

package privacy

import (
	"crypto/cipher"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
)

// Public API.

type (
	EncryptDecrypter interface {
		Encrypt(string) string
		Decrypt(string) (string, error)
	}
	Sensitive   string
	DBSensitive string
)

// Private API.

// .
var (
	errHexDecodingFailed = errors.New("failed to hex decode value")
	errDecryptionFailed  = errors.New("failed to decrypt value")
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	ed EncryptDecrypter
	_  msgpack.CustomEncoder   = (*DBSensitive)(nil)
	_  msgpack.CustomDecoder   = (*DBSensitive)(nil)
	_  msgpack.CustomEncoder   = (*Sensitive)(nil)
	_  msgpack.CustomDecoder   = (*Sensitive)(nil)
	_  json.UnmarshalerContext = (*Sensitive)(nil)
	_  json.MarshalerContext   = (*Sensitive)(nil)
)

type (
	encryptDecrypter struct {
		AES256GCMSIVCipher cipher.AEAD
		Nonce              []byte
	}
	config struct {
		Secret string `yaml:"secret" mapstructure:"secret"`
	}
)
