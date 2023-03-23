// SPDX-License-Identifier: ice License 1.0

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

	HTTPTestClient interface {
		Get(context.Context, testing.TB, URL, ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Delete(context.Context, testing.TB, URL, ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Patch(context.Context, testing.TB, URL, ReqBody, ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Put(context.Context, testing.TB, URL, ReqBody, ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Post(context.Context, testing.TB, URL, ReqBody, ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
	}
	TestConnector interface {
		connectorsfixture.TestConnector
		HTTPTestClient

		WrapJSONBody(jsonData string) (ReqBody, ContentType)
		WrapMultipartBody(tb testing.TB, values map[string]any) (ReqBody, ContentType)

		TestSwagger(context.Context, testing.TB)
		TestHealthCheck(context.Context, testing.TB)

		AssertUnauthorized(testing.TB, ExpectedRespBody, ActualRespBody, RespStatusCode, http.Header)
	}
)

// Private API.

const (
	jsonContentType = "application/json"
	crtName         = "localhost.crt"
	keyName         = "localhost.key"
	fileMode        = 0o777
)

var (
	//go:embed .testdata/localhost.crt
	localhostCrt string
	//go:embed .testdata/localhost.key
	localhostKey string
)

type (
	testConnector struct {
		*httpTestClient
		cfg                       *server.Config
		logConsumer               *containerLogConsumer
		applicationYAMLKey        string
		swaggerRoot               string
		expectedSwaggerJSON       string
		containerID               string
		serviceName               string
		serviceDir                string
		tmpFolder                 string
		dockerFileContext         string
		dockerFilePath            string
		testdataPath              string
		projectRoot               string
		serverAddr                string
		additionalContainerMounts []func(projectRoot string) testcontainers.ContainerMount
		order                     int
		started                   bool
	}
	httpTestClient struct {
		client              *http.Client
		swaggerRoot         string
		expectedSwaggerJSON string
		serverAddr          string
	}
	containerLogConsumer struct{}
)
