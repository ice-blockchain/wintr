// SPDX-License-Identifier: BUSL-1.1

package serverfixture

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/server"
)

const testDIDToken = "WyIweGFhNTBiZTcwNzI5Y2E3MDViYTdjOGQwMDE4NWM2ZjJkYTQ3OWQwZm" +
	"NkZTUzMTFjYTRjZTViMWJhNzE1YzhhNzIxYzVmMTk0ODQzNGY5NmZmNTc3ZDdiMmI2YWQ4MmQ" +
	"zZGQ1YTI0NTdmZTY5OThiMTM3ZWQ5YmMwOGQzNmU1NDljMWIiLCJ7XCJpYXRcIjoxNTg2NzY0" +
	"MjcwLFwiZXh0XCI6MTExNzM1Mjg1MDAsXCJpc3NcIjpcImRpZDpldGhyOjB4NEI3M0M1ODM3M" +
	"EFFZmNFZjg2QTYwMjFhZkNEZTU2NzM1MTEzNzZCMlwiLFwic3ViXCI6XCJOanJBNTNTY1E4SV" +
	"Y4ME5Kbng0dDNTaGk5LWtGZkY1cWF2RDJWcjBkMWRjPVwiLFwiYXVkXCI6XCJkaWQ6bWFnaWM" +
	"6NzMxODQ4Y2MtMDg0ZS00MWZmLWJiZGYtN2YxMDM4MTdlYTZiXCIsXCJuYmZcIjoxNTg2NzY0" +
	"MjcwLFwidGlkXCI6XCJlYmNjODgwYS1mZmM5LTQzNzUtODRhZS0xNTRjY2Q1Yzc0NmRcIixcI" +
	"mFkZFwiOlwiMHg4NGQ2ODM5MjY4YTFhZjkxMTFmZGVjY2QzOTZmMzAzODA1ZGNhMmJjMDM0NT" +
	"BiN2ViMTE2ZTJmNWZjOGM1YTcyMmQxZmI5YWYyMzNhYTczYzVjMTcwODM5Y2U1YWQ4MTQxYjl" +
	"iNDY0MzM4MDk4MmRhNGJmYmIwYjExMjg0OTg4ZjFiXCJ9Il0="

func GenerateMagicToken() string {
	return testDIDToken
}

func StartContainer(ctx context.Context, serviceName string, testCfg server.Config) (func(), string, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: selfContainerRequest(serviceName, testCfg),
		Started:          true,
	})
	if err != nil {
		return func() {}, "", errors.Wrap(err, "container failed to start")
	}

	ip, err := container.Host(ctx)
	if err != nil {
		return terminateContainer(ctx, container), "", errors.Wrapf(err, "could not get host for %v container", serviceName)
	}

	port := fmt.Sprintf("%v/tcp", testCfg.HTTPServer.Port)
	mappedPort, err := container.MappedPort(ctx, nat.Port(port))
	if err != nil {
		return terminateContainer(ctx, container), "", errors.Wrapf(err, "could not get port for %v container", serviceName)
	}

	serverAddr := fmt.Sprintf("%s:%s", ip, mappedPort.Port())

	return terminateContainer(ctx, container), serverAddr, nil
}

func terminateContainer(ctx context.Context, container testcontainers.Container) func() {
	return func() {
		c := ctx
		if c.Err() != nil {
			cc, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			c = cc
			defer cancel()
		}
		if err := container.Terminate(c); err != nil {
			log.Fatal(errors.Wrap(err, "ricky container failed to terminate"))
		}
	}
}

//nolint:funlen // Because the alternative is worse
func selfContainerRequest(serviceName string, testCfg server.Config) testcontainers.ContainerRequest {
	var (
		_os    = "linux"
		goarch = runtime.GOARCH
	)
	dockerFileContext, testdataPath, port := containerInfo(testCfg)

	return testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       dockerFileContext,
			Dockerfile:    fmt.Sprintf("cmd%c%v%cDockerfile", os.PathSeparator, serviceName, os.PathSeparator),
			PrintBuildLog: true,
			BuildArgs:     map[string]*string{"SERVICE_NAME": &serviceName, "TARGETOS": &_os, "TARGETARCH": &goarch, "PORT": &port},
		},
		Labels: map[string]string{"os": _os, "arch": goarch},
		Mounts: testcontainers.Mounts(
			testcontainers.BindMount(
				fmt.Sprintf("%v.testdata/localhost.crt", testdataPath),
				testcontainers.ContainerMountTarget(fmt.Sprintf("/%v", testCfg.HTTPServer.CertPath)),
			),
			testcontainers.BindMount(
				fmt.Sprintf("%v.testdata/localhost.key", testdataPath),
				testcontainers.ContainerMountTarget(fmt.Sprintf("/%v", testCfg.HTTPServer.KeyPath)),
			),
			testcontainers.BindMount(
				fmt.Sprintf("%v.testdata/application.yaml", testdataPath),
				"/application.yaml",
			),
		),
		AutoRemove:   true,
		NetworkMode:  "host",
		Name:         uuid.New().String(),
		ExposedPorts: []string{fmt.Sprintf("%v/tcp", port)},
		WaitingFor: wait.ForAll(
			wait.
				ForLog(fmt.Sprintf("server started listening on %v...", port)).WithStartupTimeout(10*time.Minute),
			wait.
				ForHTTP("/health-check").WithStartupTimeout(10*time.Minute).
				WithPort(nat.Port(fmt.Sprintf("%v/tcp", port))).
				WithTLS(true, LocalhostTLS(testCfg)),
		).WithStartupTimeout(10 * time.Minute),
	}
}

func containerInfo(testCfg server.Config) (dockerFileContext, testdataPath, port string) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not get working dir"))
	}

	if strings.HasSuffix(wd, fmt.Sprintf("cmd%cricky", os.PathSeparator)) {
		dockerFileContext = path.Join(wd, "..", "..")
		testdataPath = fmt.Sprintf("%v%c", wd, os.PathSeparator)
	} else {
		dockerFileContext = "."
		testdataPath = fmt.Sprintf("%v%ccmd%cricky%c", wd, os.PathSeparator, os.PathSeparator, os.PathSeparator)
	}

	port = fmt.Sprintf("%v", testCfg.HTTPServer.Port)

	return
}

func LocalhostTLS(testCfg server.Config) *tls.Config {
	caCertPool := x509.NewCertPool()
	if caCert, err := os.ReadFile(testCfg.HTTPServer.CertPath); err != nil {
		log.Fatal(errors.Wrapf(err, "Reading server certificate %v", testCfg.HTTPServer.CertPath))
	} else {
		caCertPool.AppendCertsFromPEM(caCert)
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    caCertPool,
	}
}
