// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"context"
	_ "embed"
	"io"
	"net/http"
	"testing"

	"github.com/testcontainers/testcontainers-go"

	connectorsfixture "github.com/ice-blockchain/wintr/connectors/fixture"
	"github.com/ice-blockchain/wintr/server"
)

// Public API.

type (
	RespStatusCode   = int
	ReqBody          = io.Reader
	URL              = string
	ExpectedRespBody = string
	ActualRespBody   = string
	ContentType      = string

	TestConnector interface {
		connectorsfixture.TestConnector

		WrapJSONBody(jsonData string) (ReqBody, ContentType)
		WrapMultipartBody(tb testing.TB, values map[string]interface{}) (ReqBody, ContentType)

		TestSwagger(context.Context, testing.TB)
		TestHealthCheck(context.Context, testing.TB)

		AssertUnauthorized(testing.TB, ExpectedRespBody, ActualRespBody, RespStatusCode, http.Header)

		Get(context.Context, testing.TB, URL, ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Delete(context.Context, testing.TB, URL, ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Patch(context.Context, testing.TB, URL, ReqBody, ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Put(context.Context, testing.TB, URL, ReqBody, ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Post(context.Context, testing.TB, URL, ReqBody, ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
	}
)

// Private API.

const (
	crtName  = "localhost.crt"
	keyName  = "localhost.key"
	fileMode = 0o777
)

//go:embed .testdata/localhost.crt
var localhostCrt string

//go:embed .testdata/localhost.key
var localhostKey string

type (
	testConnector struct {
		*httpTestClient
		main                      func()
		cfg                       *server.Config
		logConsumer               *containerLogConsumer
		applicationYAMLKey        string
		swaggerRoot               string
		expectedSwaggerJSON       string
		containerID               string
		serviceName               string
		tmpFolder                 string
		dockerFileContext         string
		testdataPath              string
		projectRoot               string
		serverAddr                string
		additionalContainerMounts []func(projectRoot string) testcontainers.ContainerMount
		order                     int
	}
	httpTestClient struct {
		client              *http.Client
		swaggerRoot         string
		expectedSwaggerJSON string
		serverAddr          string
	}
	containerLogConsumer struct{}
)
