// SPDX-License-Identifier: ice License 1.0

package terror

type (
	Err struct {
		error
		Data map[string]any `json:"data"`
	}
)
