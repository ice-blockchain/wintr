// SPDX-License-Identifier: ice License 1.0

package privacy

import (
	"encoding/hex"

	"github.com/ericlagergren/siv"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func init() { //nolint:gochecknoinits // We're initializing a global, default one.
	var cfg config
	appCfg.MustLoadFromKey("wintr/privacy", &cfg)

	ed = NewEncryptDecrypter(cfg.Secret)
}

func NewEncryptDecrypter(secret string) EncryptDecrypter {
	decodedKey, err := hex.DecodeString(secret)
	log.Panic(errors.Wrap(err, "failed to decode key value")) //nolint:revive // That's exactly what we want.
	if len(decodedKey) != 32+siv.NonceSize {
		log.Panic("we need 32+siv.NonceSize bytes key")
	}
	aes256gcmsiv, err := siv.NewGCM(decodedKey[siv.NonceSize:])
	log.Panic(errors.Wrap(err, "failed to build aes gcm siv mode"))

	return &encryptDecrypter{
		AES256GCMSIVCipher: aes256gcmsiv,
		Nonce:              decodedKey[:siv.NonceSize],
	}
}

func Encrypt(plaintext string) string {
	return ed.Encrypt(plaintext)
}

func (e *encryptDecrypter) Encrypt(plaintext string) string {
	bytes := []byte(plaintext)

	return hex.EncodeToString(e.AES256GCMSIVCipher.Seal(bytes[:0], e.Nonce, bytes, nil))
}

func Decrypt(val string) (string, error) {
	return ed.Decrypt(val) //nolint:wrapcheck // We proxy it.
}

func (e *encryptDecrypter) Decrypt(val string) (string, error) {
	decodedCiphertext, err := hex.DecodeString(val)
	if err != nil {
		return "", multierror.Append(errHexDecodingFailed, errors.Wrap(err, "failed to decode value"))
	}

	plaintext, err := e.AES256GCMSIVCipher.Open(decodedCiphertext[:0], e.Nonce, decodedCiphertext, nil)
	if err != nil {
		return "", multierror.Append(errDecryptionFailed, errors.Wrap(err, "failed to Open ciphertext"))
	}

	return string(plaintext), nil
}
