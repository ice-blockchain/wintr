// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"errors"
	"time"
)

// Private API.

var (
	errFixtureCleanUp = errors.New("fixture cleanup failed")
	errRecover        = errors.New("fixture recover is not empty")
)

const testsContextTimeout = 10 * time.Minute
