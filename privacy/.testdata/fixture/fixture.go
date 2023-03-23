// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/ericlagergren/siv"

	"github.com/ice-blockchain/wintr/log"
)

func GenerateSecret() string {
	key := make([]byte, 32+siv.NonceSize) //nolint:gomnd // 32 is the byte size, nothing magical about it.
	_, err := io.ReadFull(rand.Reader, key)
	log.Panic(err)

	return hex.EncodeToString(key)
}
