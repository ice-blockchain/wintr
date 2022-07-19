// SPDX-License-Identifier: BUSL-1.1

package storage

import (
	"time"

	"github.com/pkg/errors"
)

// Public API.

const (
	IndexName = "indexName"
)

var (
	ErrNotFound                 = errors.New("not found")
	ErrRelationNotFound         = errors.New("relation not found")
	ErrDuplicate                = errors.New("duplicate")
	ErrRetryOnInvalidForeignKey = errors.New("unexpected error when inserting or updating a entry with an invalid foreign key reference")
	ErrDDLInvalid               = errors.New("DDL is invalid")
	ErrNoSpacesConfigured       = errors.New("no spaces configured")
)

type (
	DBConfig struct {
		User               string   `yaml:"user"`
		Password           string   `yaml:"password"`
		URLs               []string `yaml:"urls"` //nolint:tagliatelle // Nope.
		Spaces             []string `yaml:"spaces"`
		ReadOnly           bool     `yaml:"readOnly"`
		SkipSchemaCreation bool     `yaml:"skipSchemaCreation"`
	}
	Config struct {
		DB DBConfig `yaml:"db"`
	}
)

// Private API.

const (
	dbSchemaInitDeadline         = 30 * time.Second
	getAllUserSpacesFunctionName = "get_all_user_spaces"
)

//
var (
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	cfg Config
)
