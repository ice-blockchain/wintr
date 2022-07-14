// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/net/http2"

	appCfg "github.com/ice-blockchain/wintr/config"
	connectorsfixture "github.com/ice-blockchain/wintr/connectors/fixture"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/server"
)

func NewTestConnector(
	applicationYAMLKey, swaggerRoot, expectedSwaggerJSON string, order int, main func(),
	additionalContainerMounts ...func(projectRoot string) testcontainers.ContainerMount,
) TestConnector {
	var cfg server.Config
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	return &testConnector{
		cfg:                       &cfg,
		serviceName:               strings.ReplaceAll(applicationYAMLKey, "cmd/", ""),
		applicationYAMLKey:        applicationYAMLKey,
		swaggerRoot:               swaggerRoot,
		expectedSwaggerJSON:       expectedSwaggerJSON,
		main:                      main,
		order:                     order,
		additionalContainerMounts: additionalContainerMounts,
		logConsumer:               new(containerLogConsumer),
	}
}

func (tc *testConnector) Order() int {
	return tc.order
}

func (tc *testConnector) Setup(ctx context.Context) connectorsfixture.ContextErrClose {
	defer func() {
		tc.httpTestClient = &httpTestClient{
			serverAddr:          tc.serverAddr,
			swaggerRoot:         tc.swaggerRoot,
			expectedSwaggerJSON: tc.expectedSwaggerJSON,
			client:              &http.Client{Transport: &http2.Transport{TLSClientConfig: tc.localhostTLS()}},
		}
	}()

	if runtime.GOOS == "darwin" { // Because it is an issue with macOS with hostNetwork and with inter container communication.
		go tc.main()
		//nolint:gomnd // It's not a magic number, it's the sleep time.
		time.Sleep(20 * time.Second)
		tc.serverAddr = fmt.Sprintf("localhost:%v", tc.cfg.HTTPServer.Port)

		return func(context.Context) error {
			return nil
		}
	}

	return tc.startContainer(ctx)
}

func (tc *testConnector) startContainer(ctx context.Context) (cleanUp connectorsfixture.ContextErrClose) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: tc.buildContainerRequest(ctx),
		Logger:           stdlog.Default(),
	})
	log.Panic(errors.Wrapf(err, "failed to build %v container", tc.serviceName))
	container.FollowOutput(tc.logConsumer)
	log.Panic(errors.Wrapf(container.StartLogProducer(ctx), "failed to start log producer for %v container", tc.serviceName))
	log.Panic(errors.Wrapf(container.Start(ctx), "failed to start %v container", tc.serviceName))
	defer func() {
		cleanUp = func(ctx context.Context) error {
			return errors.Wrapf(multierror.Append(nil,
				errors.Wrapf(container.StopLogProducer(), "%v[%v] failed to stop consuming logs for container", tc.serviceName, container.GetContainerID()),
				errors.Wrapf(container.Terminate(ctx), "%v[%v] container failed to terminate", tc.serviceName, container.GetContainerID())).ErrorOrNil(),
				"failed to cleanup container")
		}
		if e := recover(); e != nil {
			log.Error(cleanUp(ctx))
			log.Panic(e)
		}
	}()
	ip, err := container.Host(ctx)
	log.Panic(errors.Wrapf(err, "failed to get %v container host", tc.serviceName))
	port := fmt.Sprintf("%v/tcp", tc.cfg.HTTPServer.Port)
	mappedPort, err := container.MappedPort(ctx, nat.Port(port))
	log.Panic(errors.Wrapf(err, "failed to get %v container mapped port", tc.serviceName))
	tc.serverAddr = fmt.Sprintf("%s:%s", ip, mappedPort.Port())

	return
}

func (tc *testConnector) buildContainerRequest(ctx context.Context) testcontainers.ContainerRequest {
	var (
		_os    = "linux"
		goarch = runtime.GOARCH
	)
	tc.setupContainerRequiredFileSystem()
	port := fmt.Sprintf("%v", tc.cfg.HTTPServer.Port)

	return testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       tc.dockerFileContext,
			Dockerfile:    fmt.Sprintf("cmd%c%v%cDockerfile", os.PathSeparator, tc.serviceName, os.PathSeparator),
			PrintBuildLog: true,
			BuildArgs:     map[string]*string{"SERVICE_NAME": &tc.serviceName, "TARGETOS": &_os, "TARGETARCH": &goarch, "PORT": &port},
		},
		Labels:       map[string]string{"os": _os, "arch": goarch},
		Mounts:       tc.mounts(),
		AutoRemove:   true,
		NetworkMode:  "host",
		Name:         tc.containerID,
		ExposedPorts: []string{fmt.Sprintf("%v/tcp", port)},
		WaitingFor:   tc.waitFor(ctx),
	}
}

