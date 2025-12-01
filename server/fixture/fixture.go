// SPDX-License-Identifier: ice License 1.0

//go:build test

package fixture

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/net/http2"

	appcfg "github.com/ice-blockchain/wintr/config"
	connectorsfixture "github.com/ice-blockchain/wintr/connectors/fixture"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/server"
)

//nolint:revive // Need, a lot simpler to use and inline.
func NewTestConnector(
	applicationYAMLKey, serviceDir, swaggerRoot, expectedSwaggerJSON string, order int,
	additionalContainerMounts ...func(projectRoot string) testcontainers.ContainerMount,
) TestConnector {
	var cfg server.Config
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	return &testConnector{
		cfg:                       &cfg,
		serviceName:               strings.ReplaceAll(applicationYAMLKey, "cmd/", ""),
		serviceDir:                serviceDir,
		applicationYAMLKey:        applicationYAMLKey,
		swaggerRoot:               swaggerRoot,
		expectedSwaggerJSON:       expectedSwaggerJSON,
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

	tc.setupContainerRequiredFileSystem()

	if runtime.GOOS == "darwin" { // Because it is an issue with macOS with hostNetwork and with inter container communication.
		return tc.callMain(ctx)
	}

	return tc.startContainer(ctx)
}

func (tc *testConnector) callMain(ctx context.Context) connectorsfixture.ContextErrClose {
	cmd := exec.CommandContext(ctx, "go", "run", "-v", fmt.Sprintf("..%c%v", os.PathSeparator, tc.serviceName)) //nolint:gosec // False negative.
	cmd.Dir = tc.testdataPath
	cmd.Stdout = tc
	cmd.Stderr = tc
	log.Panic(errors.Wrapf(cmd.Start(), "failed start service %v", tc.serviceName)) //nolint:revive // That's the point.
	go func() {
		log.Error(errors.Wrapf(cmd.Wait(), "failed to wait for service start to finish for %v", tc.serviceName))
	}()
	startCtx, cancel := context.WithTimeout(ctx, 30*time.Second) //nolint:mnd,gomnd // Nothing magical about it.
	defer cancel()
	for !tc.started && startCtx.Err() == nil &&
		!strings.Contains(cmd.ProcessState.String(), "exit") && !strings.Contains(cmd.ProcessState.String(), "signal") { //nolint:revive // Blocking call.
	}
	if startCtx.Err() != nil || strings.Contains(cmd.ProcessState.String(), "exit") || strings.Contains(cmd.ProcessState.String(), "signal") {
		log.Panic("cmd ended unexpectedly")
	}
	tc.serverAddr = fmt.Sprintf("localhost:%v", tc.cfg.HTTPServer.Port)

	return func(context.Context) error {
		var errs []error
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			errs = append(errs,
				errors.Wrapf(err, "failed to sigterm main for %v", tc.serviceName),
				errors.Wrapf(cmd.Process.Kill(), "failed to kill main for %v, after sigterm failed", tc.serviceName))
		}
		errs = append(errs, errors.Wrapf(os.RemoveAll(tc.tmpFolder),
			"failed to clean `%v` .tmp files for `%v` test environment", tc.tmpFolder, tc.applicationYAMLKey))

		return errors.Wrapf(multierror.Append(nil, errs...).ErrorOrNil(), "failed to cleanup after main func")
	}
}

func (tc *testConnector) startContainer(ctx context.Context) (cleanUp connectorsfixture.ContextErrClose) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: tc.buildContainerRequest(ctx),
		Logger:           stdlog.Default(),
	})
	log.Panic(errors.Wrapf(err, "failed to build %v container", tc.serviceName)) //nolint:revive // That's the point.
	container.FollowOutput(tc.logConsumer)
	log.Panic(errors.Wrapf(container.StartLogProducer(ctx), "failed to start log producer for %v container", tc.serviceName))
	log.Panic(errors.Wrapf(container.Start(ctx), "failed to start %v container", tc.serviceName))
	defer func() {
		cleanUp = func(ctx context.Context) error {
			//nolint:revive // Wrong
			return errors.Wrapf(multierror.Append(nil,
				errors.Wrapf(container.StopLogProducer(), "%v[%v] failed to stop consuming logs for container", tc.serviceName, container.GetContainerID()),
				errors.Wrapf(container.Terminate(ctx), "%v[%v] container failed to terminate", tc.serviceName, container.GetContainerID()),
				errors.Wrapf(os.RemoveAll(tc.tmpFolder), "failed to clean `%v` .tmp files for `%v` test environment", tc.tmpFolder, tc.applicationYAMLKey),
			).ErrorOrNil(), "failed to cleanup container")
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

	return cleanUp
}

func (tc *testConnector) buildContainerRequest(ctx context.Context) testcontainers.ContainerRequest {
	var (
		osName = "linux"
		goarch = runtime.GOARCH
	)
	port := strconv.FormatUint(uint64(tc.cfg.HTTPServer.Port), 10)

	return testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       tc.dockerFileContext,
			Dockerfile:    tc.dockerFilePath,
			PrintBuildLog: true,
			BuildArgs:     map[string]*string{"SERVICE_NAME": &tc.serviceName, "TARGETOS": &osName, "TARGETARCH": &goarch, "PORT": &port},
		},
		Labels:       map[string]string{"os": osName, "arch": goarch},
		Mounts:       tc.mounts(),
		AutoRemove:   true,
		NetworkMode:  "host",
		Name:         tc.containerID,
		ExposedPorts: []string{fmt.Sprintf("%v/tcp", port)},
		WaitingFor:   tc.waitFor(ctx),
	}
}

