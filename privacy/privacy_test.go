// SPDX-License-Identifier: ice License 1.0

package privacy

import (
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/ice-blockchain/wintr/log"
)

func TestDeterministicEncryptDecrypt(t *testing.T) {
	t.Parallel()
	val := "bogus@foo.com" //nolint:goconst // Nope, it's not, it's just a test value, that we duplicate to improve readability/understanding.
	encrypted := Encrypt(val)
	decrypted, err := Decrypt(encrypted)
	log.Panic(err) //nolint:revive // That's exactly what we want. Everything fails if we have error there, thus we panic.
	assert.Equal(t, "e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc", encrypted)
	assert.Equal(t, val, decrypted)

	encrypted = Encrypt(val)
	decrypted, err = Decrypt(encrypted)
	log.Panic(err)
	assert.Equal(t, "e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc", encrypted)
	assert.Equal(t, val, decrypted)
}

func BenchmarkDynamicEncryptDecrypt(b *testing.B) {
	b.SetParallelism(100000)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			plaintext := strconv.Itoa(i) + "bogus@foo.com"
			encrypted := Encrypt(plaintext)
			decrypted, err := Decrypt(encrypted)
			log.Panic(err) //nolint:revive // That's exactly what we want. Everything fails if we have error there, thus we panic.
			if plaintext != decrypted {
				log.Panic(errors.Errorf("diff: %v, %v", plaintext, decrypted))
			}
		}
	})
}

func BenchmarkStaticDecrypt(b *testing.B) {
	b.SetParallelism(100000)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			decrypted, err := Decrypt("e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc")
			log.Panic(err) //nolint:revive // That's exactly what we want. Everything fails if we have error there, thus we panic.
			if decrypted != "bogus@foo.com" {
				log.Panic(errors.Errorf("diff: %v", decrypted))
			}
		}
	})
}

func BenchmarkDynamicEncrypt(b *testing.B) {
	b.SetParallelism(100000)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			_ = Encrypt(strconv.Itoa(i) + "bogus@foo.com")
		}
	})
}

func BenchmarkStaticEncrypt(b *testing.B) {
	b.SetParallelism(100000)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			d := Encrypt("bogus@foo.com")
			if d != "e95cf122b13b76295043ea46be61092d87671cb2dc6b8c397d482872bc" {
				log.Panic(errors.Errorf("diff: %v", d))
			}
		}
	})
}