func (tc *testConnector) mounts() testcontainers.ContainerMounts {
	m := testcontainers.ContainerMounts{
		testcontainers.BindMount(
			fmt.Sprintf("%v/localhost.crt", tc.tmpFolder),
			testcontainers.ContainerMountTarget(fmt.Sprintf("/%v", tc.cfg.HTTPServer.CertPath)),
		), testcontainers.BindMount(
			fmt.Sprintf("%v/localhost.key", tc.tmpFolder),
			testcontainers.ContainerMountTarget(fmt.Sprintf("/%v", tc.cfg.HTTPServer.KeyPath)),
		), testcontainers.BindMount(
			fmt.Sprintf("%v.testdata/application.yaml", tc.testdataPath),
			"/application.yaml",
		),
	}
	for i := range tc.additionalContainerMounts {
		m = append(m, tc.additionalContainerMounts[i](tc.projectRoot))
	}

	return m
}

func (tc *testConnector) waitFor(ctx context.Context) *wait.MultiStrategy {
	deadline, _ := ctx.Deadline()
	timeout := deadline.Sub(time.Now().UTC())

	return wait.ForAll(
		wait.
			ForLog(fmt.Sprintf("server started listening on %v...", tc.cfg.HTTPServer.Port)).
			WithStartupTimeout(timeout),
		wait.
			ForHTTP("/health-check").
			WithPort(nat.Port(fmt.Sprintf("%v/tcp", tc.cfg.HTTPServer.Port))).
			WithTLS(true, tc.localhostTLS()).
			WithStartupTimeout(timeout),
	).WithStartupTimeout(timeout)
}

func (tc *testConnector) setupContainerRequiredFileSystem() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not get working dir"))
	}

	if strings.HasSuffix(wd, fmt.Sprintf("cmd%c%v", os.PathSeparator, tc.serviceName)) {
		tc.dockerFileContext = path.Join(wd, "..", "..")
		tc.projectRoot = fmt.Sprintf("%v%c", path.Join(wd, "..", ".."), os.PathSeparator)
		tc.testdataPath = fmt.Sprintf("%v%c", wd, os.PathSeparator)
	} else {
		tc.dockerFileContext = "."
		tc.projectRoot = fmt.Sprintf("%v%c", wd, os.PathSeparator)
		tc.testdataPath = fmt.Sprintf("%v%ccmd%c%v%c", wd, os.PathSeparator, os.PathSeparator, tc.serviceName, os.PathSeparator)
	}

	tc.containerID = strings.ToLower(uuid.New().String())
	tc.tmpFolder = fmt.Sprintf("%v/.tmp-%s", tc.testdataPath, tc.containerID)
	log.Panic(errors.Wrapf(os.Mkdir(tc.tmpFolder, fileMode), "failed to create .tmp folder for `%v` test environment", tc.applicationYAMLKey))
	log.Panic(errors.Wrapf(os.WriteFile(fmt.Sprintf("%s/%s", tc.tmpFolder, crtName), []byte(localhostCrt), fileMode),
		"failed to create tmp `%v` docker compose for `%v` test environment", crtName, tc.applicationYAMLKey))
	log.Panic(errors.Wrapf(os.WriteFile(fmt.Sprintf("%s/%s", tc.tmpFolder, keyName), []byte(localhostKey), fileMode),
		"failed to create tmp `%v` docker compose for `%v` test environment", keyName, tc.applicationYAMLKey))
}

func (tc *testConnector) localhostTLS() *tls.Config {
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM([]byte(localhostCrt)); !ok {
		log.Panic(errors.New("failed to append localhost tls to cert pool"))
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    caCertPool,
	}
}

func (c *containerLogConsumer) Accept(logMsg testcontainers.Log) {
	switch logMsg.LogType {
	case testcontainers.StdoutLog:
		log.Info(string(logMsg.Content))
	case testcontainers.StderrLog:
		log.Error(errors.New(string(logMsg.Content)))
	default:
		log.Panic(errors.Errorf("unexpected logType %v", logMsg.LogType))
	}
}
