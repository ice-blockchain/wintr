// SPDX-License-Identifier: BUSL-1.1

package terror

type (
	Err struct {
		error
		Data map[string]interface{} `json:"data"`
	}
)
