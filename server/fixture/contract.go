// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"time"

	"github.com/testcontainers/testcontainers-go"
)

const (
	contextDeadline = 60 * time.Second
	startUpTimeout  = 10 * time.Minute
)

type Mounts = testcontainers.ContainerMount
