// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"context"
	_ "embed"
	"os"
	"testing"
)

// Public API.

type (
	ContextErrClose = func(context.Context) error
	SystemExitCode  = int

	TestRunner interface {
		RunTests(*testing.M) SystemExitCode
		StartConnectorsIndefinitely(chan os.Signal)
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

//go:embed .testdata/localhost.crt
var localhostCrt string

//go:embed .testdata/localhost.key
var localhostKey string

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
		findPort                  func(applicationYAMLKey string) (port int, ssl bool, err error)
		name                      string
		waitForLog                string
		dockerComposeYAMLTemplate string
		port                      int
		order                     int
		ssl                       bool
	}
)
