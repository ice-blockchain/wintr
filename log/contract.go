// SPDX-License-Identifier: BUSL-1.1

package log

import (
	"github.com/rs/zerolog"
)

// Private API.

const stackFramesToSkip = 2

// nolint:gochecknoglobals // we need only one log for the app, hence it is global
var logger *zerolog.Logger

type cfg struct {
	Encoder string `yaml:"encoder"`
	Level   string `yaml:"level"`
}
