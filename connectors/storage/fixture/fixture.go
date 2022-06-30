// SPDX-License-Identifier: BUSL-1.1

package storagefixture

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

//go:embed .testdata/docker-compose.yaml
var dockerComposeYAMLTemplate string

//go:embed .testdata/init.lua
var dbStartupScriptTemplate string

const (
	dockerComposeName = "docker-compose.yaml"
	scriptName        = "init.lua"
	fileMode          = 0o777
)

func TestSetup(applicationYamlKey string) func() {
	containerID := strings.ToLower(uuid.New().String())
	tmpFolder := fmt.Sprintf(".tmp-%s", containerID)

	log.Info("starting test environment docker compose...", "containerID", containerID)
	dockerCompose := startDockerCompose(tmpFolder, applicationYamlKey, containerID)

	return func() {
		var es []error
		log.Info("stopping test environment docker compose...", "containerID", containerID)
		if dErr := dockerCompose.Down(); dErr.Error != nil {
			err := errors.Wrapf(dErr.Error, "failed to stop & clean docker compose for the `%v` test environment", applicationYamlKey)
			es = append(es, err)
		}
		es = removeTMPFolder(tmpFolder, applicationYamlKey, es)

		if len(es) != 0 {
			for _, e := range es {
				log.Error(e)
			}

			log.Panic(es[0])
		}
	}
}

func startDockerCompose(tmpFolder, applicationYamlKey, containerID string) *testcontainers.LocalDockerCompose {
	dbPort, err := findDBPort(applicationYamlKey)
	if err != nil {
		log.Panic(errors.Wrap(err, "could not find db port"))
	}
	dockerCompose := testcontainers.NewLocalDockerCompose(dbDockerComposeYAMLPaths(tmpFolder, applicationYamlKey), containerID)
	dockerCompose.WithExposedService(fmt.Sprintf("db%v", dbPort), dbPort, wait.ForLog("ready to accept requests"))
	dockerCompose.Env = map[string]string{"COMPOSE_COMPATIBILITY": "true"}
	dockerCompose.WithCommand([]string{"up", "-d"})

	if execErr := dockerCompose.Invoke(); execErr.Error != nil {
		es := []error{errors.Wrapf(execErr.Error, "failed to start docker compose for `%v` test environment", applicationYamlKey)}
		es = removeTMPFolder(tmpFolder, applicationYamlKey, es)

		for _, e := range es {
			log.Error(e)
		}

		log.Panic(es[0])
	}

	return dockerCompose
}

func dbDockerComposeYAMLPaths(tmpFolder, applicationYamlKey string) []string {
	paths, err := createRequiredTestEnvFiles(tmpFolder, applicationYamlKey)
	if err != nil {
		es := []error{errors.Wrapf(err, "failed to createRequiredTestEnvFiles for `%v` test environment", applicationYamlKey)}
		es = removeTMPFolder(tmpFolder, applicationYamlKey, es)
		for _, e := range es {
			log.Error(e)
		}

		log.Panic(es[0])
	}

	return paths
}

func removeTMPFolder(tmpFolder, applicationYamlKey string, es []error) []error {
	if dErr := os.RemoveAll(tmpFolder); dErr != nil {
		es = append(es, errors.Wrapf(dErr, "failed to clean .tmp files for `%v` test environment", applicationYamlKey))
	}

	return es
}

func createRequiredTestEnvFiles(tmpFolder, applicationYamlKey string) ([]string, error) {
	dbPort, err := findDBPort(applicationYamlKey)
	if err != nil {
		return nil, errors.Wrapf(err, "could not find `%v` port for DB", applicationYamlKey)
	}
	if err = os.Mkdir(tmpFolder, fileMode); err != nil {
		return nil, errors.Wrapf(err, "failed to create .tmp folder for `%v` test environment", applicationYamlKey)
	}
	startupScriptName := fmt.Sprintf("%s/%s", tmpFolder, scriptName)
	dbStartupScript := fmt.Sprintf(dbStartupScriptTemplate, dbPort)
	if err = os.WriteFile(startupScriptName, []byte(dbStartupScript), fileMode); err != nil {
		return nil, errors.Wrapf(err, "failed to create tmp db script for `%v` test environment", applicationYamlKey)
	}
	dbDockerComposeYAMLName := fmt.Sprintf("%s/%s", tmpFolder, dockerComposeName)
	dbDockerComposeYAML := fmt.Sprintf(dockerComposeYAMLTemplate, dbPort)
	if err = os.WriteFile(dbDockerComposeYAMLName, []byte(dbDockerComposeYAML), fileMode); err != nil {
		return nil, errors.Wrapf(err, "failed to create tmp .yaml docker compose for `%v` test environment", applicationYamlKey)
	}

	return []string{dbDockerComposeYAMLName}, nil
}

func findDBPort(applicationYamlKey string) (int, error) {
	var cfg struct {
		DB struct {
			User     string   `yaml:"user"`
			Password string   `yaml:"password"`
			URLs     []string `yaml:"urls"`
		} `yaml:"db"`
	}
	config.MustLoadFromKey(applicationYamlKey, &cfg)
	if len(cfg.DB.URLs) == 0 {
		return 0, errors.Errorf("invalid/missing application.yaml for `%v`", applicationYamlKey)
	}
	port, err := strconv.Atoi(strings.Split(cfg.DB.URLs[0], ":")[1])
	if err != nil {
		return 0, errors.Wrapf(err, "could not find a valid db port for `%v`", applicationYamlKey)
	}

	return port, nil
}
