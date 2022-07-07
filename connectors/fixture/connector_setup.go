// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ice-blockchain/wintr/log"
)

func NewConnector(
	name, dockerComposeYAMLTemplate, waitForLog string, order int,
	findPort func(applicationYamlKey string) (port int, ssl bool, err error),
	createAdditionalFiles func(port int, tmpFolder string) error,
) TestConnector {
	return &testConnector{
		order:                     order,
		name:                      name,
		dockerComposeYAMLTemplate: dockerComposeYAMLTemplate,
		waitForLog:                waitForLog,
		findPort:                  findPort,
		createAdditionalFiles:     createAdditionalFiles,
	}
}

func (c *testConnector) Order() int {
	return c.order
}

func (c *testConnector) Setup(ctx context.Context) ContextErrClose {
	containerID := strings.ToLower(uuid.New().String())
	tmpFolder := fmt.Sprintf(".tmp-%s", containerID)
	applicationYamlKey, ok := ctx.Value(applicationYAMLKeyContextValueKey).(string)
	if !ok {
		log.Panic("no package name provided in context")
	}
	defer func() {
		if e := recover(); e != nil {
			log.Error(cleanUpTMPFolder(applicationYamlKey, tmpFolder))
			log.Panic(e)
		}
	}()

	log.Info(fmt.Sprintf("starting `%v` test environment docker compose for %v...", applicationYamlKey, c.name), "containerID", containerID)
	dockerCompose := c.startDockerCompose(tmpFolder, applicationYamlKey, containerID)

	return func(context.Context) error {
		log.Info(fmt.Sprintf("stopping `%v` test environment docker compose for %v...", applicationYamlKey, c.name), "containerID", containerID)
		defer func() {
			if e := recover(); e != nil {
				log.Error(cleanUpTMPFolder(applicationYamlKey, tmpFolder))
				log.Panic(e)
			}
			log.Error(cleanUpTMPFolder(applicationYamlKey, tmpFolder))
		}()

		return errors.Wrapf(dockerCompose.Down().Error, "failed to stop & clean `%v` docker compose for the `%v` test environment", c.name, applicationYamlKey)
	}
}

func (c *testConnector) startDockerCompose(tmpFolder, applicationYamlKey, containerID string) *testcontainers.LocalDockerCompose {
	var err error
	c.port, c.ssl, err = c.findPort(applicationYamlKey)
	log.Panic(errors.Wrapf(err, "could not find `%v` port for `%v`", applicationYamlKey, c.name))

	paths, err := c.createRequiredTestEnvFiles(tmpFolder, applicationYamlKey)
	log.Panic(errors.Wrapf(err, "`%v` failed to createRequiredTestEnvFiles for `%v` test environment", c.name, applicationYamlKey))

	dockerCompose := testcontainers.NewLocalDockerCompose(paths, containerID)
	dockerCompose.WithExposedService(fmt.Sprintf("%v%v", c.name, c.port), c.port, wait.ForLog(c.waitForLog))
	dockerCompose.Env = map[string]string{"COMPOSE_COMPATIBILITY": "true"}
	dockerCompose.WithCommand([]string{"up", "-d"})

	log.Panic(errors.Wrapf(dockerCompose.Invoke().Error, "failed to start `%v` docker compose for `%v` test environment", c.name, applicationYamlKey))

	return dockerCompose
}

func (c *testConnector) createRequiredTestEnvFiles(tmpFolder, applicationYamlKey string) ([]string, error) {
	if err := os.Mkdir(tmpFolder, fileMode); err != nil {
		return nil, errors.Wrapf(err, "failed to create .tmp folder for `%v` test environment", applicationYamlKey)
	}
	dbDockerComposeYAMLName := fmt.Sprintf("%s/%s", tmpFolder, dockerComposeName)
	dbDockerComposeYAML := fmt.Sprintf(c.dockerComposeYAMLTemplate, c.port, c.ssl)
	if err := os.WriteFile(dbDockerComposeYAMLName, []byte(dbDockerComposeYAML), fileMode); err != nil {
		return nil, errors.Wrapf(err, "failed to create tmp .yaml docker compose for `%v` test environment", applicationYamlKey)
	}
	if err := os.WriteFile(fmt.Sprintf("%s/%s", tmpFolder, crtName), []byte(localhostCrt), fileMode); err != nil {
		return nil, errors.Wrapf(err, "failed to create tmp `%v` docker compose for `%v` test environment", crtName, applicationYamlKey)
	}
	if err := os.WriteFile(fmt.Sprintf("%s/%s", tmpFolder, keyName), []byte(localhostKey), fileMode); err != nil {
		return nil, errors.Wrapf(err, "failed to create tmp `%v` docker compose for `%v` test environment", keyName, applicationYamlKey)
	}
	if c.createAdditionalFiles != nil {
		if err := c.createAdditionalFiles(c.port, tmpFolder); err != nil {
			return nil, errors.Wrapf(err, "failed to create additional files `%v` test environment", applicationYamlKey)
		}
	}

	return []string{dbDockerComposeYAMLName}, nil
}

func cleanUpTMPFolder(applicationYamlKey, tmpFolder string) error {
	return errors.Wrapf(os.RemoveAll(tmpFolder), "failed to clean `%v` .tmp files for `%v` test environment", tmpFolder, applicationYamlKey)
}
