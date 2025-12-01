// SPDX-License-Identifier: ice License 1.0

//go:build test

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
		Get(ctx context.Context, tb testing.TB, u URL, headers ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Delete(ctx context.Context, tb testing.TB, u URL, headers ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Patch(ctx context.Context, tb testing.TB, u URL, body ReqBody, headers ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Put(ctx context.Context, tb testing.TB, u URL, body ReqBody, headers ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
		Post(ctx context.Context, tb testing.TB, u URL, body ReqBody, headers ...http.Header) (ActualRespBody, RespStatusCode, http.Header)
	}
	TestConnector interface {
		connectorsfixture.TestConnector
		HTTPTestClient

		WrapJSONBody(jsonData string) (ReqBody, ContentType)
		WrapMultipartBody(tb testing.TB, values map[string]any) (ReqBody, ContentType)

		TestSwagger(ctx context.Context, tb testing.TB)
		TestHealthCheck(ctx context.Context, tb testing.TB)

		AssertUnauthorized(tb testing.TB, exp ExpectedRespBody, actual ActualRespBody, respCode RespStatusCode, headers http.Header)
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
