// SPDX-License-Identifier: ice License 1.0

package internal

func (t *Token) IsIce() bool {
	return t.Provider == ProviderIce
}

func (t *Token) IsFirebase() bool {
	return t.Provider == ProviderFirebase
}