func (tc *testConnector) mounts() testcontainers.ContainerMounts {
	dotEnvPath := fmt.Sprintf(`%v.env`, tc.projectRoot)
	// We create an empty .env file because otherwise container will not start if a mount is missing.
	if _, err := os.Stat(dotEnvPath); err != nil && errors.Is(err, os.ErrNotExist) {
		emptyDotEnvFile, cErr := os.Create(dotEnvPath)
		if cErr != nil {
			log.Panic(err)
		}
		log.Panic(emptyDotEnvFile.Close())
	}
	mounts := testcontainers.ContainerMounts{
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
		testcontainers.BindMount(dotEnvPath, `/.env`),
	}
	for i := range tc.additionalContainerMounts {
		mounts = append(mounts, tc.additionalContainerMounts[i](tc.projectRoot))
	}

	return mounts
}

func (tc *testConnector) waitFor(ctx context.Context) *wait.MultiStrategy {
	deadline, _ := ctx.Deadline()
	timeout := deadline.Sub(time.Now().UTC())

	return wait.
		ForAll(
			wait.
				ForLog(fmt.Sprintf("server started listening on %v...", tc.cfg.HTTPServer.Port)).
				WithStartupTimeout(timeout),
			wait.
				ForHTTP("/health-check").
				WithPort(nat.Port(fmt.Sprintf("%v/tcp", tc.cfg.HTTPServer.Port))).
				WithTLS(true, tc.localhostTLS()).
				WithStartupTimeout(timeout),
		).WithDeadline(timeout)
}

func (tc *testConnector) setupContainerRequiredFileSystem() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not get working dir"))
	}

	tc.dockerFilePath = fmt.Sprintf("cmd%c%v%cDockerfile", os.PathSeparator, tc.serviceName, os.PathSeparator)
	if strings.HasSuffix(wd, fmt.Sprintf("cmd%c%v", os.PathSeparator, tc.serviceDir)) {
		tc.dockerFileContext = path.Join(wd, "..", "..")
		tc.projectRoot = fmt.Sprintf("%v%c", path.Join(wd, "..", ".."), os.PathSeparator)
		tc.testdataPath = fmt.Sprintf("%v%c", wd, os.PathSeparator)
	} else {
		tc.dockerFileContext = "."
		tc.projectRoot = fmt.Sprintf("%v%c", wd, os.PathSeparator)
		tc.testdataPath = fmt.Sprintf("%v%ccmd%c%v%c", wd, os.PathSeparator, os.PathSeparator, tc.serviceDir, os.PathSeparator)
	}

	tc.containerID = strings.ToLower(uuid.New().String())
	tc.tmpFolder = fmt.Sprintf("%v/.tmp-%s", tc.testdataPath, tc.containerID)
	//nolint:revive // That's the point.
	log.Panic(errors.Wrapf(os.Mkdir(tc.tmpFolder, fileMode), "failed to create .tmp folder for `%v` test environment", tc.applicationYAMLKey))
	log.Panic(errors.Wrapf(os.WriteFile(fmt.Sprintf("%s/%s", tc.tmpFolder, crtName), []byte(localhostCrt), fileMode),
		"failed to create tmp `%v` docker compose for `%v` test environment", crtName, tc.applicationYAMLKey))
	log.Panic(errors.Wrapf(os.WriteFile(fmt.Sprintf("%s/%s", tc.tmpFolder, keyName), []byte(localhostKey), fileMode),
		"failed to create tmp `%v` docker compose for `%v` test environment", keyName, tc.applicationYAMLKey))
}

func (*testConnector) localhostTLS() *tls.Config {
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM([]byte(localhostCrt)); !ok {
		log.Panic(errors.New("failed to append localhost tls to cert pool"))
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    caCertPool,
	}
}

func (*containerLogConsumer) Accept(logMsg testcontainers.Log) { //nolint:gocritic // It's external API, can't control it.
	switch logMsg.LogType {
	case testcontainers.StdoutLog:
		log.Info(string(logMsg.Content))
	case testcontainers.StderrLog:
		log.Error(errors.New(string(logMsg.Content)))
	default:
		log.Panic(errors.Errorf("unexpected logType %v", logMsg.LogType))
	}
}

func (tc *testConnector) Write(p []byte) (n int, err error) {
	if strings.Contains(string(p), fmt.Sprintf("server started listening on %v...", tc.cfg.HTTPServer.Port)) {
		tc.started = true
	}

	return stdlog.Writer().Write(p) //nolint:wrapcheck // It's a proxy.
}
