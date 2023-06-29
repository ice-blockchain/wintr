// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"strings"
)

func (t *Token) IsIce() bool {
	return t.Provider == AccessJwtIssuer || t.Provider == RefreshJwtIssuer
}

func (t *Token) IsFirebase() bool {
	return strings.HasPrefix(t.Provider, "https://securetoken.google.com")
}
