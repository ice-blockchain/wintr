// SPDX-License-Identifier: BUSL-1.1

package storagefixture

import (
	_ "embed"

	"github.com/framey-io/go-tarantool"

	connectorsfixture "github.com/ice-blockchain/wintr/connectors/fixture"
	"github.com/ice-blockchain/wintr/connectors/storage"
)

// Public API.

type (
	TestConnector interface {
		connectorsfixture.TestConnector

		tarantool.Connector
	}
)

// Private API.

const (
	scriptName = "init.lua"
	fileMode   = 0o777
)

//go:embed .testdata/docker-compose.yaml
var dockerComposeYAMLTemplate string

//go:embed .testdata/init.lua
var dbStartupScriptTemplate string

type (
	dbCfg struct {
		SchemaPath       string `yaml:"schemaPath"`
		storage.DBConfig `mapstructure:",squash"`
	}
	cfg struct {
		DB dbCfg `yaml:"db"`
	}
	testConnector struct {
		delegate connectorsfixture.TestConnector
		tarantool.Connector
		cfg                *cfg
		applicationYamlKey string
		order              int
	}
)
