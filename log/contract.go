// SPDX-License-Identifier: ice License 1.0

package log

// Private API.

type (
	cfg struct {
		Encoder string `yaml:"encoder"`
		Level   string `yaml:"level"`
	}
)
