// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"context"
	_ "embed"
	"testing"
)

// Public API.

type (
	ContextErrClose = func(context.Context) error

	TestRunner interface {
		RunTests(*testing.M)
		StartConnectorsIndefinitely()
	}
	TestConnector interface {
		Setup(context.Context) ContextErrClose
		Order() int
	}
	ConnectorLifecycleHooks struct {
		AfterConnectorsStarted  func(context.Context) ContextErrClose
		BeforeConnectorsStarted func(context.Context) ContextErrClose

		AfterConnectorsStopped  func(context.Context) ContextErrClose
		BeforeConnectorsStopped func(context.Context) ContextErrClose
	}
)

// Private API.

const (
	applicationYAMLKeyContextValueKey = "applicationYAMLKey"
	dockerComposeName                 = "docker-compose.yaml"
	crtName                           = "localhost.crt"
	keyName                           = "localhost.key"
	fileMode                          = 0o777
)

var (
	//go:embed .testdata/localhost.crt
	localhostCrt string
	//go:embed .testdata/localhost.key
	localhostKey string
)

type (
	orderedCleanUp struct {
		cleanUp ContextErrClose
		order   int
	}
	testRunner struct {
		orderedTestConnectors    map[int][]TestConnector
		orderedConnectorCleanUps map[int][]ContextErrClose
		*ConnectorLifecycleHooks
		applicationYAMLKey string
		orderedSequence    []int
		testConnectorCount int
	}
	testConnector struct {
		createAdditionalFiles     func(port int, tmpFolder string) error
		findPort                  func() (port int, ssl bool, err error)
		name                      string
		waitForLog                string
		dockerComposeYAMLTemplate string
		port                      int
		order                     int
		ssl                       bool
	}
)
